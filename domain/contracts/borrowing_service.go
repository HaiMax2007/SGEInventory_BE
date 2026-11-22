package contracts

import (
	"context"

	"SGEInventory_BE/domain/dto"
	"SGEInventory_BE/helpers/pagination"
)

// BorrowingService berisi aturan bisnis peminjaman. Implementasinya ada di internal/app/borrowing/service.go.
type BorrowingService interface {
	// CreateBorrowing memvalidasi & menyimpan pengajuan milik userID (dari token).
	CreateBorrowing(ctx context.Context, userID uint, req dto.CreateBorrowingRequest) (*dto.BorrowingResponse, error)
	// GetMyBorrowings mengambil daftar pengajuan milik userID.
	GetMyBorrowings(ctx context.Context, userID uint, query dto.BorrowingListQuery) ([]dto.BorrowingResponse, *pagination.Pagination, error)
	// GetBorrowings mengambil daftar SEMUA pengajuan (khusus admin), bisa difilter status.
	GetBorrowings(ctx context.Context, query dto.BorrowingListQuery) ([]dto.BorrowingResponse, *pagination.Pagination, error)
	// GetBorrowing mengambil detail + timeline. Hanya pemilik atau admin yang boleh melihat.
	GetBorrowing(ctx context.Context, id, userID uint, isAdmin bool) (*dto.BorrowingResponse, error)
}
