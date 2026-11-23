package userlog

import (
	"context"
	"slices"

	"SGEInventory_BE/domain/contracts"
	"SGEInventory_BE/domain/dto"
	"SGEInventory_BE/domain/enums"
	"SGEInventory_BE/domain/errx"
	"SGEInventory_BE/helpers/pagination"
)

type userLogService struct {
	repo contracts.UserLogRepository
}

// NewUserLogService membuat service riwayat aktivitas. Repository dioper dari main.go (dependency injection).
func NewUserLogService(repo contracts.UserLogRepository) contracts.UserLogService {
	return &userLogService{repo: repo}
}

// GetLogs mengambil semua log untuk admin. Filter requestType kosong = semua jenis pengajuan.
func (s *userLogService) GetLogs(ctx context.Context, query dto.UserLogListQuery) ([]dto.UserLogListItemResponse, *pagination.Pagination, error) {
	if query.RequestType != "" && !slices.Contains(enums.RequestTypes, query.RequestType) {
		return nil, nil, errx.ErrInvalidRequestType
	}

	page := pagination.New(query.Page, query.Limit)
	logs, total, err := s.repo.FindAll(ctx, query.RequestType, page.Offset(), page.Limit)
	if err != nil {
		return nil, nil, err
	}
	page.SetTotal(total)

	res := make([]dto.UserLogListItemResponse, 0, len(logs))
	for i := range logs {
		res = append(res, dto.NewUserLogListItemResponse(&logs[i]))
	}
	return res, page, nil
}
