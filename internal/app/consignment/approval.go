package consignment

import (
	"context"

	"SGEInventory_BE/domain/dto"
	"SGEInventory_BE/domain/entity"
	"SGEInventory_BE/domain/enums"
	"SGEInventory_BE/pkg/photo"
)

// File ini berisi tahap 2 penitipan: admin menyetujui (ACC) atau menolak.
//
//	PENDING ─ACC─→ APPROVED (barang titipan tetap ada, status barang TERSEDIA)
//	   └─tolak─→ REJECTED (barang titipan dihapus permanen, foto dihapus; baris penitipan tetap ada)
//
// Keduanya hanya dari PENDING dan hanya sekali (final), sama seperti ACC/tolak peminjaman.

// ApproveConsignment menyetujui penitipan dalam SATU transaksi: kunci baris penitipan, pastikan masih PENDING,
// status APPROVED, catat log APPROVED (pelaku = admin).
func (s *consignmentService) ApproveConsignment(ctx context.Context, id, adminID uint) (*dto.ConsignmentResponse, error) {
	err := s.tx.WithinTransaction(ctx, func(ctx context.Context) error {
		// Dikunci supaya dua admin yang memproses bersamaan tidak sama-sama berhasil.
		consignment, err := s.repo.FindByIDForUpdate(ctx, id)
		if err != nil {
			return err // errx.ErrConsignmentNotFound
		}
		if err := consignment.Approve(); err != nil {
			return err // bukan PENDING → 409
		}
		if err := s.repo.UpdateStatus(ctx, consignment); err != nil {
			return err
		}
		activity := entity.NewUserLog(adminID, enums.RequestTypeConsignment, id, enums.ActivityApproved)
		return s.logRepo.Create(ctx, activity)
	})
	if err != nil {
		return nil, err
	}
	return s.findResponse(ctx, id) // tanpa timeline
}

// RejectConsignment menolak penitipan dalam SATU transaksi: status REJECTED + kolom foto dikosongkan, barang
// titipannya dihapus permanen, catat log REJECTED (pelaku = admin). File foto dihapus SETELAH transaksi berhasil,
// supaya foto tidak hilang jika penyimpanan ke database gagal.
func (s *consignmentService) RejectConsignment(ctx context.Context, id, adminID uint) (*dto.ConsignmentResponse, error) {
	var oldPhoto string
	err := s.tx.WithinTransaction(ctx, func(ctx context.Context) error {
		consignment, err := s.repo.FindByIDForUpdate(ctx, id)
		if err != nil {
			return err // errx.ErrConsignmentNotFound
		}
		oldPhoto = consignment.Photo
		if err := consignment.Reject(); err != nil {
			return err // bukan PENDING → 409
		}
		if err := s.repo.UpdateStatus(ctx, consignment); err != nil {
			return err
		}
		if err := s.repo.DeleteItems(ctx, id); err != nil {
			return err
		}
		activity := entity.NewUserLog(adminID, enums.RequestTypeConsignment, id, enums.ActivityRejected)
		return s.logRepo.Create(ctx, activity)
	})
	if err != nil {
		return nil, err
	}

	if oldPhoto != "" {
		photo.Delete(ctx, s.storage, oldPhoto)
	}
	return s.findResponse(ctx, id) // tanpa timeline
}
