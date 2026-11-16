package contracts

import (
	"context"

	"SGEInventory_BE/domain/entity"
)

// UserRepository adalah "janji" cara mengakses data user di database.
// Implementasinya ada di internal/infra/repository/user_repository.go (GORM).
// Service hanya mengenal interface ini, jadi mudah diganti versi palsu (mock) saat testing.
type UserRepository interface {
	Create(ctx context.Context, user *entity.User) error
	FindByID(ctx context.Context, id uint) (*entity.User, error)
	FindByEmail(ctx context.Context, email string) (*entity.User, error)
	FindAll(ctx context.Context, search, role string, offset, limit int) ([]entity.User, int64, error)
	Update(ctx context.Context, user *entity.User) error
	Delete(ctx context.Context, id uint) error
}
