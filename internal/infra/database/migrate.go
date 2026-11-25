package database

import (
	"fmt"

	"gorm.io/gorm"

	"SGEInventory_BE/domain/entity"
)

// Migrate menyamakan struktur tabel di database dengan struct entity (tag `gorm:"..."`).
// Dipanggil setiap kali app start dari main.go.
//
// AutoMigrate aman dijalankan berulang kali: hanya MENAMBAH yang belum ada (tabel, kolom, index, CHECK)
// dan menyesuaikan tipe/ukuran kolom. Ia TIDAK menghapus atau mengganti nama kolom, jadi kolom yang
// dihapus dari struct tetap tertinggal di database dan harus dihapus manual.
//
// Fitur baru: buat entity-nya, lalu tambahkan &entity.Xxx{} ke daftar di bawah.
// Urutan penting: tabel yang dirujuk foreign key harus ada lebih dulu.
func Migrate(db *gorm.DB) error {
	if err := db.AutoMigrate(
		&entity.User{},
		&entity.Lembaga{},
		&entity.Consignment{}, // sebelum Item: items.consignment_id merujuk ke consignments
		&entity.Item{},
		&entity.Borrowing{},
		&entity.BorrowingItem{},
		&entity.UserLog{},
	); err != nil {
		return fmt.Errorf("auto migrate: %w", err)
	}
	return nil
}
