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

// devLembagaNames adalah daftar lembaga awal untuk development.
var devLembagaNames = []string{"BEM", "HMDTIF", "DPM"}

// Seed mengisi data awal untuk development: user (Max & Raka) dan lembaga.
// Dipanggil dari main.go setelah Migrate, hanya jika APP_ENV=development.
//
// Semua seed memakai FirstOrCreate: cari dulu datanya; INSERT hanya jika belum ada.
// Jadi Seed aman dijalankan setiap start (Air me-restart app tiap file disimpan) tanpa data dobel.
// Kenapa tidak langsung INSERT ... ON CONFLICT DO NOTHING? Postgres tetap "memakai" satu nomor ID
// setiap kali INSERT dicoba walau akhirnya dilewati, sehingga ID berikutnya meloncat-loncat.
//
// Unscoped: ikut mencari data yang sudah soft delete, karena nilai uniknya (email/NIM/nama) tetap terpakai.
// ID tidak diisi manual supaya penghitung ID Postgres tetap sinkron.
func Seed(db *gorm.DB) error {
	if err := seedUsers(db); err != nil {
		return err
	}
	return seedLembaga(db)
}

// seedUsers membuat Max (ADMIN) & Raka (USER). Pada database baru: Max = id 1, Raka = id 2.
func seedUsers(db *gorm.DB) error {
	users := []entity.User{
		{
			FullName:     "Max",
			NIM:          "00000001",
			Email:        "max@gmail.com",
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

// seedLembaga membuat lembaga awal (BEM, HMDTIF, DPM) jika namanya belum ada.
func seedLembaga(db *gorm.DB) error {
	for _, name := range devLembagaNames {
		lembaga := entity.NewLembaga(name)
		err := db.Unscoped().
			Where("name = ?", name).
			FirstOrCreate(lembaga).Error
		if err != nil {
			return fmt.Errorf("seed lembaga %s: %w", name, err)
		}
	}
	return nil
}
