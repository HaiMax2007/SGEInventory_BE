package contracts

import (
	"context"
	"io"
)

// FileStorage adalah "janji" tempat menyimpan file unggahan (mis. foto pengembalian).
// Ada dua implementasi di internal/infra/storage:
//   - CloudinaryStorage: dipakai jika env CLOUDINARY_URL diisi (wajib di production)
//   - LocalStorage: folder di server, cadangan di development jika CLOUDINARY_URL kosong
//
// Karena service hanya mengenal interface ini, tempat penyimpanan bisa diganti tanpa mengubah service.
//
// key selalu relatif, mis. "returns/8f3a...jpg". Key inilah yang disimpan di database (bukan URL),
// sehingga URL publik selalu dibuat oleh storage yang sedang dipakai.
type FileStorage interface {
	// Save menyimpan isi r dengan nama key. Gagal jika key yang sama sudah ada.
	Save(ctx context.Context, key string, r io.Reader) error
	// Delete menghapus file key. File yang sudah tidak ada dianggap berhasil dihapus.
	Delete(ctx context.Context, key string) error
	// URL mengubah key menjadi alamat yang bisa dibuka FE, mis. "https://res.cloudinary.com/...".
	URL(key string) string
}
