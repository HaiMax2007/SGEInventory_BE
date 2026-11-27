package repository

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	"SGEInventory_BE/domain/contracts"
	"SGEInventory_BE/domain/entity"
)

type userLogRepository struct {
	db *gorm.DB
}

// NewUserLogRepository membuat repository log aktivitas. Tipe kembaliannya interface contracts.UserLogRepository.
func NewUserLogRepository(db *gorm.DB) contracts.UserLogRepository {
	return &userLogRepository{db: db}
}

// FindByRequest mengambil riwayat sebuah pengajuan, urut dari yang paling lama.
// Pelaku (User) ikut dimuat, termasuk user yang sudah dihapus (Unscoped) agar riwayat tetap lengkap.
func (r *userLogRepository) FindByRequest(ctx context.Context, requestType string, requestID uint) ([]entity.UserLog, error) {
	logs := []entity.UserLog{}
	err := conn(ctx, r.db).
		Preload("User", func(db *gorm.DB) *gorm.DB { return db.Unscoped() }).
		Where("request_type = ? AND request_id = ?", requestType, requestID).
		Order("created_at ASC, id ASC").
		Find(&logs).Error
	if err != nil {
		return nil, fmt.Errorf("find logs %s %d: %w", requestType, requestID, err)
	}
	return logs, nil
}

// FindAll mengambil satu halaman semua log, terbaru dulu (kebalikan dari timeline yang urut dari yang lama).
func (r *userLogRepository) FindAll(ctx context.Context, requestType, search string, offset, limit int) ([]entity.UserLog, int64, error) {
	db := conn(ctx, r.db)
	query := db.Model(&entity.UserLog{})
	if requestType != "" {
		query = query.Where("request_type = ?", requestType)
	}
	if search != "" {
		// WHERE user_id IN (SELECT id FROM users WHERE full_name ILIKE '%search%')
		// Subquery (bukan JOIN) supaya Count & Preload tetap sederhana. Unscoped: pelaku yang sudah dihapus
		// tetap bisa dicari, sama seperti Preload di bawah yang tetap menampilkannya.
		actors := db.Unscoped().Model(&entity.User{}).Select("id").Where("full_name ILIKE ?", "%"+search+"%")
		query = query.Where("user_id IN (?)", actors)
	}
	return r.paginate(query, offset, limit)
}

// FindByUser: log yang pelakunya userID (index idx_log_users_user_id), terbaru dulu.
func (r *userLogRepository) FindByUser(ctx context.Context, userID uint, requestType string, offset, limit int) ([]entity.UserLog, int64, error) {
	query := conn(ctx, r.db).Model(&entity.UserLog{}).Where("user_id = ?", userID)
	if requestType != "" {
		query = query.Where("request_type = ?", requestType)
	}
	return r.paginate(query, offset, limit)
}

// paginate menghitung total lalu mengambil satu halaman log (terbaru dulu) beserta pelakunya.
// Dipakai bersama FindAll dan FindByUser; yang berbeda hanya WHERE-nya.
func (r *userLogRepository) paginate(query *gorm.DB, offset, limit int) ([]entity.UserLog, int64, error) {
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count logs: %w", err)
	}

	logs := []entity.UserLog{}
	err := query.
		Preload("User", func(db *gorm.DB) *gorm.DB { return db.Unscoped() }).
		Order("id DESC").Offset(offset).Limit(limit).
		Find(&logs).Error
	if err != nil {
		return nil, 0, fmt.Errorf("list logs: %w", err)
	}
	return logs, total, nil
}

func (r *userLogRepository) Create(ctx context.Context, log *entity.UserLog) error {
	if err := conn(ctx, r.db).Create(log).Error; err != nil {
		return fmt.Errorf("create log: %w", err)
	}
	return nil
}

// CreateMany: GORM membuat satu INSERT berisi banyak baris (VALUES (...), (...), ...).
func (r *userLogRepository) CreateMany(ctx context.Context, logs []entity.UserLog) error {
	if len(logs) == 0 {
		return nil
	}
	if err := conn(ctx, r.db).Create(&logs).Error; err != nil {
		return fmt.Errorf("create logs: %w", err)
	}
	return nil
}
