package contracts

import (
	"context"

	"SGEInventory_BE/domain/dto"
	"SGEInventory_BE/helpers/pagination"
)

// UserLogService berisi aturan membaca riwayat aktivitas. Implementasinya ada di internal/app/userlog/service.go.
type UserLogService interface {
	// GetLogs mengambil daftar SEMUA log (khusus admin), bisa difilter jenis pengajuan.
	GetLogs(ctx context.Context, query dto.UserLogListQuery) ([]dto.UserLogListItemResponse, *pagination.Pagination, error)
}
