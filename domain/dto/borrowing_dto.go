package dto

import (
	"mime/multipart"
	"time"

	"SGEInventory_BE/domain/entity"
	"SGEInventory_BE/domain/errx"
)

// DateLayout adalah format tanggal (tanpa jam) yang dipakai FE & BE, mis. "2026-11-25".
const DateLayout = "2006-01-02"

// CreateBorrowingRequest adalah isi body POST /api/borrowings. Peminjam tidak ada di body, tapi diambil dari token.
//
// Aturan per field ada di tag `validate` (urutan field = urutan pengecekan, error pertama yang dikirim).
// "dive" berarti setiap barang di dalam items ikut dicek dengan tag milik BorrowingItemRequest.
// Aturan antar-field (barang dobel, tanggal lewat, kembali setelah ambil, maksimal 7 hari) dicek di service.
type CreateBorrowingRequest struct {
	Items      []BorrowingItemRequest `json:"items" validate:"required,min=1,max=20,dive"`
	LembagaID  uint                   `json:"lembagaId" validate:"required"`
	Purpose    string                 `json:"purpose" validate:"required,max=500"`
	PickUpDate string                 `json:"pickUpDate" validate:"required,datetime=2006-01-02"` // "YYYY-MM-DD"
	ReturnDate string                 `json:"returnDate" validate:"required,datetime=2006-01-02"` // "YYYY-MM-DD"
}

// FieldErrors memetakan field yang gagal validasi ke error-nya (dipakai validator.ValidateStruct).
// ItemID & Quantity adalah field di dalam setiap barang (BorrowingItemRequest).
func (CreateBorrowingRequest) FieldErrors() map[string]error {
	return map[string]error{
		"Items":      errx.ErrInvalidBorrowItems,
		"ItemID":     errx.ErrInvalidBorrowItems,
		"Quantity":   errx.ErrInvalidBorrowItems,
		"LembagaID":  errx.ErrInvalidBorrowLembaga,
		"Purpose":    errx.ErrInvalidPurpose,
		"PickUpDate": errx.ErrInvalidPickupDate,
		"ReturnDate": errx.ErrInvalidReturnDate,
	}
}

// BorrowingItemRequest adalah satu barang di dalam "items". Nama barang tidak dikirim FE;
// BE mengambilnya dari database.
type BorrowingItemRequest struct {
	ItemID   uint `json:"itemId" validate:"required"`
	Quantity int  `json:"quantity" validate:"gt=0"`
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
	// ReturnPhotoURL: alamat foto bukti pengembalian, mis. "/uploads/returns/8f3a...jpg"
	// (gabungkan dengan alamat API di FE). null jika belum ada foto.
	ReturnPhotoURL *string           `json:"returnPhotoUrl"`
	Timeline       []UserLogResponse `json:"timeline,omitempty"`
	CreatedAt      time.Time         `json:"createdAt"`
	UpdatedAt      time.Time         `json:"updatedAt"`
}

// ApproveBorrowingResponse adalah hasil ACC: pengajuan yang disetujui (detail + timeline) dan id pengajuan
// PENDING lain yang ikut ditolak otomatis karena stoknya tidak cukup lagi ([] jika tidak ada).
type ApproveBorrowingResponse struct {
	Borrowing       BorrowingResponse `json:"borrowing"`
	AutoRejectedIDs []uint            `json:"autoRejectedIds"`
}

// NewBorrowingResponse mengubah entity.Borrowing menjadi BorrowingResponse (tanpa timeline).
func NewBorrowingResponse(b *entity.Borrowing) BorrowingResponse {
	items := make([]BorrowingItemResponse, 0, len(b.Items))
	for _, it := range b.Items {
		items = append(items, BorrowingItemResponse{ItemID: it.ItemID, ItemName: it.ItemName, Quantity: it.Quantity})
	}

	return BorrowingResponse{
		ID:             b.ID,
		Status:         b.Status,
		Purpose:        b.Purpose,
		PickUpDate:     b.PickupDate.Format(DateLayout),
		ReturnDate:     b.ReturnDate.Format(DateLayout),
		User:           NewUserSummary(b.User),
		Lembaga:        NewLembagaSummary(b.Lembaga),
		Items:          items,
		ReturnPhotoURL: uploadURL(b.ReturnPhoto),
		CreatedAt:      b.CreatedAt,
		UpdatedAt:      b.UpdatedAt,
	}
}

// UploadsURLPrefix adalah awalan URL file unggahan. Route static di cmd/app/main.go memakai awalan yang sama.
const UploadsURLPrefix = "/uploads/"

// uploadURL mengubah path file (mis. "returns/a.jpg") menjadi URL ("/uploads/returns/a.jpg"). nil tetap nil.
func uploadURL(path *string) *string {
	if path == nil {
		return nil
	}
	url := UploadsURLPrefix + *path
	return &url
}

// SubmitReturnRequest adalah isi form POST /api/borrowings/:id/return (multipart/form-data, field "photo").
// "required" hanya memastikan file dikirim; ukuran (maks 5 MB) & jenis gambar dicek service dari isi file.
type SubmitReturnRequest struct {
	Photo *multipart.FileHeader `validate:"required"`
}

// FieldErrors memetakan field yang gagal validasi ke error-nya (dipakai validator.ValidateStruct).
func (SubmitReturnRequest) FieldErrors() map[string]error {
	return map[string]error{"Photo": errx.ErrInvalidReturnPhoto}
}
