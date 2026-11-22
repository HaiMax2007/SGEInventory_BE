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
	repo        contracts.ItemRepository
	lembagaRepo contracts.LembagaRepository // untuk memastikan lembaga asal barang ada
}

// NewItemService membuat service barang. Repository dioper dari main.go (dependency injection).
func NewItemService(repo contracts.ItemRepository, lembagaRepo contracts.LembagaRepository) contracts.ItemService {
	return &itemService{repo: repo, lembagaRepo: lembagaRepo}
}

func (s *itemService) CreateItem(ctx context.Context, req dto.CreateItemRequest) (*dto.ItemResponse, error) {
	// Lembaga harus ada dan belum dihapus. Foreign key di database tidak cukup, karena lembaga
	// yang di-soft delete barisnya masih ada (hanya deleted_at yang terisi).
	lembaga, err := s.lembagaRepo.FindByID(ctx, req.LembagaID)
	if err != nil {
		return nil, err // errx.ErrLembagaNotFound
	}

	item := entity.NewItem(req.Name, req.Description, req.LembagaID, req.TotalQuantity)
	if err := s.repo.Create(ctx, item); err != nil {
		return nil, err
	}

	// Diisi SETELAH Create: kalau diisi sebelumnya, GORM ikut mencoba menyimpan data lembaga.
	item.Lembaga = lembaga
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
	if req.Name == "" && req.Description == nil && req.LembagaID == nil && req.TotalQuantity == nil {
		return nil, errx.ErrNothingToUpdate
	}

	item, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Lembaga asal yang baru dicek dulu (harus ada & belum dihapus).
	var newLembaga *entity.Lembaga
	if req.LembagaID != nil {
		newLembaga, err = s.lembagaRepo.FindByID(ctx, *req.LembagaID)
		if err != nil {
			return nil, err // errx.ErrLembagaNotFound
		}
		item.LembagaID = newLembaga.ID
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

	// item.Lembaga dari FindByID masih lembaga lama; ganti supaya response menampilkan yang baru.
	if newLembaga != nil {
		item.Lembaga = newLembaga
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
