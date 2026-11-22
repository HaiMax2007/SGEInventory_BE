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

type borrowingRepository struct {
	db *gorm.DB
}

// NewBorrowingRepository membuat repository peminjaman. Tipe kembaliannya interface contracts.BorrowingRepository.
func NewBorrowingRepository(db *gorm.DB) contracts.BorrowingRepository {
	return &borrowingRepository{db: db}
}

// Create menyimpan 3 hal dalam SATU transaksi:
//
//  1. INSERT borrowings (header)
//  2. INSERT borrowing_items (otomatis oleh GORM karena borrowing.Items terisi / relasi has-many)
//  3. INSERT log_users (aktivitas, request_id = id header yang baru)
//
// Jika salah satu gagal, semuanya dibatalkan (ROLLBACK), jadi tidak ada pengajuan tanpa barang/log.
// db.Transaction otomatis COMMIT jika fungsi mengembalikan nil, dan ROLLBACK jika mengembalikan error.
func (r *borrowingRepository) Create(ctx context.Context, borrowing *entity.Borrowing, activity *entity.UserLog) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Semua query di dalam fungsi ini WAJIB memakai tx (bukan r.db) agar ikut transaksi.
		if err := tx.Create(borrowing).Error; err != nil {
			return fmt.Errorf("create borrowing: %w", err)
		}

		activity.RequestID = borrowing.ID // id baru terisi setelah INSERT di atas
		if err := tx.Create(activity).Error; err != nil {
			return fmt.Errorf("create borrowing log: %w", err)
		}
		return nil
	})
}

func (r *borrowingRepository) FindByID(ctx context.Context, id uint) (*entity.Borrowing, error) {
	var borrowing entity.Borrowing
	err := r.withDetails(r.db.WithContext(ctx)).First(&borrowing, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errx.ErrBorrowingNotFound
		}
		return nil, fmt.Errorf("find borrowing %d: %w", id, err)
	}
	return &borrowing, nil
}

func (r *borrowingRepository) FindByUser(ctx context.Context, userID uint, offset, limit int) ([]entity.Borrowing, int64, error) {
	query := r.db.WithContext(ctx).Model(&entity.Borrowing{}).Where("user_id = ?", userID)
	return r.paginate(query, offset, limit)
}

func (r *borrowingRepository) FindAll(ctx context.Context, status string, offset, limit int) ([]entity.Borrowing, int64, error) {
	query := r.db.WithContext(ctx).Model(&entity.Borrowing{})
	if status != "" {
		query = query.Where("status = ?", status)
	}
	return r.paginate(query, offset, limit)
}

// paginate menghitung jumlah data yang cocok dengan query, lalu mengambil satu halaman (terbaru dulu)
// beserta detailnya. Dipakai bersama oleh FindByUser dan FindAll; yang berbeda hanya WHERE-nya.
func (r *borrowingRepository) paginate(query *gorm.DB, offset, limit int) ([]entity.Borrowing, int64, error) {
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count borrowings: %w", err)
	}

	borrowings := []entity.Borrowing{}
	err := r.withDetails(query).Order("id DESC").Offset(offset).Limit(limit).Find(&borrowings).Error
	if err != nil {
		return nil, 0, fmt.Errorf("list borrowings: %w", err)
	}
	return borrowings, total, nil
}

// withDetails ikut memuat peminjam, lembaga & barang-barang pengajuan.
// Unscoped: user/lembaga yang sudah dihapus (soft delete) tetap ditampilkan di riwayat peminjaman lama.
func (r *borrowingRepository) withDetails(query *gorm.DB) *gorm.DB {
	return query.
		Preload("User", func(db *gorm.DB) *gorm.DB { return db.Unscoped() }).
		Preload("Lembaga", func(db *gorm.DB) *gorm.DB { return db.Unscoped() }).
		Preload("Items", func(db *gorm.DB) *gorm.DB { return db.Order("id ASC") })
}
