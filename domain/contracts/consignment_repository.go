package contracts

import (
	"context"

	"SGEInventory_BE/domain/entity"
)

// ConsignmentRepository adalah "janji" cara mengakses data penitipan di database.
// Implementasinya ada di internal/infra/repository/consignment_repository.go (GORM).
type ConsignmentRepository interface {
	// Create menyimpan penitipan + barang-barangnya (consignment.Items) + log dalam SATU transaksi.
	// activity.RequestID diisi id penitipan yang baru.
	Create(ctx context.Context, consignment *entity.Consignment, activity *entity.UserLog) error
	// FindByID mengambil satu penitipan beserta penitip, lembaga, dan barang-barangnya.
	FindByID(ctx context.Context, id uint) (*entity.Consignment, error)
	// FindByUser mengambil satu halaman penitipan milik seorang user (terbaru dulu) + jumlah seluruhnya.
	FindByUser(ctx context.Context, userID uint, offset, limit int) ([]entity.Consignment, int64, error)
	// FindAll mengambil satu halaman SEMUA penitipan (terbaru dulu) + jumlah seluruhnya.
	// status kosong = semua status.
	FindAll(ctx context.Context, status string, offset, limit int) ([]entity.Consignment, int64, error)
	// Count menghitung jumlah SEMUA penitipan (semua status).
	Count(ctx context.Context) (int64, error)

	// Dipakai di dalam Transactor.WithinTransaction (ACC / tolak).

	// FindByIDForUpdate mengambil & mengunci satu penitipan (tanpa relasi) sampai transaksi selesai.
	FindByIDForUpdate(ctx context.Context, id uint) (*entity.Consignment, error)
	// UpdateStatus menyimpan status & kolom foto.
	UpdateStatus(ctx context.Context, consignment *entity.Consignment) error
	// DeleteItems menghapus PERMANEN semua barang titipan milik penitipan ini (dipakai saat ditolak).
	DeleteItems(ctx context.Context, consignmentID uint) error
}
