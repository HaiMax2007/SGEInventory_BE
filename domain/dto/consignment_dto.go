package dto

import (
	"encoding/json"
	"mime/multipart"
	"strings"
	"time"

	"SGEInventory_BE/domain/entity"
	"SGEInventory_BE/domain/errx"
)

// CreateConsignmentRequest adalah isi form POST /api/consignments (multipart/form-data, karena ada foto).
//
// Field biasa dibaca c.Bind().Body (tag `form`). Dua field diisi handler sendiri (`form:"-"`):
//   - Items: field "items" berisi teks JSON, bentuknya sama dengan pengajuan: [{"name":"Kursi","quantity":3}]
//     (dibaca lewat ParseItems)
//   - Photo: file dari field "photo"
//
// Urutan field = urutan di form; jika ada beberapa kesalahan, yang dikirim hanya yang pertama.
type CreateConsignmentRequest struct {
	LembagaID uint                     `form:"lembagaId" validate:"required"`
	EventName string                   `form:"eventName" validate:"required,min=3,max=255"`
	Items     []ConsignmentItemRequest `form:"-" validate:"required,min=1,max=20,dive"`
	StartDate string                   `form:"startDate" validate:"required,datetime=2006-01-02"`
	EndDate   string                   `form:"endDate" validate:"required,datetime=2006-01-02"`
	// "required" hanya memastikan file dikirim; ukuran (maks 5 MB) & jenis gambar dicek service dari isi file.
	Photo *multipart.FileHeader `form:"-" validate:"required"`
	Notes string                `form:"notes" validate:"omitempty,max=1000"`
}

// ConsignmentItemRequest adalah satu barang titipan di dalam field "items".
type ConsignmentItemRequest struct {
	Name     string `json:"name" validate:"required,min=2,max=255"`
	Quantity int    `json:"quantity" validate:"gt=0,lte=1000000"`
}

// ParseItems membaca teks JSON field "items" ke r.Items. Teks kosong, JSON rusak, atau tipe salah (mis.
// quantity berupa teks) membuat Items kosong (nil), lalu tag `required` di validator membalasnya dengan
// errx.ErrInvalidConsignmentItems. Dengan begitu urutan error tetap sesuai urutan form (lembaga & nama acara
// dicek lebih dulu). Nama barang di-trim di sini, karena validator hanya men-trim teks di level atas struct.
func (r *CreateConsignmentRequest) ParseItems(raw string) {
	r.Items = nil
	var items []ConsignmentItemRequest
	if err := json.Unmarshal([]byte(raw), &items); err != nil {
		return
	}
	for i := range items {
		items[i].Name = strings.TrimSpace(items[i].Name)
	}
	r.Items = items
}

// FieldErrors memetakan field yang gagal validasi ke error-nya (dipakai validator.ValidateStruct).
// Name & Quantity adalah field di dalam items, jadi ikut dipetakan ke error daftar barang.
func (CreateConsignmentRequest) FieldErrors() map[string]error {
	return map[string]error{
		"LembagaID": errx.ErrInvalidConsignmentLembaga,
		"EventName": errx.ErrInvalidEventName,
		"Items":     errx.ErrInvalidConsignmentItems,
		"Name":      errx.ErrInvalidConsignmentItems,
		"Quantity":  errx.ErrInvalidConsignmentItems,
		"StartDate": errx.ErrInvalidStartDate,
		"EndDate":   errx.ErrInvalidEndDate,
		"Photo":     errx.ErrInvalidConsignmentPhoto,
		"Notes":     errx.ErrInvalidConsignmentNotes,
	}
}

// ConsignmentItemResponse adalah satu barang titipan di response (id = id barang di tabel items).
type ConsignmentItemResponse struct {
	ID       uint   `json:"id"`
	Name     string `json:"name"`
	Quantity int    `json:"quantity"`
}

// ConsignmentResponse adalah bentuk data pengajuan penitipan yang dikirim ke FE.
type ConsignmentResponse struct {
	ID        uint                      `json:"id"`
	Status    string                    `json:"status"`
	EventName string                    `json:"eventName"`
	StartDate string                    `json:"startDate"`
	EndDate   string                    `json:"endDate"`
	Notes     string                    `json:"notes"`
	PhotoURL  *string                   `json:"photoUrl"` // diisi service dari storage; null jika foto sudah dihapus (ditolak)
	User      *UserSummary              `json:"user"`     // penitip (pemilik pengajuan)
	Lembaga   *LembagaSummary           `json:"lembaga"`
	Items     []ConsignmentItemResponse `json:"items"`
	CreatedAt time.Time                 `json:"createdAt"`
	UpdatedAt time.Time                 `json:"updatedAt"`
}

// NewConsignmentResponse mengubah entity.Consignment menjadi ConsignmentResponse.
// PhotoURL tidak diisi di sini, karena URL-nya bergantung pada storage yang dipakai.
func NewConsignmentResponse(c *entity.Consignment) ConsignmentResponse {
	items := make([]ConsignmentItemResponse, 0, len(c.Items))
	for _, it := range c.Items {
		items = append(items, ConsignmentItemResponse{ID: it.ID, Name: it.Name, Quantity: it.TotalQuantity})
	}
	return ConsignmentResponse{
		ID:        c.ID,
		Status:    c.Status,
		EventName: c.EventName,
		StartDate: c.StartDate.Format(DateLayout),
		EndDate:   c.EndDate.Format(DateLayout),
		Notes:     c.Notes,
		User:      NewUserSummary(c.User),
		Lembaga:   NewLembagaSummary(c.Lembaga),
		Items:     items,
		CreatedAt: c.CreatedAt,
		UpdatedAt: c.UpdatedAt,
	}
}

// ConsignmentListQuery adalah query string daftar penitipan:
// GET /api/consignments/me?page=1&limit=10 dan GET /api/consignments?page=1&limit=10&status=PENDING (admin).
type ConsignmentListQuery struct {
	Page   int
	Limit  int
	Status string // kosong = semua status (hanya dipakai di daftar admin)
}
