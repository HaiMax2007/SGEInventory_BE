package contracts

import (
	"context"

	"SGEInventory_BE/domain/dto"
	"SGEInventory_BE/helpers/pagination"
)

// ItemService berisi aturan bisnis barang. Implementasinya ada di internal/app/item/service.go.
type ItemService interface {
	CreateItem(ctx context.Context, req dto.CreateItemRequest) (*dto.ItemResponse, error)
	GetItems(ctx context.Context, query dto.ItemListQuery) ([]dto.ItemResponse, *pagination.Pagination, error)
	GetItem(ctx context.Context, id uint) (*dto.ItemResponse, error)
	UpdateItem(ctx context.Context, id uint, req dto.UpdateItemRequest) (*dto.ItemResponse, error)
	DeleteItem(ctx context.Context, id uint) error
}
