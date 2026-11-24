package unit

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"SGEInventory_BE/internal/infra/storage"
)

// Tes LocalStorage sungguhan di folder sementara (t.TempDir() otomatis dihapus setelah tes).

func TestLocalStorage_SaveAndDelete(t *testing.T) {
	t.Parallel()
	dir := filepath.Join(t.TempDir(), "uploads") // folder belum ada: dibuat NewLocalStorage
	s, err := storage.NewLocalStorage(dir)
	mustNoErr(t, err)
	t.Cleanup(func() { _ = s.Close() }) // ditutup sebelum t.TempDir() menghapus folder

	// Save membuat folder returns/ otomatis.
	mustNoErr(t, s.Save(t.Context(), "returns/a.png", strings.NewReader("isi foto")))
	got, err := os.ReadFile(filepath.Join(dir, "returns", "a.png"))
	mustNoErr(t, err)
	if string(got) != "isi foto" {
		t.Errorf("isi file = %q; want \"isi foto\"", got)
	}

	// File yang sudah ada tidak boleh tertimpa.
	if err := s.Save(t.Context(), "returns/a.png", strings.NewReader("lain")); err == nil {
		t.Error("Save ke path yang sudah ada = nil; want error (tidak menimpa file)")
	}

	mustNoErr(t, s.Delete(t.Context(), "returns/a.png"))
	if _, err := os.Stat(filepath.Join(dir, "returns", "a.png")); !os.IsNotExist(err) {
		t.Errorf("file masih ada setelah Delete (stat err = %v)", err)
	}
	// Menghapus file yang sudah tidak ada dianggap berhasil.
	mustNoErr(t, s.Delete(t.Context(), "returns/a.png"))
}

// os.Root menolak path yang keluar dari folder unggahan.
func TestLocalStorage_RejectsPathOutsideRoot(t *testing.T) {
	t.Parallel()
	base := t.TempDir()
	s, err := storage.NewLocalStorage(filepath.Join(base, "uploads"))
	mustNoErr(t, err)
	t.Cleanup(func() { _ = s.Close() })

	if err := s.Save(t.Context(), "../bocor.txt", strings.NewReader("x")); err == nil {
		t.Error("Save(\"../bocor.txt\") = nil; want error")
	}
	if _, err := os.Stat(filepath.Join(base, "bocor.txt")); !os.IsNotExist(err) {
		t.Error("file tertulis di luar folder uploads")
	}

	outside := filepath.Join(base, "penting.txt")
	mustNoErr(t, os.WriteFile(outside, []byte("jangan dihapus"), 0o644))
	if err := s.Delete(t.Context(), "../penting.txt"); err == nil {
		t.Error("Delete(\"../penting.txt\") = nil; want error")
	}
	if _, err := os.Stat(outside); err != nil {
		t.Errorf("file di luar folder uploads terhapus: %v", err)
	}
}

func TestLocalStorage_URL(t *testing.T) {
	t.Parallel()
	s, err := storage.NewLocalStorage(filepath.Join(t.TempDir(), "uploads"))
	mustNoErr(t, err)
	t.Cleanup(func() { _ = s.Close() })

	if got, want := s.URL("returns/a1.png"), "/uploads/returns/a1.png"; got != want {
		t.Errorf("URL = %q; want %q", got, want)
	}
}

// URL Cloudinary dibuat dari key tanpa memanggil internet: public_id = folder + key tanpa ekstensi,
// ekstensi menjadi format gambar di akhir URL.
func TestCloudinaryStorage_URL(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		folder string
		key    string
		want   string
	}{
		{"dengan folder", "sgeinventory", "returns/a1.png", "https://res.cloudinary.com/democloud/image/upload/sgeinventory/returns/a1.png"},
		{"garis miring di folder dirapikan", "/sgeinventory/", "returns/b2.jpg", "https://res.cloudinary.com/democloud/image/upload/sgeinventory/returns/b2.jpg"},
		{"tanpa folder", "", "returns/c3.webp", "https://res.cloudinary.com/democloud/image/upload/returns/c3.webp"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			s, err := storage.NewCloudinaryStorage("cloudinary://kunci:rahasia@democloud", tt.folder)
			mustNoErr(t, err)
			if got := s.URL(tt.key); got != tt.want {
				t.Errorf("URL(%q) = %q; want %q", tt.key, got, tt.want)
			}
		})
	}
}

// CLOUDINARY_URL yang salah ditolak, dan pesan errornya tidak boleh memuat secret.
func TestNewCloudinaryStorage_InvalidURL(t *testing.T) {
	t.Parallel()
	for _, raw := range []string{
		"",
		"https://kunci:RAHASIA@democloud",     // skema salah
		"cloudinary://kunci@democloud",        // tanpa secret
		"cloudinary://:RAHASIA@democloud",     // tanpa API key
		"cloudinary://kunci:RAHASIA@",         // tanpa nama cloud
		"cloudinary://kunci:RAHASIA@demo\x7f", // karakter tidak valid (url.Parse gagal)
	} {
		_, err := storage.NewCloudinaryStorage(raw, "sgeinventory")
		if err == nil {
			t.Errorf("NewCloudinaryStorage(%q) = nil; want error", raw)
			continue
		}
		if strings.Contains(err.Error(), "RAHASIA") {
			t.Errorf("pesan error membocorkan secret: %q", err)
		}
	}
}
