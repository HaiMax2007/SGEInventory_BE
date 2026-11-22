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
}
