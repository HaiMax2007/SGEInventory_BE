package database

import (
	"fmt"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"SGEInventory_BE/internal/config"
)

// NewPostgres membuka koneksi ke PostgreSQL. Hasilnya (*gorm.DB) dioper ke repository di main.go.
// Tabel dibuat lewat file SQL di docker/*/postgres, jadi tidak ada AutoMigrate di sini.
func NewPostgres(cfg *config.Config) (*gorm.DB, error) {
	// TranslateError: email duplikat (kode error Postgres 23505) menjadi gorm.ErrDuplicatedKey
	// sehingga repository cukup mengecek errors.Is(err, gorm.ErrDuplicatedKey).
	db, err := gorm.Open(postgres.Open(cfg.DSN()), &gorm.Config{TranslateError: true})
	if err != nil {
		return nil, fmt.Errorf("connect database: %w", err)
	}
	return db, nil
}
