package lembaga

import (
	"context"

	"SGEInventory_BE/domain/contracts"
	"SGEInventory_BE/domain/dto"
	"SGEInventory_BE/domain/entity"
	"SGEInventory_BE/domain/errx"
	"SGEInventory_BE/helpers/pagination"
)

type lembagaService struct {
	repo contracts.LembagaRepository
}

// NewLembagaService membuat service lembaga. Repository dioper dari main.go (dependency injection).
func NewLembagaService(repo contracts.LembagaRepository) contracts.LembagaService {
	return &lembagaService{repo: repo}
}

func (s *lembagaService) CreateLembaga(ctx context.Context, req dto.CreateLembagaRequest) (*dto.LembagaResponse, error) {
	lembaga := entity.NewLembaga(req.Name)
	if err := s.repo.Create(ctx, lembaga); err != nil {
		return nil, err // errx.ErrLembagaNameTaken jika nama sudah dipakai
	}

	res := dto.NewLembagaResponse(lembaga)
	return &res, nil
}

func (s *lembagaService) GetLembagas(ctx context.Context, query dto.LembagaListQuery) ([]dto.LembagaResponse, *pagination.Pagination, error) {
	page := pagination.New(query.Page, query.Limit)

	lembagas, total, err := s.repo.FindAll(ctx, query.Search, page.Offset(), page.Limit)
	if err != nil {
		return nil, nil, err
	}
	page.SetTotal(total)

	res := make([]dto.LembagaResponse, 0, len(lembagas))
	for i := range lembagas {
		res = append(res, dto.NewLembagaResponse(&lembagas[i]))
	}
	return res, page, nil
}

func (s *lembagaService) GetLembaga(ctx context.Context, id uint) (*dto.LembagaResponse, error) {
	lembaga, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	res := dto.NewLembagaResponse(lembaga)
	return &res, nil
}

func (s *lembagaService) UpdateLembaga(ctx context.Context, id uint, req dto.UpdateLembagaRequest) (*dto.LembagaResponse, error) {
	lembaga, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	lembaga.Name = req.Name
	if err := s.repo.Update(ctx, lembaga); err != nil {
		return nil, err // errx.ErrLembagaNameTaken jika nama sudah dipakai lembaga lain
	}

	res := dto.NewLembagaResponse(lembaga)
	return &res, nil
}

func (s *lembagaService) DeleteLembaga(ctx context.Context, id uint) error {
	lembaga, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return err
	}

	// Lembaga yang masih menjadi asal barang tidak boleh dihapus. Pengecekan ini perlu di aplikasi
	// karena hapus di sini adalah soft delete (UPDATE deleted_at), sehingga FOREIGN KEY RESTRICT
	// di database tidak ikut menghalangi.
	total, err := s.repo.CountItems(ctx, lembaga.ID)
	if err != nil {
		return err
	}
	if total > 0 {
		return errx.ErrLembagaInUse
	}
	return s.repo.Delete(ctx, lembaga)
}
