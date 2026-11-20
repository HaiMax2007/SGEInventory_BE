package database

import (
	"fmt"

	"gorm.io/gorm"

	"SGEInventory_BE/domain/entity"
	"SGEInventory_BE/domain/enums"
)

// devPasswordHash adalah hash bcrypt untuk akun awal development (password aslinya tidak disimpan di kode).
// Hanya dipakai di development; production tidak menjalankan Seed.
const devPasswordHash = "$2a$10$S7QOM2wmv727X/rQo.okB.ta0isdJztH63bqwxcHYeTI5.p1tfX6."

// Seed mengisi user awal untuk development. Dipanggil dari main.go setelah Migrate,
// hanya jika APP_ENV=development.
//
// FirstOrCreate: cari dulu user dengan email/NIM yang sama; INSERT hanya jika belum ada.
// Jadi Seed aman dijalankan setiap start (Air me-restart app tiap file disimpan) tanpa data dobel.
// Kenapa tidak langsung INSERT ... ON CONFLICT DO NOTHING? Postgres tetap "memakai" satu nomor ID
// setiap kali INSERT dicoba walau akhirnya dilewati, sehingga ID user berikutnya meloncat-loncat.
//
// Unscoped: ikut mencari user yang sudah soft delete, karena email/NIM-nya tetap terpakai di tabel.
// ID tidak diisi manual supaya penghitung ID Postgres tetap sinkron (Max = 1, Raka = 2 pada database baru).
func Seed(db *gorm.DB) error {
	users := []entity.User{
		{
			FullName:     "Max",
			NIM:          "00000001",
			Email:        "marshallmaranda34@gmail.com",
			PasswordHash: devPasswordHash,
			Contact:      "089602160276",
			Role:         enums.RoleAdmin,
		},
		{
			FullName:     "Raka",
			NIM:          "00000002",
			Email:        "rakhaadrian@gmail.com",
			PasswordHash: devPasswordHash,
			Role:         enums.RoleUser,
		},
	}

	for i := range users {
		u := &users[i]
		err := db.Unscoped().
			Where("email = ? OR nim = ?", u.Email, u.NIM).
			FirstOrCreate(u).Error
		if err != nil {
			return fmt.Errorf("seed user %s: %w", u.Email, err)
		}
	}
	return nil
}
