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

type consignmentRepository struct {
	db *gorm.DB
}

// NewConsignmentRepository membuat repository penitipan. Tipe kembaliannya interface contracts.ConsignmentRepository.
func NewConsignmentRepository(db *gorm.DB) contracts.ConsignmentRepository {
	return &consignmentRepository{db: db}
}

// Create menyimpan 3 hal dalam SATU transaksi (pola sama dengan borrowingRepository.Create):
//
//  1. INSERT consignments
//  2. INSERT items (otomatis oleh GORM karena consignment.Items terisi; items.consignment_id = id baru)
//  3. INSERT log_users (request_id = id penitipan yang baru)
//
// Jika salah satu gagal, semuanya dibatalkan (ROLLBACK), jadi tidak ada penitipan tanpa barang/log.
func (r *consignmentRepository) Create(ctx context.Context, consignment *entity.Consignment, activity *entity.UserLog) error {
	return conn(ctx, r.db).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(consignment).Error; err != nil {
			return fmt.Errorf("create consignment: %w", err)
		}

		activity.RequestID = consignment.ID // id baru terisi setelah INSERT di atas
		if err := tx.Create(activity).Error; err != nil {
			return fmt.Errorf("create consignment log: %w", err)
		}
		return nil
	})
}

// FindByID: penitip & lembaga tetap dimuat walau sudah dihapus (Unscoped), supaya riwayat tetap lengkap.
func (r *consignmentRepository) FindByID(ctx context.Context, id uint) (*entity.Consignment, error) {
	var consignment entity.Consignment
	err := conn(ctx, r.db).
		Preload("User", func(db *gorm.DB) *gorm.DB { return db.Unscoped() }).
		Preload("Lembaga", func(db *gorm.DB) *gorm.DB { return db.Unscoped() }).
		Preload("Items", func(db *gorm.DB) *gorm.DB { return db.Unscoped().Order("id ASC") }).
		First(&consignment, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errx.ErrConsignmentNotFound
		}
		return nil, fmt.Errorf("find consignment %d: %w", id, err)
	}
	return &consignment, nil
}
