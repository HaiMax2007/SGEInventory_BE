package unit

import (
	"testing"

	"SGEInventory_BE/internal/config"
)

// Tes config memakai t.Setenv, jadi tidak boleh t.Parallel.

const testJWTSecret = "rahasia-uji-minimal-32-karakter-xx"

func TestLoadConfig_CloudinaryWajibDiProduction(t *testing.T) {
	t.Setenv("JWT_SECRET", testJWTSecret)
	t.Setenv("APP_ENV", "production")
	t.Setenv("CLOUDINARY_URL", "")

	if _, err := config.LoadConfig(); err == nil {
		t.Fatal("LoadConfig production tanpa CLOUDINARY_URL = nil; want error")
	}

	t.Setenv("CLOUDINARY_URL", "cloudinary://kunci:rahasia@democloud")
	cfg, err := config.LoadConfig()
	mustNoErr(t, err)
	if cfg.CloudinaryFolder != "sgeinventory" {
		t.Errorf("CloudinaryFolder = %q; want default \"sgeinventory\"", cfg.CloudinaryFolder)
	}
}

// Di development CLOUDINARY_URL boleh kosong: foto disimpan di folder lokal.
func TestLoadConfig_DevelopmentBolehTanpaCloudinary(t *testing.T) {
	t.Setenv("JWT_SECRET", testJWTSecret)
	t.Setenv("APP_ENV", "development")
	t.Setenv("CLOUDINARY_URL", "")

	cfg, err := config.LoadConfig()
	mustNoErr(t, err)
	if cfg.CloudinaryURL != "" {
		t.Errorf("CloudinaryURL = %q; want kosong", cfg.CloudinaryURL)
	}
}
