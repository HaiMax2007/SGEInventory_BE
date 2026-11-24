package entity

import (
	"fmt"
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
	// keterangan barang (opsional). Kolom boleh NULL, GORM membacanya sebagai "".
	Description string `gorm:"type:text"`
	// lembaga asal barang (wajib). Ini kolom foreign key items.lembaga_id → lembaga.id.
	LembagaID uint `gorm:"not null;index:idx_items_lembaga_id"`
	// Lembaga BUKAN kolom: ini relasi "belongs to" untuk GORM, hanya terisi jika query memakai
	// Preload("Lembaga"). Aturan FOREIGN KEY-nya ditulis di sisi lain relasi (Lembaga.Items).
	Lembaga *Lembaga
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
func NewItem(name, description string, lembagaID uint, totalQuantity int) *Item {
	return &Item{
		Name:              name,
		Description:       description,
		LembagaID:         lembagaID,
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

// CanLend bernilai true jika stok tersedia cukup untuk dipinjam sebanyak qty.
func (i *Item) CanLend(qty int) bool {
	return qty > 0 && qty <= i.AvailableQuantity
}

// Lend mengurangi stok tersedia karena qty barang dipinjam (dipanggil saat pengajuan di-ACC).
func (i *Item) Lend(qty int) error {
	if !i.CanLend(qty) {
		return errx.NewInsufficientStock([]string{LendShortage(i.Name, qty, i.AvailableQuantity)})
	}
	i.AvailableQuantity -= qty
	return nil
}

// Restock menambah stok tersedia karena qty barang dikembalikan (dipanggil saat pengembalian di-ACC).
// Stok tersedia tidak boleh melebihi total. Ini seharusnya tidak pernah terjadi (ChangeTotal menjaga
// total ≥ jumlah dipinjam), jadi errornya error biasa: dicatat di log & dibalas 500, bukan kesalahan user.
func (i *Item) Restock(qty int) error {
	if qty <= 0 || i.AvailableQuantity+qty > i.TotalQuantity {
		return fmt.Errorf("restock item %d: tersedia %d + %d melebihi total %d",
			i.ID, i.AvailableQuantity, qty, i.TotalQuantity)
	}
	i.AvailableQuantity += qty
	return nil
}

// LendShortage membuat satu keterangan barang yang stoknya kurang, mis. "Proyektor (diminta 5, tersedia 3)".
func LendShortage(name string, requested, available int) string {
	return fmt.Sprintf("%s (diminta %d, tersedia %d)", name, requested, available)
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
