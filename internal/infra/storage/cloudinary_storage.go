package storage

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/url"
	"path"
	"strings"
	"time"

	"github.com/cloudinary/cloudinary-go/v2"
	"github.com/cloudinary/cloudinary-go/v2/api"
	"github.com/cloudinary/cloudinary-go/v2/api/uploader"

	"SGEInventory_BE/domain/contracts"
)

// cloudinaryTimeout: batas waktu satu panggilan ke Cloudinary (unggah / hapus).
const cloudinaryTimeout = 30 * time.Second

// errInvalidCloudinaryURL sengaja TIDAK membungkus error asli: error url.Parse memuat URL lengkap,
// termasuk API secret, sehingga bisa bocor ke log.
var errInvalidCloudinaryURL = errors.New("CLOUDINARY_URL tidak valid, format: cloudinary://API_KEY:API_SECRET@CLOUD_NAME")

// CloudinaryStorage menyimpan file di Cloudinary (layanan penyimpanan gambar di internet).
//
// key "returns/8f3a...png" disimpan dengan public_id "<folder>/returns/8f3a..." (tanpa ekstensi, karena
// Cloudinary menyimpan formatnya sendiri) dan dibuka lewat
// https://res.cloudinary.com/<cloud>/image/upload/<folder>/returns/8f3a....png
type CloudinaryStorage struct {
	cld       *cloudinary.Cloudinary
	cloudName string
	folder    string // mis. "sgeinventory"; boleh kosong
}

var _ contracts.FileStorage = (*CloudinaryStorage)(nil) // memastikan CloudinaryStorage memenuhi interface

// NewCloudinaryStorage membuat storage dari CLOUDINARY_URL (cloudinary://API_KEY:API_SECRET@CLOUD_NAME).
// Tidak ada panggilan jaringan di sini; kredensial yang salah baru ketahuan saat unggah pertama.
func NewCloudinaryStorage(cloudinaryURL, folder string) (*CloudinaryStorage, error) {
	cloudName, err := cloudNameFromURL(cloudinaryURL)
	if err != nil {
		return nil, err
	}
	cld, err := cloudinary.NewFromURL(cloudinaryURL)
	if err != nil {
		return nil, errInvalidCloudinaryURL
	}
	return &CloudinaryStorage{cld: cld, cloudName: cloudName, folder: strings.Trim(folder, "/")}, nil
}

// cloudNameFromURL memastikan format CLOUDINARY_URL benar lalu mengambil nama cloud-nya.
func cloudNameFromURL(raw string) (string, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return "", errInvalidCloudinaryURL
	}
	secret, hasSecret := u.User.Password()
	if u.Scheme != "cloudinary" || u.Host == "" || u.User.Username() == "" || !hasSecret || secret == "" {
		return "", errInvalidCloudinaryURL
	}
	return u.Host, nil
}

// Save mengunggah isi r ke Cloudinary. Overwrite=false: asset yang sudah ada tidak pernah tertimpa.
// AllowedFormats: Cloudinary ikut menolak file selain JPG/PNG/WebP (lapis kedua setelah cek di service).
func (s *CloudinaryStorage) Save(ctx context.Context, key string, r io.Reader) error {
	ctx, cancel := context.WithTimeout(ctx, cloudinaryTimeout)
	defer cancel()

	publicID := s.publicID(key)
	res, err := s.cld.Upload.Upload(ctx, r, uploader.UploadParams{
		PublicID:       publicID,
		AssetFolder:    path.Dir(publicID), // folder yang terlihat di Media Library Cloudinary
		ResourceType:   "image",
		Overwrite:      api.Bool(false),
		UniqueFilename: api.Bool(false),
		AllowedFormats: api.CldAPIArray{"jpg", "png", "webp"},
	})
	if err != nil {
		return fmt.Errorf("cloudinary upload %s: %w", key, err)
	}
	// SDK bisa membalas err == nil tetapi menaruh pesan error di response.
	if res.Error.Message != "" {
		return fmt.Errorf("cloudinary upload %s: %s", key, res.Error.Message)
	}
	return nil
}

// Delete menghapus asset dari Cloudinary. Invalidate=true juga membersihkan salinan di cache CDN supaya foto
// yang ditolak tidak bisa dibuka lagi. Asset yang sudah tidak ada ("not found") dianggap berhasil.
func (s *CloudinaryStorage) Delete(ctx context.Context, key string) error {
	ctx, cancel := context.WithTimeout(ctx, cloudinaryTimeout)
	defer cancel()

	res, err := s.cld.Upload.Destroy(ctx, uploader.DestroyParams{
		PublicID:     s.publicID(key),
		ResourceType: "image",
		Invalidate:   api.Bool(true),
	})
	if err != nil {
		return fmt.Errorf("cloudinary destroy %s: %w", key, err)
	}
	if res.Error.Message != "" {
		return fmt.Errorf("cloudinary destroy %s: %s", key, res.Error.Message)
	}
	if res.Result != "ok" && res.Result != "not found" {
		return fmt.Errorf("cloudinary destroy %s: result %q", key, res.Result)
	}
	return nil
}

// URL membuat alamat publik foto tanpa memanggil Cloudinary, mis.
// "returns/a1.png" → "https://res.cloudinary.com/<cloud>/image/upload/sgeinventory/returns/a1.png".
func (s *CloudinaryStorage) URL(key string) string {
	return "https://res.cloudinary.com/" + s.cloudName + "/image/upload/" + s.publicID(key) + path.Ext(key)
}

// publicID: "returns/a1.png" → "sgeinventory/returns/a1" (folder + key tanpa ekstensi).
func (s *CloudinaryStorage) publicID(key string) string {
	id := strings.TrimSuffix(key, path.Ext(key))
	if s.folder == "" {
		return id
	}
	return s.folder + "/" + id
}
