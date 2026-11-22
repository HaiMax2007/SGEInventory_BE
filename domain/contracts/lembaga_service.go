package contracts

import (
	"context"

	"SGEInventory_BE/domain/dto"
	"SGEInventory_BE/helpers/pagination"
)

// LembagaService berisi aturan bisnis lembaga. Implementasinya ada di internal/app/lembaga/service.go.
type LembagaService interface {
	CreateLembaga(ctx context.Context, req dto.CreateLembagaRequest) (*dto.LembagaResponse, error)
	GetLembagas(ctx context.Context, query dto.LembagaListQuery) ([]dto.LembagaResponse, *pagination.Pagination, error)
	GetLembaga(ctx context.Context, id uint) (*dto.LembagaResponse, error)
	UpdateLembaga(ctx context.Context, id uint, req dto.UpdateLembagaRequest) (*dto.LembagaResponse, error)
	DeleteLembaga(ctx context.Context, id uint) error
}
