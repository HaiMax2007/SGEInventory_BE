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

type lembagaRepository struct {
	db *gorm.DB
}

// NewLembagaRepository membuat repository lembaga. Tipe kembaliannya interface contracts.LembagaRepository.
func NewLembagaRepository(db *gorm.DB) contracts.LembagaRepository {
	return &lembagaRepository{db: db}
}

func (r *lembagaRepository) Create(ctx context.Context, lembaga *entity.Lembaga) error {
	if err := r.db.WithContext(ctx).Create(lembaga).Error; err != nil {
		// Satu-satunya kolom unik di tabel lembaga adalah name (uq_lembaga_name).
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return errx.ErrLembagaNameTaken
		}
		return fmt.Errorf("create lembaga: %w", err)
	}
	return nil
}

func (r *lembagaRepository) FindByID(ctx context.Context, id uint) (*entity.Lembaga, error) {
	var lembaga entity.Lembaga
	if err := r.db.WithContext(ctx).First(&lembaga, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errx.ErrLembagaNotFound
		}
		return nil, fmt.Errorf("find lembaga %d: %w", id, err)
	}
	return &lembaga, nil
}

func (r *lembagaRepository) FindAll(ctx context.Context, search string, offset, limit int) ([]entity.Lembaga, int64, error) {
	query := r.db.WithContext(ctx).Model(&entity.Lembaga{})
	if search != "" {
		query = query.Where("name ILIKE ?", "%"+search+"%")
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count lembaga: %w", err)
	}

	// Diurutkan berdasarkan nama (A-Z) supaya enak dipakai FE sebagai pilihan dropdown.
	lembagas := []entity.Lembaga{}
	if err := query.Order("name ASC").Offset(offset).Limit(limit).Find(&lembagas).Error; err != nil {
		return nil, 0, fmt.Errorf("list lembaga: %w", err)
	}
	return lembagas, total, nil
}

func (r *lembagaRepository) Update(ctx context.Context, lembaga *entity.Lembaga) error {
	// Select menentukan kolom yang disimpan. Kolom baru yang bisa di-update wajib ditambahkan di sini.
	err := r.db.WithContext(ctx).Model(lembaga).Select("name").Updates(lembaga).Error
	if err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return errx.ErrLembagaNameTaken
		}
		return fmt.Errorf("update lembaga %d: %w", lembaga.ID, err)
	}
	return nil
}

func (r *lembagaRepository) CountItems(ctx context.Context, id uint) (int64, error) {
	var total int64
	// Model(&entity.Item{}) otomatis menambah "deleted_at IS NULL", jadi barang yang sudah dihapus tidak dihitung.
	if err := r.db.WithContext(ctx).Model(&entity.Item{}).Where("lembaga_id = ?", id).Count(&total).Error; err != nil {
		return 0, fmt.Errorf("count items of lembaga %d: %w", id, err)
	}
	return total, nil
}

func (r *lembagaRepository) Delete(ctx context.Context, lembaga *entity.Lembaga) error {
	if err := r.db.WithContext(ctx).Delete(lembaga).Error; err != nil {
		return fmt.Errorf("delete lembaga %d: %w", lembaga.ID, err)
	}
	return nil
}
