package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"

	"SGEInventory_BE/domain/contracts"
	"SGEInventory_BE/domain/entity"
	"SGEInventory_BE/domain/errx"
)

type userRepository struct {
	db *gorm.DB
}

// NewUserRepository membuat repository user. Tipe kembaliannya interface contracts.UserRepository.
func NewUserRepository(db *gorm.DB) contracts.UserRepository {
	return &userRepository{db: db}
}

func (r *userRepository) Create(ctx context.Context, user *entity.User) error {
	if err := r.db.WithContext(ctx).Create(user).Error; err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return duplicateUserError(err)
		}
		return fmt.Errorf("create user: %w", err)
	}
	return nil
}

// duplicateUserError menentukan kolom unik mana yang bentrok (email atau NIM).
// Error asli Postgres (*pgconn.PgError) masih terbungkus di dalam err, dan field
// ConstraintName berisi nama constraint dari 001_schema.sql (uq_users_email / uq_users_nim).
func duplicateUserError(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.ConstraintName == "uq_users_nim" {
		return errx.ErrNIMTaken
	}
	return errx.ErrEmailTaken
}

func (r *userRepository) FindByID(ctx context.Context, id uint) (*entity.User, error) {
	var user entity.User
	if err := r.db.WithContext(ctx).First(&user, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errx.ErrUserNotFound
		}
		return nil, fmt.Errorf("find user %d: %w", id, err)
	}
	return &user, nil
}

func (r *userRepository) FindByEmail(ctx context.Context, email string) (*entity.User, error) {
	var user entity.User
	if err := r.db.WithContext(ctx).Where("email = ?", email).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errx.ErrUserNotFound
		}
		return nil, fmt.Errorf("find user by email: %w", err)
	}
	return &user, nil
}

// FindAll mengembalikan satu halaman user + jumlah seluruh user yang cocok dengan filter.
func (r *userRepository) FindAll(ctx context.Context, search, role string, offset, limit int) ([]entity.User, int64, error) {
	query := r.db.WithContext(ctx).Model(&entity.User{})
	if search != "" {
		// ILIKE = LIKE yang tidak membedakan huruf besar/kecil (di Postgres, LIKE membedakannya).
		query = query.Where("full_name ILIKE ? OR email ILIKE ?", "%"+search+"%", "%"+search+"%")
	}
	if role != "" {
		query = query.Where("role = ?", role)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count users: %w", err)
	}

	users := []entity.User{}
	if err := query.Order("id DESC").Offset(offset).Limit(limit).Find(&users).Error; err != nil {
		return nil, 0, fmt.Errorf("list users: %w", err)
	}
	return users, total, nil
}

func (r *userRepository) Update(ctx context.Context, user *entity.User) error {
	// Select menentukan kolom yang disimpan, termasuk jika nilainya kosong.
	err := r.db.WithContext(ctx).Model(user).Select("full_name", "email", "contact").Updates(user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return errx.ErrEmailTaken
		}
		return fmt.Errorf("update user %d: %w", user.ID, err)
	}
	return nil
}

// Delete melakukan soft delete (baris tetap ada, deleted_at terisi).
func (r *userRepository) Delete(ctx context.Context, id uint) error {
	result := r.db.WithContext(ctx).Delete(&entity.User{}, id)
	if result.Error != nil {
		return fmt.Errorf("delete user %d: %w", id, result.Error)
	}
	if result.RowsAffected == 0 {
		return errx.ErrUserNotFound
	}
	return nil
}
