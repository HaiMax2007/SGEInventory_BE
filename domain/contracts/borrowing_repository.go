package contracts

import (
	"context"

	"SGEInventory_BE/domain/entity"
)

// BorrowingRepository adalah "janji" cara mengakses data peminjaman di database.
// Implementasinya ada di internal/infra/repository/borrowing_repository.go (GORM).
type BorrowingRepository interface {
	// Create menyimpan pengajuan + barang-barangnya + log aktivitas dalam SATU transaksi.
	// RequestID pada activity diisi otomatis dengan id pengajuan yang baru dibuat.
	Create(ctx context.Context, borrowing *entity.Borrowing, activity *entity.UserLog) error
	// FindByID mengambil pengajuan beserta lembaga & barang-barangnya.
	FindByID(ctx context.Context, id uint) (*entity.Borrowing, error)
	// FindByUser mengambil satu halaman pengajuan milik seorang user (terbaru dulu) + jumlah seluruhnya.
	FindByUser(ctx context.Context, userID uint, offset, limit int) ([]entity.Borrowing, int64, error)
	// FindAll mengambil satu halaman SEMUA pengajuan (terbaru dulu) + jumlah seluruhnya.
	// status kosong = semua status.
	FindAll(ctx context.Context, status string, offset, limit int) ([]entity.Borrowing, int64, error)

	// Method di bawah dipakai di dalam Transactor.WithinTransaction (tahap approval admin).

	// FindByIDForUpdate mengambil & MENGUNCI satu pengajuan (tanpa relasi) sampai transaksi selesai.
	FindByIDForUpdate(ctx context.Context, id uint) (*entity.Borrowing, error)
	// UpdateStatus menyimpan status pengajuan (updated_at ikut berubah).
	UpdateStatus(ctx context.Context, borrowing *entity.Borrowing) error
	// UpdateReturn menyimpan status + foto pengembalian (return_photo; nil → NULL).
	UpdateReturn(ctx context.Context, borrowing *entity.Borrowing) error
	// LockUnfulfillablePending mencari & MENGUNCI pengajuan PENDING yang memuat salah satu itemIDs
	// dengan jumlah melebihi stok tersedia saat ini. Hasil: id pengajuan, urut naik.
	LockUnfulfillablePending(ctx context.Context, itemIDs []uint) ([]uint, error)
	// UpdateStatuses mengubah status banyak pengajuan sekaligus.
	UpdateStatuses(ctx context.Context, ids []uint, status string) error
}
