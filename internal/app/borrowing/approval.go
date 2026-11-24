package borrowing

import (
	"context"

	"SGEInventory_BE/domain/dto"
	"SGEInventory_BE/domain/entity"
	"SGEInventory_BE/domain/enums"
	"SGEInventory_BE/domain/errx"
)

// File ini berisi tahap 2: admin memproses pengajuan (ACC / tolak) + tolak otomatis.
//
// Aturan urutan kunci (mencegah deadlock): BARANG dulu (urut id), baru PENGAJUAN (urut id).
// Semua alur yang mengunci keduanya (ACC, edit total barang) mengikuti urutan ini, sehingga dua transaksi
// tidak pernah saling menunggu baris yang dipegang yang lain.

// ApproveBorrowing menyetujui pengajuan dalam SATU transaksi:
//
//  1. kunci barang-barangnya (urut id), lalu kunci pengajuannya
//  2. pastikan masih PENDING, barang masih ada, dan stok cukup (dicek ulang setelah terkunci)
//  3. kurangi stok, status → APPROVED, catat log APPROVED (pelaku = admin)
//  4. tolak otomatis pengajuan PENDING lain yang barangnya kini tidak cukup
//
// Jika salah satu langkah gagal, semuanya dibatalkan (stok tidak jadi berkurang).
func (s *borrowingService) ApproveBorrowing(ctx context.Context, id, adminID uint) (*dto.ApproveBorrowingResponse, error) {
	// Daftar barang di pengajuan tidak pernah berubah, jadi boleh dibaca tanpa kunci.
	borrowing, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err // errx.ErrBorrowingNotFound
	}
	if borrowing.Status != enums.BorrowingPending {
		return nil, errx.NewBorrowingAlreadyProcessed(borrowing.Status) // cek cepat sebelum mengunci apa pun
	}
	itemIDs := make([]uint, 0, len(borrowing.Items))
	for _, it := range borrowing.Items {
		itemIDs = append(itemIDs, it.ItemID)
	}

	var autoRejected []uint
	err = s.tx.WithinTransaction(ctx, func(ctx context.Context) error {
		// 1. Kunci barang dulu, baru pengajuan.
		items, err := s.itemRepo.FindByIDsForUpdate(ctx, itemIDs)
		if err != nil {
			return err
		}
		locked, err := s.repo.FindByIDForUpdate(ctx, id)
		if err != nil {
			return err
		}

		// 2. Status dicek ulang: mungkin sudah diproses admin lain / ditolak otomatis sejak cek cepat di atas.
		if err := locked.Approve(); err != nil {
			return err
		}
		lent, err := lendItems(borrowing.Items, items)
		if err != nil {
			return err
		}

		// 3. Simpan stok baru, status, dan log.
		for _, item := range lent {
			if err := s.itemRepo.Update(ctx, item); err != nil {
				return err
			}
		}
		if err := s.repo.UpdateStatus(ctx, locked); err != nil {
			return err
		}
		activity := entity.NewUserLog(adminID, enums.RequestTypeBorrowing, id, enums.ActivityApproved)
		if err := s.logRepo.Create(ctx, activity); err != nil {
			return err
		}

		// 4. Hanya barang di pengajuan ini yang stoknya berubah, jadi hanya itu yang perlu diperiksa.
		autoRejected, err = s.RejectUnfulfillablePending(ctx, itemIDs, adminID)
		return err
	})
	if err != nil {
		return nil, err
	}

	detail, err := s.GetBorrowing(ctx, id, adminID, true)
	if err != nil {
		return nil, err
	}
	return &dto.ApproveBorrowingResponse{Borrowing: *detail, AutoRejectedIDs: autoRejected}, nil
}

