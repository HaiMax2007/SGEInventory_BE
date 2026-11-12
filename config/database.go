package config

import (
	"fmt"
	"os"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// DB dipakai oleh semua controller untuk mengakses database.
var DB *gorm.DB

// ConnectDatabase membuka koneksi ke MariaDB memakai env DB_* dari compose.
// Tabel dibuat lewat file SQL di docker/dev/mariadb, jadi tidak ada AutoMigrate di sini.
func ConnectDatabase() error {
	dsn := fmt.Sprintf(
		"%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		os.Getenv("DB_USER"),
		os.Getenv("DB_PASSWORD"),
		os.Getenv("DB_HOST"),
		os.Getenv("DB_PORT"),
		os.Getenv("DB_NAME"),
	)

	// TranslateError: email duplikat menjadi gorm.ErrDuplicatedKey sehingga mudah dicek.
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{TranslateError: true})
	if err != nil {
		return fmt.Errorf("connect database: %w", err)
	}

	DB = db
	return nil
}
