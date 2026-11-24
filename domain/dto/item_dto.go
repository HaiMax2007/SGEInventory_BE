package dto

import (
	"time"

	"SGEInventory_BE/domain/entity"
	"SGEInventory_BE/domain/errx"
)

// CreateItemRequest adalah isi body POST /api/items. Description opsional, lembagaId wajib.
// "required" pada uint berarti tidak boleh 0 (0 = tidak dikirim).
// TotalQuantity sengaja TIDAK "required": pada angka, required menolak 0, padahal total 0 boleh.
type CreateItemRequest struct {
	Name          string `json:"name" validate:"required,min=2,max=255"`
	Description   string `json:"description" validate:"omitempty,max=1000"`
	LembagaID     uint   `json:"lembagaId" validate:"required"`
	TotalQuantity int    `json:"totalQuantity" validate:"gte=0,lte=1000000"`
}

// FieldErrors memetakan field yang gagal validasi ke error-nya (dipakai validator.ValidateStruct).
func (CreateItemRequest) FieldErrors() map[string]error {
	return itemFieldErrors()
}

// UpdateItemRequest adalah isi body PUT /api/items/:id. Semua field opsional.
// TotalQuantity memakai pointer (*int) supaya bisa membedakan "tidak diisi" (nil)
// dengan "diisi 0". Dengan int biasa, keduanya sama-sama bernilai 0.
// Description juga pointer (*string): tidak dikirim (nil) = tidak berubah,
// dikirim "" = keterangan dikosongkan.
// LembagaID pointer (*uint): tidak dikirim (nil) = lembaga asal tidak berubah.
type UpdateItemRequest struct {
	Name          string  `json:"name" validate:"omitempty,min=2,max=255"`
	Description   *string `json:"description" validate:"omitempty,max=1000"`
	LembagaID     *uint   `json:"lembagaId" validate:"omitempty,gt=0"`
	TotalQuantity *int    `json:"totalQuantity" validate:"omitempty,gte=0,lte=1000000"`
}

// FieldErrors memetakan field yang gagal validasi ke error-nya (dipakai validator.ValidateStruct).
func (UpdateItemRequest) FieldErrors() map[string]error {
	return itemFieldErrors()
}

// itemFieldErrors dipakai bersama form tambah & ubah barang (nama field-nya sama).
func itemFieldErrors() map[string]error {
	return map[string]error{
		"Name":          errx.ErrInvalidItemName,
		"Description":   errx.ErrInvalidItemDescription,
		"LembagaID":     errx.ErrInvalidItemLembaga,
		"TotalQuantity": errx.ErrInvalidItemTotal,
	}
}

// ItemListQuery adalah query string GET /api/items?page=1&limit=10&search=kabel&available=true.
type ItemListQuery struct {
	Page          int
	Limit         int
	Search        string
	OnlyAvailable bool
}

// ItemResponse adalah bentuk data barang yang dikirim ke FE.
type ItemResponse struct {
	ID                uint            `json:"id"`
	Name              string          `json:"name"`
	Description       string          `json:"description"`
	Lembaga           *LembagaSummary `json:"lembaga"` // {id, name} lembaga asal barang
	TotalQuantity     int             `json:"totalQuantity"`
	AvailableQuantity int             `json:"availableQuantity"`
	Status            string          `json:"status"` // TERSEDIA / HABIS, dihitung dari stok
	CreatedAt         time.Time       `json:"createdAt"`
	UpdatedAt         time.Time       `json:"updatedAt"`
}

// NewItemResponse mengubah entity.Item menjadi ItemResponse.
func NewItemResponse(i *entity.Item) ItemResponse {
	return ItemResponse{
		ID:                i.ID,
		Name:              i.Name,
		Description:       i.Description,
		Lembaga:           NewLembagaSummary(i.Lembaga),
		TotalQuantity:     i.TotalQuantity,
		AvailableQuantity: i.AvailableQuantity,
		Status:            i.Status(),
		CreatedAt:         i.CreatedAt,
		UpdatedAt:         i.UpdatedAt,
	}
}
