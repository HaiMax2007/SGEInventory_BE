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

func (r *consignmentRepository) FindByID(ctx context.Context, id uint) (*entity.Consignment, error) {
	var consignment entity.Consignment
	if err := r.withDetails(conn(ctx, r.db)).First(&consignment, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errx.ErrConsignmentNotFound
		}
		return nil, fmt.Errorf("find consignment %d: %w", id, err)
	}
	return &consignment, nil
}

func (r *consignmentRepository) FindByUser(ctx context.Context, userID uint, offset, limit int) ([]entity.Consignment, int64, error) {
	query := conn(ctx, r.db).Model(&entity.Consignment{}).Where("user_id = ?", userID)
	return r.paginate(query, offset, limit)
}

func (r *consignmentRepository) FindAll(ctx context.Context, status string, offset, limit int) ([]entity.Consignment, int64, error) {
	query := conn(ctx, r.db).Model(&entity.Consignment{})
	if status != "" {
		query = query.Where("status = ?", status)
	}
	return r.paginate(query, offset, limit)
}

// Count: SELECT count(*) FROM consignments.
func (r *consignmentRepository) Count(ctx context.Context) (int64, error) {
	var total int64
	if err := conn(ctx, r.db).Model(&entity.Consignment{}).Count(&total).Error; err != nil {
		return 0, fmt.Errorf("count consignments: %w", err)
	}
	return total, nil
}

// FindByIDForUpdate: SELECT * FROM consignments WHERE id = ? FOR UPDATE (tanpa relasi).
// Dua admin yang memproses penitipan yang sama bergantian: yang kedua menunggu, lalu melihat status terbaru.
func (r *consignmentRepository) FindByIDForUpdate(ctx context.Context, id uint) (*entity.Consignment, error) {
	var consignment entity.Consignment
	if err := conn(ctx, r.db).Clauses(forUpdate).First(&consignment, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errx.ErrConsignmentNotFound
		}
		return nil, fmt.Errorf("lock consignment %d: %w", id, err)
	}
	return &consignment, nil
}

// UpdateStatus: UPDATE consignments SET status = ?, photo = ?, updated_at = NOW() WHERE id = ?.
// Memakai map supaya photo "" (setelah ditolak) benar-benar disimpan (Updates dengan struct melewati nilai kosong).
func (r *consignmentRepository) UpdateStatus(ctx context.Context, consignment *entity.Consignment) error {
	err := conn(ctx, r.db).Model(consignment).Updates(map[string]any{
		"status": consignment.Status,
		"photo":  consignment.Photo,
	}).Error
	if err != nil {
		return fmt.Errorf("update consignment %d status: %w", consignment.ID, err)
	}
	return nil
}

// DeleteItems: DELETE FROM items WHERE consignment_id = ? (hapus PERMANEN, Unscoped = tanpa soft delete).
// Aman karena barang titipan tidak bisa dipinjam, jadi tidak dirujuk borrowing_items.
func (r *consignmentRepository) DeleteItems(ctx context.Context, consignmentID uint) error {
	err := conn(ctx, r.db).Unscoped().Where("consignment_id = ?", consignmentID).Delete(&entity.Item{}).Error
	if err != nil {
		return fmt.Errorf("delete items of consignment %d: %w", consignmentID, err)
	}
	return nil
}

// paginate menghitung total lalu mengambil satu halaman (terbaru dulu) beserta detailnya.
func (r *consignmentRepository) paginate(query *gorm.DB, offset, limit int) ([]entity.Consignment, int64, error) {
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count consignments: %w", err)
	}

	consignments := []entity.Consignment{}
	err := r.withDetails(query).Order("id DESC").Offset(offset).Limit(limit).Find(&consignments).Error
	if err != nil {
		return nil, 0, fmt.Errorf("list consignments: %w", err)
	}
	return consignments, total, nil
}

// withDetails ikut memuat penitip, lembaga & barang titipan. Unscoped: data yang sudah dihapus (soft delete)
// tetap ditampilkan, supaya riwayat penitipan lama tetap lengkap.
func (r *consignmentRepository) withDetails(query *gorm.DB) *gorm.DB {
	return query.
		Preload("User", func(db *gorm.DB) *gorm.DB { return db.Unscoped() }).
		Preload("Lembaga", func(db *gorm.DB) *gorm.DB { return db.Unscoped() }).
		Preload("Items", func(db *gorm.DB) *gorm.DB { return db.Unscoped().Order("id ASC") })
}
