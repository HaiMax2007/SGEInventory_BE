package item

import (
	"context"

	"SGEInventory_BE/domain/contracts"
	"SGEInventory_BE/domain/dto"
	"SGEInventory_BE/domain/entity"
	"SGEInventory_BE/domain/errx"
	"SGEInventory_BE/helpers/pagination"
)

type itemService struct {
	repo contracts.ItemRepository
}

// NewItemService membuat service barang. Repository dioper dari main.go (dependency injection).
func NewItemService(repo contracts.ItemRepository) contracts.ItemService {
	return &itemService{repo: repo}
}

func (s *itemService) CreateItem(ctx context.Context, req dto.CreateItemRequest) (*dto.ItemResponse, error) {
	item := entity.NewItem(req.Name, req.Description, req.TotalQuantity)
	if err := s.repo.Create(ctx, item); err != nil {
		return nil, err
	}

	res := dto.NewItemResponse(item)
	return &res, nil
}

func (s *itemService) GetItems(ctx context.Context, query dto.ItemListQuery) ([]dto.ItemResponse, *pagination.Pagination, error) {
	page := pagination.New(query.Page, query.Limit)

	items, total, err := s.repo.FindAll(ctx, query.Search, query.OnlyAvailable, page.Offset(), page.Limit)
	if err != nil {
		return nil, nil, err
	}
	page.SetTotal(total)

	res := make([]dto.ItemResponse, 0, len(items))
	for i := range items {
		res = append(res, dto.NewItemResponse(&items[i]))
	}
	return res, page, nil
}

func (s *itemService) GetItem(ctx context.Context, id uint) (*dto.ItemResponse, error) {
	item, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	res := dto.NewItemResponse(item)
	return &res, nil
}

func (s *itemService) UpdateItem(ctx context.Context, id uint, req dto.UpdateItemRequest) (*dto.ItemResponse, error) {
	if req.Name == "" && req.Description == nil && req.TotalQuantity == nil {
		return nil, errx.ErrNothingToUpdate
	}

	item, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if req.Name != "" {
		item.Name = req.Name
	}
	if req.Description != nil {
		item.Description = *req.Description // boleh "" untuk mengosongkan keterangan
	}
	if req.TotalQuantity != nil {
		// Aturan "total tidak boleh di bawah jumlah yang dipinjam" ada di entity.
		if err := item.ChangeTotal(*req.TotalQuantity); err != nil {
			return nil, err
		}
	}

	if err := s.repo.Update(ctx, item); err != nil {
		return nil, err
	}

	res := dto.NewItemResponse(item)
	return &res, nil
}

func (s *itemService) DeleteItem(ctx context.Context, id uint) error {
	item, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return err
	}

	// Barang yang sebagian masih dipinjam tidak boleh dihapus.
	if item.IsBorrowed() {
		return errx.ErrItemStillBorrowed
	}
	return s.repo.Delete(ctx, item)
}
