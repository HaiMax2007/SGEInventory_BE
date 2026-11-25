package borrowing

import (
	"context"
	"errors"
	"fmt"

	"SGEInventory_BE/domain/dto"
	"SGEInventory_BE/domain/entity"
	"SGEInventory_BE/domain/enums"
	"SGEInventory_BE/domain/errx"
	"SGEInventory_BE/pkg/photo"
)

// File ini berisi tahap 3: pengembalian.
//
//	user unggah foto  → RETURN_PENDING
//	admin ACC         → RETURNED (stok kembali, alur selesai)
//	admin tolak       → RETURN_REJECTED (foto dihapus, user unggah foto baru)
//
// Urutan kunci sama dengan approval.go: BARANG dulu (urut id), baru PENGAJUAN.

// returnPhotoFolder: folder key foto pengembalian, mis. "returns/8f3a...jpg".
const returnPhotoFolder = "returns"

// SubmitReturn menyimpan foto bukti pengembalian dari user pemilik pengajuan:
//
//  1. cek foto (ukuran & jenis dari isi file)
//  2. pengajuan harus milik user dan sedang dipinjam (APPROVED) atau fotonya baru ditolak (RETURN_REJECTED)
//  3. simpan file dengan nama acak, lalu dalam SATU transaksi: status RETURN_PENDING + path foto + log
//  4. balas data pengajuan TANPA timeline
//
// File disimpan sebelum transaksi (menulis file tidak bisa di-rollback). Jika transaksi gagal, file itu
// dihapus lagi supaya tidak ada file yatim.
func (s *borrowingService) SubmitReturn(ctx context.Context, id, userID uint, req dto.SubmitReturnRequest) (*dto.BorrowingResponse, error) {
	ext, err := photo.Extension(req.Photo)
	if errors.Is(err, photo.ErrInvalid) {
		return nil, errx.ErrInvalidReturnPhoto
	}
	if err != nil {
		return nil, err
	}

	borrowing, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err // errx.ErrBorrowingNotFound
	}
	// Pengajuan orang lain dibalas "tidak ditemukan", sama seperti GET /api/borrowings/:id.
	if borrowing.UserID != userID {
		return nil, errx.ErrBorrowingNotFound
	}
	// Cek cepat sebelum menyimpan file; dicek ulang setelah baris dikunci.
	if borrowing.Status != enums.BorrowingApproved && borrowing.Status != enums.BorrowingReturnRejected {
		return nil, errx.NewBorrowingNotReturnable(borrowing.Status)
	}

	photoPath, err := photo.NewKey(returnPhotoFolder, ext)
	if err != nil {
		return nil, err
	}
	if err := photo.Save(ctx, s.storage, req.Photo, photoPath); err != nil {
		return nil, err
	}

	err = s.tx.WithinTransaction(ctx, func(ctx context.Context) error {
		locked, err := s.repo.FindByIDForUpdate(ctx, id)
		if err != nil {
			return err
		}
		if err := locked.SubmitReturn(photoPath); err != nil {
			return err // status berubah sejak cek cepat (mis. permintaan ganda bersamaan)
		}
		if err := s.repo.UpdateReturn(ctx, locked); err != nil {
			return err
		}
		activity := entity.NewUserLog(userID, enums.RequestTypeBorrowing, id, enums.ActivityReturnSubmitted)
		return s.logRepo.Create(ctx, activity)
	})
	if err != nil {
		photo.Delete(ctx, s.storage, photoPath)
		return nil, err
	}

	return s.findResponse(ctx, id) // tanpa timeline
}

