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

	// ApproveBorrowing (admin): kurangi stok, status APPROVED, lalu tolak otomatis pengajuan PENDING lain
	// yang stoknya kini tidak cukup.
	ApproveBorrowing(ctx context.Context, id, adminID uint) (*dto.ApproveBorrowingResponse, error)
	// RejectBorrowing (admin): status REJECTED, stok tidak berubah.
	RejectBorrowing(ctx context.Context, id, adminID uint) (*dto.BorrowingResponse, error)

	// SubmitReturn (user pemilik): simpan foto bukti, status RETURN_PENDING.
	SubmitReturn(ctx context.Context, id, userID uint, req dto.SubmitReturnRequest) (*dto.BorrowingResponse, error)
	// ApproveReturn (admin): stok tiap barang kembali, status RETURNED (alur selesai).
	ApproveReturn(ctx context.Context, id, adminID uint) (*dto.BorrowingResponse, error)
	// RejectReturn (admin): status RETURN_REJECTED, foto dihapus, user mengunggah foto baru.
	RejectReturn(ctx context.Context, id, adminID uint) (*dto.BorrowingResponse, error)

	PendingBorrowingRejecter
}

// PendingBorrowingRejecter menolak otomatis pengajuan PENDING yang tidak bisa dipenuhi lagi karena stok
// barangnya berkurang. Dipakai saat ACC dan saat admin mengurangi total barang (ItemService),
// jadi dibuat interface kecil sendiri: ItemService cukup bergantung pada kemampuan ini saja.
type PendingBorrowingRejecter interface {
	// RejectUnfulfillablePending memeriksa pengajuan PENDING yang memuat itemIDs. Yang jumlahnya melebihi
	// stok tersedia diubah ke REJECTED + log AUTO_REJECTED dengan pelaku actorID. Hasil: id yang ditolak.
	RejectUnfulfillablePending(ctx context.Context, itemIDs []uint, actorID uint) ([]uint, error)
}
