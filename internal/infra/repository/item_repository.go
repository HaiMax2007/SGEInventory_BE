package repository

import (
	"context"
	"errors"
	"fmt"

	"gorm.io/gorm"

	"SGEInventory_BE/domain/contracts"
	"SGEInventory_BE/domain/entity"
	"SGEInventory_BE/domain/errx"
)

type itemRepository struct {
	db *gorm.DB
}

// NewItemRepository membuat repository barang. Tipe kembaliannya interface contracts.ItemRepository.
func NewItemRepository(db *gorm.DB) contracts.ItemRepository {
	return &itemRepository{db: db}
}

func (r *itemRepository) Create(ctx context.Context, item *entity.Item) error {
	if err := conn(ctx, r.db).Create(item).Error; err != nil {
		return fmt.Errorf("create item: %w", err)
	}
	return nil
}

func (r *itemRepository) FindByID(ctx context.Context, id uint) (*entity.Item, error) {
	var item entity.Item
	// Preload("Lembaga"): GORM menjalankan query kedua ke tabel lembaga dan mengisi item.Lembaga.
	if err := conn(ctx, r.db).Preload("Lembaga").First(&item, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errx.ErrItemNotFound
		}
		return nil, fmt.Errorf("find item %d: %w", id, err)
	}
	return &item, nil
}

// FindByIDs mengambil banyak barang dengan satu query: SELECT * FROM items WHERE id IN (...).
// Barang yang sudah dihapus (soft delete) otomatis tidak ikut.
func (r *itemRepository) FindByIDs(ctx context.Context, ids []uint) ([]entity.Item, error) {
	items := []entity.Item{}
	if err := conn(ctx, r.db).Where("id IN ?", ids).Find(&items).Error; err != nil {
		return nil, fmt.Errorf("find items by ids: %w", err)
	}
	return items, nil
}

// FindAll mengembalikan satu halaman barang + jumlah seluruh barang yang cocok dengan filter.
func (r *itemRepository) FindAll(ctx context.Context, search string, onlyAvailable bool, offset, limit int) ([]entity.Item, int64, error) {
	query := conn(ctx, r.db).Model(&entity.Item{})
	if search != "" {
		// ILIKE = LIKE yang tidak membedakan huruf besar/kecil (di Postgres, LIKE membedakannya).
		like := "%" + search + "%"
		query = query.Where("name ILIKE ? OR description ILIKE ?", like, like)
	}
	if onlyAvailable {
		query = query.Where("available_quantity > 0")
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count items: %w", err)
	}

	items := []entity.Item{}
	// Preload diletakkan setelah Count: Count tidak butuh data lembaga.
	// Untuk satu halaman, GORM cukup satu query tambahan: SELECT * FROM lembaga WHERE id IN (...).
	if err := query.Preload("Lembaga").Order("id DESC").Offset(offset).Limit(limit).Find(&items).Error; err != nil {
		return nil, 0, fmt.Errorf("list items: %w", err)
	}
	return items, total, nil
}

func (r *itemRepository) Update(ctx context.Context, item *entity.Item) error {
	// Select menentukan kolom yang disimpan, termasuk jika nilainya 0 atau teks kosong "".
	// Kolom baru yang boleh diubah lewat update WAJIB ditambahkan di sini, kalau tidak GORM tidak menyimpannya.
	err := conn(ctx, r.db).Model(item).
		Select("name", "description", "lembaga_id", "total_quantity", "available_quantity").Updates(item).Error
	if err != nil {
		return fmt.Errorf("update item %d: %w", item.ID, err)
	}
	return nil
}

// Delete melakukan soft delete (baris tetap ada, deleted_at terisi).
func (r *itemRepository) Delete(ctx context.Context, item *entity.Item) error {
	if err := conn(ctx, r.db).Delete(item).Error; err != nil {
		return fmt.Errorf("delete item %d: %w", item.ID, err)
	}
	return nil
}

// FindByIDForUpdate: SELECT * FROM items WHERE id = ? AND deleted_at IS NULL FOR UPDATE.
// Tanpa Preload: kunci cukup di baris barang, data lembaga tidak dibutuhkan untuk mengubah stok.
func (r *itemRepository) FindByIDForUpdate(ctx context.Context, id uint) (*entity.Item, error) {
	var item entity.Item
	if err := conn(ctx, r.db).Clauses(forUpdate).First(&item, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errx.ErrItemNotFound
		}
		return nil, fmt.Errorf("lock item %d: %w", id, err)
	}
	return &item, nil
}

// FindByIDsForUpdate: SELECT * FROM items WHERE id IN (...) ORDER BY id FOR UPDATE.
// ORDER BY id penting: semua transaksi mengunci barang dengan urutan yang sama, jadi dua transaksi tidak
// bisa saling menunggu barang yang dipegang yang lain (deadlock).
func (r *itemRepository) FindByIDsForUpdate(ctx context.Context, ids []uint) ([]entity.Item, error) {
	items := []entity.Item{}
	err := conn(ctx, r.db).Clauses(forUpdate).Where("id IN ?", ids).Order("id ASC").Find(&items).Error
	if err != nil {
		return nil, fmt.Errorf("lock items: %w", err)
	}
	return items, nil
}