// RejectBorrowing menolak pengajuan: status → REJECTED + log REJECTED (pelaku = admin).
// Stok barang tidak disentuh, karena stok baru berkurang saat ACC.
func (s *borrowingService) RejectBorrowing(ctx context.Context, id, adminID uint) (*dto.BorrowingResponse, error) {
	err := s.tx.WithinTransaction(ctx, func(ctx context.Context) error {
		borrowing, err := s.repo.FindByIDForUpdate(ctx, id)
		if err != nil {
			return err // errx.ErrBorrowingNotFound
		}
		if err := borrowing.Reject(); err != nil {
			return err // bukan PENDING → 409
		}
		if err := s.repo.UpdateStatus(ctx, borrowing); err != nil {
			return err
		}
		activity := entity.NewUserLog(adminID, enums.RequestTypeBorrowing, id, enums.ActivityRejected)
		return s.logRepo.Create(ctx, activity)
	})
	if err != nil {
		return nil, err
	}
	return s.GetBorrowing(ctx, id, adminID, true)
}

// RejectUnfulfillablePending menolak otomatis pengajuan PENDING yang memuat salah satu itemIDs dengan
// jumlah melebihi stok tersedia saat ini. Dipanggil di dalam transaksi ACC atau edit barang (barang sudah
// terkunci), tapi juga aman dipanggil sendiri karena membuka transaksi jika belum ada.
// Setiap pengajuan yang ditolak mendapat log AUTO_REJECTED dengan pelaku actorID.
func (s *borrowingService) RejectUnfulfillablePending(ctx context.Context, itemIDs []uint, actorID uint) ([]uint, error) {
	rejected := []uint{} // bukan nil, supaya di JSON tampil [] (bukan null)
	if len(itemIDs) == 0 {
		return rejected, nil
	}

	err := s.tx.WithinTransaction(ctx, func(ctx context.Context) error {
		ids, err := s.repo.LockUnfulfillablePending(ctx, itemIDs)
		if err != nil || len(ids) == 0 {
			return err
		}
		if err := s.repo.UpdateStatuses(ctx, ids, enums.BorrowingRejected); err != nil {
			return err
		}

		logs := make([]entity.UserLog, 0, len(ids))
		for _, id := range ids {
			logs = append(logs, *entity.NewUserLog(actorID, enums.RequestTypeBorrowing, id, enums.ActivityAutoRejected))
		}
		if err := s.logRepo.CreateMany(ctx, logs); err != nil {
			return err
		}
		rejected = ids
		return nil
	})
	if err != nil {
		return nil, err
	}
	return rejected, nil
}

// lendItems mengurangi stok barang sesuai isi pengajuan. stock = barang yang sudah dikunci.
// Semua masalah dikumpulkan dulu supaya admin melihat semuanya sekaligus:
// barang yang sudah dihapus (409), lalu barang yang stoknya kurang (409).
// Hasil: barang yang stoknya sudah dikurangi (belum disimpan).
func lendItems(requested []entity.BorrowingItem, stock []entity.Item) ([]*entity.Item, error) {
	byID := make(map[uint]*entity.Item, len(stock))
	for i := range stock {
		byID[stock[i].ID] = &stock[i]
	}

	var deleted, shortages []string
	for _, it := range requested {
		item, ok := byID[it.ItemID]
		if !ok {
			deleted = append(deleted, it.ItemName) // nama dari salinan di borrowing_items
			continue
		}
		if !item.CanLend(it.Quantity) {
			shortages = append(shortages, entity.LendShortage(item.Name, it.Quantity, item.AvailableQuantity))
		}
	}
	if len(deleted) > 0 {
		return nil, errx.NewBorrowItemsUnavailable(deleted)
	}
	if len(shortages) > 0 {
		return nil, errx.NewInsufficientStock(shortages)
	}

	lent := make([]*entity.Item, 0, len(requested))
	for _, it := range requested {
		item := byID[it.ItemID]
		if err := item.Lend(it.Quantity); err != nil {
			return nil, err
		}
		lent = append(lent, item)
	}
	return lent, nil
}
