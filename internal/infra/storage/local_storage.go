// Package storage berisi implementasi contracts.FileStorage.
package storage

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"

	"SGEInventory_BE/domain/contracts"
)

// LocalStorage menyimpan file di sebuah folder di server (mis. "uploads").
//
// Semua akses lewat os.Root: path seperti "../../etc/passwd" otomatis ditolak karena keluar dari folder
// unggahan. Jadi walaupun suatu saat path berasal dari luar, file di luar folder tidak bisa disentuh.
type LocalStorage struct {
	root *os.Root
}

// NewLocalStorage membuat folder dir (jika belum ada) lalu membukanya sebagai root.
func NewLocalStorage(dir string) (*LocalStorage, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("create upload dir: %w", err)
	}
	root, err := os.OpenRoot(dir)
	if err != nil {
		return nil, fmt.Errorf("open upload dir: %w", err)
	}
	return &LocalStorage{root: root}, nil
}

var _ contracts.FileStorage = (*LocalStorage)(nil) // memastikan LocalStorage memenuhi interface

// Close menutup folder unggahan. Server memakainya sampai mati, jadi Close terutama dipakai tes:
// di Windows, folder yang masih terbuka tidak bisa dihapus.
func (s *LocalStorage) Close() error {
	return s.root.Close()
}

// Save menulis isi r ke file path (folder induknya dibuat otomatis). O_EXCL: gagal jika file sudah ada,
// supaya file lain tidak pernah tertimpa. Jika penulisan gagal di tengah jalan, file setengah jadi dihapus.
func (s *LocalStorage) Save(filePath string, r io.Reader) (err error) {
	if err := s.root.MkdirAll(path.Dir(filePath), 0o755); err != nil {
		return fmt.Errorf("create dir for %s: %w", filePath, err)
	}
	f, err := s.root.OpenFile(filePath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("create file %s: %w", filePath, err)
	}
	defer func() {
		if closeErr := f.Close(); err == nil && closeErr != nil {
			err = fmt.Errorf("close file %s: %w", filePath, closeErr)
		}
		if err != nil {
			_ = s.root.Remove(filePath) // jangan tinggalkan file rusak
		}
	}()

	if _, err := io.Copy(f, r); err != nil {
		return fmt.Errorf("write file %s: %w", filePath, err)
	}
	return nil
}

// Delete menghapus file path. File yang sudah tidak ada dianggap berhasil.
func (s *LocalStorage) Delete(filePath string) error {
	if err := s.root.Remove(filePath); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("delete file %s: %w", filePath, err)
	}
	return nil
}
