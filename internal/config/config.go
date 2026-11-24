// Package config membaca semua pengaturan aplikasi dari environment variable.
// Nilainya diisi oleh compose.dev.yaml / compose.prod.yaml.
package config

import (
	"errors"
	"fmt"
	"os"
	"strings"
)

type Config struct {
	AppEnv      string // "development" atau "production"
	AppPort     string
	DBHost      string
	DBPort      string
	DBUser      string
	DBPassword  string
	DBName      string
	DBSSLMode   string
	JWTSecret   string
	CORSOrigins []string
	UploadDir   string // folder file unggahan cadangan (dev tanpa Cloudinary), dilayani di URL /uploads/...

	// CloudinaryURL: cloudinary://API_KEY:API_SECRET@CLOUD_NAME. Berisi SECRET, jangan pernah dicetak ke log.
	// Diisi → foto disimpan di Cloudinary. Kosong → folder lokal (hanya boleh di development).
	CloudinaryURL    string
	CloudinaryFolder string // folder di Cloudinary, default "sgeinventory"
}

// LoadConfig membaca env lalu memastikan nilai penting sudah benar.
func LoadConfig() (*Config, error) {
	cfg := &Config{
		// Default "production" supaya data awal (Seed) tidak pernah masuk tanpa sengaja.
		AppEnv:           getEnv("APP_ENV", "production"),
		AppPort:          getEnv("PORT", "3000"),
		DBHost:           os.Getenv("DB_HOST"),
		DBPort:           os.Getenv("DB_PORT"),
		DBUser:           os.Getenv("DB_USER"),
		DBPassword:       os.Getenv("DB_PASSWORD"),
		DBName:           os.Getenv("DB_NAME"),
		DBSSLMode:        getEnv("DB_SSLMODE", "disable"),
		JWTSecret:        os.Getenv("JWT_SECRET"),
		CORSOrigins:      strings.Split(getEnv("CORS_ORIGINS", "http://localhost:5173,http://localhost:3001"), ","),
		UploadDir:        getEnv("UPLOAD_DIR", "uploads"),
		CloudinaryURL:    os.Getenv("CLOUDINARY_URL"),
		CloudinaryFolder: getEnv("CLOUDINARY_FOLDER", "sgeinventory"),
	}

	if len(cfg.JWTSecret) < 32 {
		return nil, errors.New("JWT_SECRET wajib diisi, minimal 32 karakter")
	}
	// Di production foto wajib ke Cloudinary: folder di dalam container hilang saat container dibuat ulang.
	if !cfg.IsDevelopment() && cfg.CloudinaryURL == "" {
		return nil, errors.New("CLOUDINARY_URL wajib diisi di production")
	}
	return cfg, nil
}

// IsDevelopment bernilai true jika APP_ENV=development (diset di compose.dev.yaml).
func (c *Config) IsDevelopment() bool {
	return c.AppEnv == "development"
}

// DSN adalah alamat koneksi PostgreSQL untuk GORM, formatnya "kunci=nilai" dipisah spasi.
// sslmode=disable cukup karena API dan database berada di jaringan Docker yang sama.
func (c *Config) DSN() string {
	return fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		c.DBHost, c.DBPort, c.DBUser, c.DBPassword, c.DBName, c.DBSSLMode)
}

// getEnv mengambil env; jika kosong, pakai nilai fallback.
func getEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}
