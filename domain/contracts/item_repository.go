package contracts

import (
	"context"

	"SGEInventory_BE/domain/entity"
)

// ItemRepository adalah "janji" cara mengakses data barang di database.
// Implementasinya ada di internal/infra/repository/item_repository.go (GORM).
type ItemRepository interface {
	Create(ctx context.Context, item *entity.Item) error
	FindByID(ctx context.Context, id uint) (*entity.Item, error)
	FindAll(ctx context.Context, search string, onlyAvailable bool, offset, limit int) ([]entity.Item, int64, error)
	Update(ctx context.Context, item *entity.Item) error
	Delete(ctx context.Context, item *entity.Item) error
}
