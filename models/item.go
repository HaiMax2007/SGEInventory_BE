package models

import (
	"time"

	"gorm.io/gorm"
)

// Status barang tidak disimpan di database, tapi dihitung dari stok yang tersisa.
const (
	ItemStatusAvailable = "TERSEDIA"
	ItemStatusEmpty     = "HABIS"
)

// Item adalah cermin tabel items.
type Item struct {
	ID                uint           `gorm:"primaryKey" json:"id"`
	Name              string         `gorm:"size:255;not null" json:"name"`
	TotalQuantity     int            `gorm:"not null" json:"totalQuantity"`     // jumlah yang dimiliki
	AvailableQuantity int            `gorm:"not null" json:"availableQuantity"` // sisa yang bisa dipinjam
	Status            string         `gorm:"-" json:"status"`                   // gorm:"-" = bukan kolom tabel
	CreatedAt         time.Time      `json:"createdAt"`
	UpdatedAt         time.Time      `json:"updatedAt"`
	DeletedAt         gorm.DeletedAt `gorm:"index" json:"-"`
}

func (Item) TableName() string {
	return "items"
}

// SetStatus mengisi Status berdasarkan stok. Panggil sebelum item dikirim sebagai respons.
func (i *Item) SetStatus() {
	if i.AvailableQuantity > 0 {
		i.Status = ItemStatusAvailable
	} else {
		i.Status = ItemStatusEmpty
	}
}

// CreateItemRequest adalah isi body POST /api/items.
type CreateItemRequest struct {
	Name          string `json:"name" validate:"required,min=2,max=255"`
	TotalQuantity int    `json:"totalQuantity" validate:"gte=0,lte=1000000"`
}

// UpdateItemRequest adalah isi body PUT /api/items/:id. Semua field opsional.
// TotalQuantity memakai pointer (*int) supaya bisa membedakan "tidak diisi" (nil)
// dengan "diisi 0". Dengan int biasa, keduanya sama-sama bernilai 0.
type UpdateItemRequest struct {
	Name          string `json:"name" validate:"omitempty,min=2,max=255"`
	TotalQuantity *int   `json:"totalQuantity" validate:"omitempty,gte=0,lte=1000000"`
}
