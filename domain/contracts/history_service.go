package contracts

import (
	"context"

	"SGEInventory_BE/domain/dto"
	"SGEInventory_BE/helpers/pagination"
)

// HistoryService menyusun riwayat pengajuan seorang user: gabungan peminjaman & penitipan, terbaru dulu.
// Implementasinya ada di internal/app/history/service.go.
type HistoryService interface {
	GetMyHistory(ctx context.Context, userID uint, query dto.HistoryListQuery) ([]dto.HistoryResponse, *pagination.Pagination, error)
}
