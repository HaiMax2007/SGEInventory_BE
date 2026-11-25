package contracts

import (
	"context"

	"SGEInventory_BE/domain/dto"
	"SGEInventory_BE/helpers/pagination"
)

// ItemService berisi aturan bisnis barang. Implementasinya ada di internal/app/item/service.go.
type ItemService interface {
	CreateItem(ctx context.Context, req dto.CreateItemRequest) (*dto.ItemResponse, error)
	// GetItems & GetItem: isAdmin=false → barang titipan (CONSIGNMENT) tidak terlihat.
	GetItems(ctx context.Context, query dto.ItemListQuery, isAdmin bool) ([]dto.ItemResponse, *pagination.Pagination, error)
	GetItem(ctx context.Context, id uint, isAdmin bool) (*dto.ItemResponse, error)
	// UpdateItem: adminID dicatat sebagai pelaku jika ada pengajuan yang ditolak otomatis karena total dikurangi.
	UpdateItem(ctx context.Context, id, adminID uint, req dto.UpdateItemRequest) (*dto.ItemResponse, error)
	DeleteItem(ctx context.Context, id uint) error
}
