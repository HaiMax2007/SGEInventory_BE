// Package photo berisi aturan foto unggahan yang dipakai bersama (foto pengembalian, foto penitipan):
// cek ukuran & jenis dari isi file, nama acak, simpan & hapus lewat contracts.FileStorage.
package photo

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"

	"SGEInventory_BE/domain/contracts"
)

// MaxBytes: batas ukuran satu foto (5 MB). Fiber juga membatasi seluruh body request (BodyLimit di main.go)
// sedikit di atas angka ini.
const MaxBytes = 5 << 20

// ErrInvalid: foto kosong, terlalu besar, atau bukan JPG/PNG/WebP. Pemanggil mengubahnya menjadi errx miliknya
// sendiri (mis. errx.ErrInvalidReturnPhoto), supaya pesan ke user sesuai form-nya.
var ErrInvalid = errors.New("photo: invalid")

// extensions: jenis gambar yang diterima → ekstensi file yang disimpan.
// Jenis dibaca dari ISI file (http.DetectContentType), bukan dari nama file kiriman user,
// supaya file lain yang diganti namanya menjadi .jpg tetap ditolak.
var extensions = map[string]string{
	"image/jpeg": ".jpg",
	"image/png":  ".png",
	"image/webp": ".webp",
}

// Extension memastikan foto berukuran 1 byte – 5 MB dan isinya JPG/PNG/WebP, lalu mengembalikan ekstensi
// file yang akan dipakai (mis. ".png"). Foto yang tidak memenuhi aturan → ErrInvalid.
func Extension(fh *multipart.FileHeader) (string, error) {
	if fh == nil || fh.Size <= 0 || fh.Size > MaxBytes {
		return "", ErrInvalid
	}

	f, err := fh.Open()
	if err != nil {
		return "", fmt.Errorf("open uploaded photo: %w", err)
	}
	defer f.Close()

	// DetectContentType cukup membaca 512 byte pertama ("tanda tangan" format file).
	head := make([]byte, 512)
	n, err := io.ReadFull(f, head)
	if err != nil && !errors.Is(err, io.ErrUnexpectedEOF) {
		return "", fmt.Errorf("read uploaded photo: %w", err)
	}

	ext, ok := extensions[http.DetectContentType(head[:n])]
	if !ok {
		return "", ErrInvalid
	}
	return ext, nil
}

// NewKey membuat key acak di folder, mis. NewKey("returns", ".jpg") → "returns/8f3a9c...e1.jpg"
// (32 karakter hex dari crypto/rand). Nama acak tidak bisa ditebak, jadi foto orang lain tidak bisa dicari
// dari URL walaupun URL foto bersifat publik.
func NewKey(folder, ext string) (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate photo name: %w", err)
	}
	return folder + "/" + hex.EncodeToString(b) + ext, nil
}

// Save menyalin isi file unggahan ke storage dengan nama key.
func Save(ctx context.Context, st contracts.FileStorage, fh *multipart.FileHeader, key string) error {
	f, err := fh.Open()
	if err != nil {
		return fmt.Errorf("open uploaded photo: %w", err)
	}
	defer f.Close()
	return st.Save(ctx, key, f)
}

// Delete menghapus foto. Gagal menghapus tidak membatalkan request (data di database sudah benar, dan file
// yatim tidak berbahaya), jadi error cukup dicatat di log.
//
// context.WithoutCancel: penghapusan tetap dijalankan walaupun client sudah menutup koneksi, karena data di
// database sudah terlanjur berubah. Batas waktunya diatur oleh storage (mis. 30 detik untuk Cloudinary).
func Delete(ctx context.Context, st contracts.FileStorage, key string) {
	if err := st.Delete(context.WithoutCancel(ctx), key); err != nil {
		log.Println("delete photo:", err)
	}
}
