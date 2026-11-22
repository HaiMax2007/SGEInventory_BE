package dto

import (
	"time"

	"SGEInventory_BE/domain/entity"
)

// DateLayout adalah format tanggal (tanpa jam) yang dipakai FE & BE, mis. "2026-11-25".
const DateLayout = "2006-01-02"

// CreateBorrowingRequest adalah isi body POST /api/borrowings.
// Sengaja tanpa tag `validate`: tiap aturan dicek di service agar setiap kesalahan punya pesan sendiri
// (lihat domain/errx/borrowing_errx.go). Peminjam tidak ada di body, tapi diambil dari token.
type CreateBorrowingRequest struct {
	LembagaID  uint                   `json:"lembagaId"`
	Purpose    string                 `json:"purpose"`
	PickUpDate string                 `json:"pickUpDate"` // "YYYY-MM-DD"
	ReturnDate string                 `json:"returnDate"` // "YYYY-MM-DD"
	Items      []BorrowingItemRequest `json:"items"`
}

// BorrowingItemRequest adalah satu barang di dalam "items". Nama barang tidak dikirim FE;
// BE mengambilnya dari database.
type BorrowingItemRequest struct {
	ItemID   uint `json:"itemId"`
	Quantity int  `json:"quantity"`
}

// BorrowingListQuery adalah query string daftar pengajuan:
// GET /api/borrowings/me?page=1&limit=10 dan GET /api/borrowings?page=1&limit=10&status=PENDING (admin).
type BorrowingListQuery struct {
	Page   int
	Limit  int
	Status string // kosong = semua status (hanya dipakai di daftar admin)
}

// BorrowingItemResponse adalah satu barang di dalam pengajuan.
type BorrowingItemResponse struct {
	ItemID   uint   `json:"itemId"`
	ItemName string `json:"itemName"`
	Quantity int    `json:"quantity"`
}

// BorrowingResponse adalah bentuk data pengajuan peminjaman yang dikirim ke FE.
// Timeline hanya diisi di detail (GET /api/borrowings/:id); di daftar, key-nya tidak muncul.
type BorrowingResponse struct {
	ID         uint                    `json:"id"`
	Status     string                  `json:"status"`
	Purpose    string                  `json:"purpose"`
	PickUpDate string                  `json:"pickUpDate"`
	ReturnDate string                  `json:"returnDate"`
	User       *UserSummary            `json:"user"` // peminjam (pemilik pengajuan)
	Lembaga    *LembagaSummary         `json:"lembaga"`
	Items      []BorrowingItemResponse `json:"items"`
	Timeline   []UserLogResponse       `json:"timeline,omitempty"`
	CreatedAt  time.Time               `json:"createdAt"`
	UpdatedAt  time.Time               `json:"updatedAt"`
}

// NewBorrowingResponse mengubah entity.Borrowing menjadi BorrowingResponse (tanpa timeline).
func NewBorrowingResponse(b *entity.Borrowing) BorrowingResponse {
	items := make([]BorrowingItemResponse, 0, len(b.Items))
	for _, it := range b.Items {
		items = append(items, BorrowingItemResponse{ItemID: it.ItemID, ItemName: it.ItemName, Quantity: it.Quantity})
	}

	return BorrowingResponse{
		ID:         b.ID,
		Status:     b.Status,
		Purpose:    b.Purpose,
		PickUpDate: b.PickupDate.Format(DateLayout),
		ReturnDate: b.ReturnDate.Format(DateLayout),
		User:       NewUserSummary(b.User),
		Lembaga:    NewLembagaSummary(b.Lembaga),
		Items:      items,
		CreatedAt:  b.CreatedAt,
		UpdatedAt:  b.UpdatedAt,
	}
}
