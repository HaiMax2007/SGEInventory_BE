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
	if err := r.db.WithContext(ctx).Create(item).Error; err != nil {
		return fmt.Errorf("create item: %w", err)
	}
	return nil
}

func (r *itemRepository) FindByID(ctx context.Context, id uint) (*entity.Item, error) {
	var item entity.Item
	// Preload("Lembaga"): GORM menjalankan query kedua ke tabel lembaga dan mengisi item.Lembaga.
	if err := r.db.WithContext(ctx).Preload("Lembaga").First(&item, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errx.ErrItemNotFound
		}
		return nil, fmt.Errorf("find item %d: %w", id, err)
	}
	return &item, nil
}

// FindAll mengembalikan satu halaman barang + jumlah seluruh barang yang cocok dengan filter.
func (r *itemRepository) FindAll(ctx context.Context, search string, onlyAvailable bool, offset, limit int) ([]entity.Item, int64, error) {
	query := r.db.WithContext(ctx).Model(&entity.Item{})
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
	err := r.db.WithContext(ctx).Model(item).
		Select("name", "description", "lembaga_id", "total_quantity", "available_quantity").Updates(item).Error
	if err != nil {
		return fmt.Errorf("update item %d: %w", item.ID, err)
	}
	return nil
}

// Delete melakukan soft delete (baris tetap ada, deleted_at terisi).
func (r *itemRepository) Delete(ctx context.Context, item *entity.Item) error {
	if err := r.db.WithContext(ctx).Delete(item).Error; err != nil {
		return fmt.Errorf("delete item %d: %w", item.ID, err)
	}
	return nil
}
