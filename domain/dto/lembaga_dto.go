package dto

import (
	"time"

	"SGEInventory_BE/domain/entity"
)

// CreateLembagaRequest adalah isi body POST /api/lembaga.
type CreateLembagaRequest struct {
	Name string `json:"name" validate:"required,min=2,max=150"`
}

// UpdateLembagaRequest adalah isi body PUT /api/lembaga/:id.
// Lembaga hanya punya satu field yang bisa diubah, jadi name tetap wajib diisi.
type UpdateLembagaRequest struct {
	Name string `json:"name" validate:"required,min=2,max=150"`
}

// LembagaListQuery adalah query string GET /api/lembaga?page=1&limit=10&search=bem.
type LembagaListQuery struct {
	Page   int
	Limit  int
	Search string
}

// LembagaResponse adalah bentuk data lembaga yang dikirim ke FE.
type LembagaResponse struct {
	ID        uint      `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// LembagaSummary adalah ringkasan lembaga yang ikut ditampilkan di data lain (mis. lembaga asal barang).
type LembagaSummary struct {
	ID   uint   `json:"id"`
	Name string `json:"name"`
}

// NewLembagaSummary mengubah entity.Lembaga menjadi LembagaSummary.
// Mengembalikan nil jika lembaga belum di-Preload, sehingga JSON-nya "lembaga": null.
func NewLembagaSummary(l *entity.Lembaga) *LembagaSummary {
	if l == nil {
		return nil
	}
	return &LembagaSummary{ID: l.ID, Name: l.Name}
}

// NewLembagaResponse mengubah entity.Lembaga menjadi LembagaResponse.
func NewLembagaResponse(l *entity.Lembaga) LembagaResponse {
	return LembagaResponse{
		ID:        l.ID,
		Name:      l.Name,
		CreatedAt: l.CreatedAt,
		UpdatedAt: l.UpdatedAt,
	}
}