// ApproveReturn menyetujui pengembalian dalam SATU transaksi: kunci barang lalu pengajuan, pastikan masih
// RETURN_PENDING, kembalikan stok tiap barang sesuai jumlah di pengajuan, status RETURNED, catat log.
// Foto tetap disimpan sebagai bukti.
func (s *borrowingService) ApproveReturn(ctx context.Context, id, adminID uint) (*dto.BorrowingResponse, error) {
	borrowing, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if borrowing.Status != enums.BorrowingReturnPending {
		return nil, errx.NewReturnNotPending(borrowing.Status) // cek cepat sebelum mengunci apa pun
	}
	itemIDs := make([]uint, 0, len(borrowing.Items))
	for _, it := range borrowing.Items {
		itemIDs = append(itemIDs, it.ItemID)
	}

	err = s.tx.WithinTransaction(ctx, func(ctx context.Context) error {
		// 1. Kunci barang dulu, baru pengajuan (urutan sama dengan ACC peminjaman).
		items, err := s.itemRepo.FindByIDsForUpdate(ctx, itemIDs)
		if err != nil {
			return err
		}
		locked, err := s.repo.FindByIDForUpdate(ctx, id)
		if err != nil {
			return err
		}

		// 2. Status dicek ulang: mungkin sudah diproses admin lain.
		if err := locked.ApproveReturn(); err != nil {
			return err
		}

		// 3. Kembalikan stok, lalu simpan status & log.
		restocked, err := restockItems(borrowing.Items, items)
		if err != nil {
			return err
		}
		for _, item := range restocked {
			if err := s.itemRepo.Update(ctx, item); err != nil {
				return err
			}
		}
		if err := s.repo.UpdateStatus(ctx, locked); err != nil {
			return err
		}
		activity := entity.NewUserLog(adminID, enums.RequestTypeBorrowing, id, enums.ActivityReturnApproved)
		return s.logRepo.Create(ctx, activity)
	})
	if err != nil {
		return nil, err
	}

	return s.findResponse(ctx, id) // tanpa timeline
}

// RejectReturn menolak foto pengembalian: status RETURN_REJECTED, kolom foto dikosongkan, catat log.
// Stok tidak berubah (barang dianggap belum kembali). File foto lama dihapus SETELAH transaksi berhasil,
// supaya foto tidak hilang jika penyimpanan ke database gagal.
func (s *borrowingService) RejectReturn(ctx context.Context, id, adminID uint) (*dto.BorrowingResponse, error) {
	var oldPhoto *string
	err := s.tx.WithinTransaction(ctx, func(ctx context.Context) error {
		locked, err := s.repo.FindByIDForUpdate(ctx, id)
		if err != nil {
			return err // errx.ErrBorrowingNotFound
		}
		oldPhoto = locked.ReturnPhoto
		if err := locked.RejectReturn(); err != nil {
			return err
		}
		if err := s.repo.UpdateReturn(ctx, locked); err != nil {
			return err
		}
		activity := entity.NewUserLog(adminID, enums.RequestTypeBorrowing, id, enums.ActivityReturnRejected)
		return s.logRepo.Create(ctx, activity)
	})
	if err != nil {
		return nil, err
	}

	if oldPhoto != nil {
		photo.Delete(ctx, s.storage, *oldPhoto)
	}
	return s.findResponse(ctx, id) // tanpa timeline
}

// restockItems mengembalikan stok setiap barang di pengajuan. items adalah barang yang sudah dikunci.
// Barang yang masih dipinjam tidak bisa dihapus admin, jadi barang yang hilang berarti data tidak wajar
// (error biasa → 500).
func restockItems(lines []entity.BorrowingItem, items []entity.Item) ([]*entity.Item, error) {
	byID := make(map[uint]*entity.Item, len(items))
	for i := range items {
		byID[items[i].ID] = &items[i]
	}

	restocked := make([]*entity.Item, 0, len(lines))
	for _, line := range lines {
		item, ok := byID[line.ItemID]
		if !ok {
			return nil, fmt.Errorf("restock: item %d (%s) tidak ditemukan", line.ItemID, line.ItemName)
		}
		if err := item.Restock(line.Quantity); err != nil {
			return nil, err
		}
		restocked = append(restocked, item)
	}
	return restocked, nil
}
