package contracts

import "io"

// FileStorage adalah "janji" tempat menyimpan file unggahan (mis. foto pengembalian).
// Implementasinya sekarang folder di server (internal/infra/storage/local_storage.go). Karena service hanya
// mengenal interface ini, penyimpanan bisa diganti (mis. ke cloud) tanpa mengubah service.
//
// path selalu relatif terhadap folder unggahan, mis. "returns/8f3a...jpg".
type FileStorage interface {
	// Save menulis isi r ke path. Gagal jika file dengan path yang sama sudah ada.
	Save(path string, r io.Reader) error
	// Delete menghapus file di path. File yang sudah tidak ada dianggap berhasil dihapus.
	Delete(path string) error
}
