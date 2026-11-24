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
	// rejecter menolak otomatis pengajuan PENDING yang tidak tercukupi lagi saat total barang dikurangi.
	// Diisi BorrowingService (hanya kemampuan ini yang dipakai, lihat contracts.PendingBorrowingRejecter).
	rejecter contracts.PendingBorrowingRejecter
	tx       contracts.Transactor
}

// NewItemService membuat service barang. Dependency dioper dari main.go (dependency injection).
func NewItemService(
	repo contracts.ItemRepository,
	lembagaRepo contracts.LembagaRepository,
	rejecter contracts.PendingBorrowingRejecter,
	tx contracts.Transactor,
) contracts.ItemService {
	return &itemService{repo: repo, lembagaRepo: lembagaRepo, rejecter: rejecter, tx: tx}
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

// UpdateItem mengubah barang di dalam transaksi dengan baris barang DIKUNCI (SELECT ... FOR UPDATE).
// Tanpa kunci, edit total yang berjalan bersamaan dengan ACC bisa menimpa stok dengan nilai lama
// (lost update). Jika stok tersedia berkurang, pengajuan PENDING yang tidak tercukupi lagi ditolak otomatis
// dengan pelaku adminID.
func (s *itemService) UpdateItem(ctx context.Context, id, adminID uint, req dto.UpdateItemRequest) (*dto.ItemResponse, error) {
	if req.Name == "" && req.Description == nil && req.LembagaID == nil && req.TotalQuantity == nil {
		return nil, errx.ErrNothingToUpdate
	}

	err := s.tx.WithinTransaction(ctx, func(ctx context.Context) error {
		item, err := s.repo.FindByIDForUpdate(ctx, id)
		if err != nil {
			return err // errx.ErrItemNotFound
		}

		// Lembaga asal yang baru dicek dulu (harus ada & belum dihapus).
		if req.LembagaID != nil {
			lembaga, err := s.lembagaRepo.FindByID(ctx, *req.LembagaID)
			if err != nil {
				return err // errx.ErrLembagaNotFound
			}
			item.LembagaID = lembaga.ID
		}

		if req.Name != "" {
			item.Name = req.Name
		}
		if req.Description != nil {
			item.Description = *req.Description // boleh "" untuk mengosongkan keterangan
		}
		availableBefore := item.AvailableQuantity
		if req.TotalQuantity != nil {
			// Aturan "total tidak boleh di bawah jumlah yang dipinjam" ada di entity.
			if err := item.ChangeTotal(*req.TotalQuantity); err != nil {
				return err
			}
		}

		if err := s.repo.Update(ctx, item); err != nil {
			return err
		}

		// Stok tersedia berkurang → pengajuan PENDING yang memuat barang ini mungkin tidak tercukupi lagi.
		if item.AvailableQuantity < availableBefore {
			if _, err := s.rejecter.RejectUnfulfillablePending(ctx, []uint{item.ID}, adminID); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	// Ambil ulang (dengan data lembaga) setelah transaksi selesai, untuk response.
	return s.GetItem(ctx, id)
}

// DeleteItem menghapus barang (soft delete). Baris barang dikunci dulu supaya tidak ada ACC yang
// mengurangi stoknya di antara pengecekan "masih dipinjam?" dan penghapusan.
func (s *itemService) DeleteItem(ctx context.Context, id uint) error {
	return s.tx.WithinTransaction(ctx, func(ctx context.Context) error {
		item, err := s.repo.FindByIDForUpdate(ctx, id)
		if err != nil {
			return err
		}

		// Barang yang sebagian masih dipinjam tidak boleh dihapus.
		if item.IsBorrowed() {
			return errx.ErrItemStillBorrowed
		}
		return s.repo.Delete(ctx, item)
	})
}
