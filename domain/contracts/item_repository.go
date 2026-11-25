package contracts

import (
	"context"

	"SGEInventory_BE/domain/entity"
)

// ItemRepository adalah "janji" cara mengakses data barang di database.
// Implementasinya ada di internal/infra/repository/item_repository.go (GORM).
type ItemRepository interface {
	Create(ctx context.Context, item *entity.Item) error
	FindByID(ctx context.Context, id uint) (*entity.Item, error)
	// FindByIDs mengambil banyak barang sekaligus (yang belum dihapus). Id yang tidak ada tidak ikut di hasil.
	FindByIDs(ctx context.Context, ids []uint) ([]entity.Item, error)
	// FindAll: itemType kosong = semua tipe barang; diisi (BORROWABLE / CONSIGNMENT) = hanya tipe itu.
	FindAll(ctx context.Context, search string, onlyAvailable bool, itemType string, offset, limit int) ([]entity.Item, int64, error)
	Update(ctx context.Context, item *entity.Item) error
	Delete(ctx context.Context, item *entity.Item) error

	// Dipakai di dalam Transactor.WithinTransaction: baris yang dibaca dikunci sampai transaksi selesai,
	// supaya stok tidak bisa diubah transaksi lain di antara "baca" dan "simpan".

	// FindByIDForUpdate mengambil & mengunci satu barang (tanpa relasi lembaga).
	FindByIDForUpdate(ctx context.Context, id uint) (*entity.Item, error)
	// FindByIDsForUpdate mengambil & mengunci banyak barang, URUT id (urutan tetap mencegah deadlock).
	// Barang yang sudah dihapus tidak ikut di hasil.
	FindByIDsForUpdate(ctx context.Context, ids []uint) ([]entity.Item, error)
}
