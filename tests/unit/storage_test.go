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

	// Save membuat folder returns/ otomatis.
	mustNoErr(t, s.Save("returns/a.png", strings.NewReader("isi foto")))
	got, err := os.ReadFile(filepath.Join(dir, "returns", "a.png"))
	mustNoErr(t, err)
	if string(got) != "isi foto" {
		t.Errorf("isi file = %q; want \"isi foto\"", got)
	}

	// File yang sudah ada tidak boleh tertimpa.
	if err := s.Save("returns/a.png", strings.NewReader("lain")); err == nil {
		t.Error("Save ke path yang sudah ada = nil; want error (tidak menimpa file)")
	}

	mustNoErr(t, s.Delete("returns/a.png"))
	if _, err := os.Stat(filepath.Join(dir, "returns", "a.png")); !os.IsNotExist(err) {
		t.Errorf("file masih ada setelah Delete (stat err = %v)", err)
	}
	// Menghapus file yang sudah tidak ada dianggap berhasil.
	mustNoErr(t, s.Delete("returns/a.png"))
}

// os.Root menolak path yang keluar dari folder unggahan.
func TestLocalStorage_RejectsPathOutsideRoot(t *testing.T) {
	t.Parallel()
	base := t.TempDir()
	s, err := storage.NewLocalStorage(filepath.Join(base, "uploads"))
	mustNoErr(t, err)

	if err := s.Save("../bocor.txt", strings.NewReader("x")); err == nil {
		t.Error("Save(\"../bocor.txt\") = nil; want error")
	}
	if _, err := os.Stat(filepath.Join(base, "bocor.txt")); !os.IsNotExist(err) {
		t.Error("file tertulis di luar folder uploads")
	}

	outside := filepath.Join(base, "penting.txt")
	mustNoErr(t, os.WriteFile(outside, []byte("jangan dihapus"), 0o644))
	if err := s.Delete("../penting.txt"); err == nil {
		t.Error("Delete(\"../penting.txt\") = nil; want error")
	}
	if _, err := os.Stat(outside); err != nil {
		t.Errorf("file di luar folder uploads terhapus: %v", err)
	}
}
