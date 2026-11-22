package contracts

import (
	"context"

	"SGEInventory_BE/domain/entity"
)

// LembagaRepository adalah "janji" cara mengakses data lembaga di database.
// Implementasinya ada di internal/infra/repository/lembaga_repository.go (GORM).
type LembagaRepository interface {
	Create(ctx context.Context, lembaga *entity.Lembaga) error
	FindByID(ctx context.Context, id uint) (*entity.Lembaga, error)
	FindAll(ctx context.Context, search string, offset, limit int) ([]entity.Lembaga, int64, error)
	Update(ctx context.Context, lembaga *entity.Lembaga) error
	Delete(ctx context.Context, lembaga *entity.Lembaga) error
	// CountItems menghitung barang (yang belum dihapus) yang berasal dari lembaga ini.
	CountItems(ctx context.Context, id uint) (int64, error)
}
