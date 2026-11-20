package entity

import (
	"time"

	"gorm.io/gorm"

	"SGEInventory_BE/domain/enums"
	"SGEInventory_BE/domain/errx"
)

// Item adalah cermin tabel items (dibuat oleh database.Migrate).
// CHECK menjaga stok tetap masuk akal walau ada yang mengubah data langsung di database.
type Item struct {
	ID   uint   `gorm:"primaryKey"`
	Name string `gorm:"size:255;not null"`
	// jumlah barang yang dimiliki
	TotalQuantity int `gorm:"not null;check:chk_items_total_non_negative,total_quantity >= 0"`
	// sisa yang bisa dipinjam: 0 sampai total. GORM hanya membaca satu "check:" per field,
	// jadi dua aturan (>= 0 dan <= total) digabung dalam satu CHECK.
	AvailableQuantity int `gorm:"not null;index:idx_items_available;check:chk_items_available_range,available_quantity >= 0 AND available_quantity <= total_quantity"`
	CreatedAt         time.Time
	UpdatedAt         time.Time
	DeletedAt         gorm.DeletedAt `gorm:"index"` // soft delete
}

func (Item) TableName() string {
	return "items"
}

// NewItem membuat barang baru. Belum ada yang dipinjam, jadi sisa = total.
func NewItem(name string, totalQuantity int) *Item {
	return &Item{
		Name:              name,
		TotalQuantity:     totalQuantity,
		AvailableQuantity: totalQuantity,
	}
}

// BorrowedQuantity adalah jumlah barang yang sedang dipinjam.
func (i *Item) BorrowedQuantity() int {
	return i.TotalQuantity - i.AvailableQuantity
}

// IsBorrowed bernilai true jika masih ada barang yang dipinjam.
func (i *Item) IsBorrowed() bool {
	return i.BorrowedQuantity() > 0
}

// ChangeTotal mengubah total barang. Sisa ikut bergeser sebesar selisihnya.
// Total tidak boleh lebih kecil dari jumlah yang sedang dipinjam.
func (i *Item) ChangeTotal(newTotal int) error {
	borrowed := i.BorrowedQuantity()
	if newTotal < borrowed {
		return errx.NewTotalBelowBorrowed(borrowed)
	}
	i.TotalQuantity = newTotal
	i.AvailableQuantity = newTotal - borrowed
	return nil
}

// Status dihitung dari stok, tidak disimpan di database.
func (i *Item) Status() string {
	if i.AvailableQuantity > 0 {
		return enums.ItemStatusAvailable
	}
	return enums.ItemStatusEmpty
}
