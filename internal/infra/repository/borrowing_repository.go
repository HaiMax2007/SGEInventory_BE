package repository

import (
	"context"
	"errors"
	"fmt"

	"gorm.io/gorm"

	"SGEInventory_BE/domain/contracts"
	"SGEInventory_BE/domain/entity"
	"SGEInventory_BE/domain/enums"
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
	return conn(ctx, r.db).Transaction(func(tx *gorm.DB) error {
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
	err := r.withDetails(conn(ctx, r.db)).First(&borrowing, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errx.ErrBorrowingNotFound
		}
		return nil, fmt.Errorf("find borrowing %d: %w", id, err)
	}
	return &borrowing, nil
}

func (r *borrowingRepository) FindByUser(ctx context.Context, userID uint, offset, limit int) ([]entity.Borrowing, int64, error) {
	query := conn(ctx, r.db).Model(&entity.Borrowing{}).Where("user_id = ?", userID)
	return r.paginate(query, offset, limit)
}

func (r *borrowingRepository) FindAll(ctx context.Context, status string, offset, limit int) ([]entity.Borrowing, int64, error) {
	query := conn(ctx, r.db).Model(&entity.Borrowing{})
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

// FindByIDForUpdate: SELECT * FROM borrowings WHERE id = ? FOR UPDATE (tanpa relasi).
// Dua admin yang memproses pengajuan yang sama bergantian: yang kedua menunggu, lalu melihat status terbaru.
func (r *borrowingRepository) FindByIDForUpdate(ctx context.Context, id uint) (*entity.Borrowing, error) {
	var borrowing entity.Borrowing
	if err := conn(ctx, r.db).Clauses(forUpdate).First(&borrowing, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errx.ErrBorrowingNotFound
		}
		return nil, fmt.Errorf("lock borrowing %d: %w", id, err)
	}
	return &borrowing, nil
}

// UpdateStatus: UPDATE borrowings SET status = ?, updated_at = NOW() WHERE id = ?.
// Hanya kolom status yang disimpan (relasi tidak ikut tersimpan).
func (r *borrowingRepository) UpdateStatus(ctx context.Context, borrowing *entity.Borrowing) error {
	if err := conn(ctx, r.db).Model(borrowing).Update("status", borrowing.Status).Error; err != nil {
		return fmt.Errorf("update borrowing %d status: %w", borrowing.ID, err)
	}
	return nil
}

// UpdateReturn: UPDATE borrowings SET status = ?, return_photo = ?, updated_at = NOW() WHERE id = ?.
// Memakai map supaya return_photo nil benar-benar disimpan sebagai NULL (Updates dengan struct
// melewati field kosong).
func (r *borrowingRepository) UpdateReturn(ctx context.Context, borrowing *entity.Borrowing) error {
	err := conn(ctx, r.db).Model(borrowing).Updates(map[string]any{
		"status":       borrowing.Status,
		"return_photo": borrowing.ReturnPhoto,
	}).Error
	if err != nil {
		return fmt.Errorf("update borrowing %d return: %w", borrowing.ID, err)
	}
	return nil
}

// LockUnfulfillablePending menjalankan:
//
//	SELECT id FROM borrowings
//	WHERE status = 'PENDING' AND id IN (
//	    SELECT bi.borrowing_id FROM borrowing_items bi JOIN items i ON i.id = bi.item_id
//	    WHERE bi.item_id IN (...) AND bi.quantity > i.available_quantity)
//	ORDER BY id FOR UPDATE
//
// Hanya barang di itemIDs (yang stoknya baru berkurang) yang diperiksa, memakai index
// idx_borrowing_items_item_id. Subquery dipakai karena Postgres tidak mengizinkan DISTINCT/JOIN-duplikat
// bersama FOR UPDATE. Pengajuan dikunci urut id agar tidak deadlock dengan transaksi lain.
func (r *borrowingRepository) LockUnfulfillablePending(ctx context.Context, itemIDs []uint) ([]uint, error) {
	db := conn(ctx, r.db)
	unfulfillable := db.Table("borrowing_items AS bi").
		Select("bi.borrowing_id").
		Joins("JOIN items i ON i.id = bi.item_id").
		Where("bi.item_id IN ? AND bi.quantity > i.available_quantity", itemIDs)

	ids := []uint{}
	err := db.Model(&entity.Borrowing{}).Clauses(forUpdate).
		Where("status = ? AND id IN (?)", enums.BorrowingPending, unfulfillable).
		Order("id ASC").
		Pluck("id", &ids).Error
	if err != nil {
		return nil, fmt.Errorf("lock unfulfillable borrowings: %w", err)
	}
	return ids, nil
}

// UpdateStatuses: UPDATE borrowings SET status = ?, updated_at = NOW() WHERE id IN (...).
func (r *borrowingRepository) UpdateStatuses(ctx context.Context, ids []uint, status string) error {
	if len(ids) == 0 {
		return nil
	}
	err := conn(ctx, r.db).Model(&entity.Borrowing{}).Where("id IN ?", ids).Update("status", status).Error
	if err != nil {
		return fmt.Errorf("update borrowings status: %w", err)
	}
	return nil
}
