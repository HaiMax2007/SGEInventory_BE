package entity

import "time"

// BorrowingItem adalah cermin tabel borrowing_items: satu baris per barang dalam satu pengajuan.
// Satu barang hanya boleh muncul sekali per pengajuan (index unik uq_borrowing_items_borrowing_item).
type BorrowingItem struct {
	ID          uint  `gorm:"primaryKey"`
	BorrowingID uint  `gorm:"not null;uniqueIndex:uq_borrowing_items_borrowing_item,priority:1"`
	ItemID      uint  `gorm:"not null;uniqueIndex:uq_borrowing_items_borrowing_item,priority:2;index:idx_borrowing_items_item_id"`
	Item        *Item `gorm:"constraint:OnUpdate:CASCADE,OnDelete:RESTRICT"`
	// salinan nama barang saat diajukan, supaya riwayat tetap benar walau barangnya nanti diganti nama
	ItemName  string `gorm:"size:255;not null"`
	Quantity  int    `gorm:"not null;check:chk_borrowing_items_quantity_positive,quantity > 0"`
	CreatedAt time.Time
}

func (BorrowingItem) TableName() string {
	return "borrowing_items"
}

// NewBorrowingItem membuat satu baris barang untuk pengajuan. BorrowingID diisi GORM saat header disimpan.
func NewBorrowingItem(itemID uint, itemName string, quantity int) BorrowingItem {
	return BorrowingItem{ItemID: itemID, ItemName: itemName, Quantity: quantity}
}
