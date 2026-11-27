package userlog

import (
	"context"
	"slices"

	"SGEInventory_BE/domain/contracts"
	"SGEInventory_BE/domain/dto"
	"SGEInventory_BE/domain/entity"
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

// GetLogs mengambil semua log untuk admin. Filter requestType kosong = semua jenis pengajuan;
// search mencocokkan nama lengkap pelaku.
func (s *userLogService) GetLogs(ctx context.Context, query dto.UserLogListQuery) ([]dto.UserLogListItemResponse, *pagination.Pagination, error) {
	if err := checkRequestType(query.RequestType); err != nil {
		return nil, nil, err
	}

	page := pagination.New(query.Page, query.Limit)
	logs, total, err := s.repo.FindAll(ctx, query.RequestType, query.Search, page.Offset(), page.Limit)
	if err != nil {
		return nil, nil, err
	}
	page.SetTotal(total)
	return toResponses(logs), page, nil
}

// GetMyLogs mengambil aktivitas yang DILAKUKAN userID sendiri (pelaku = user itu), terbaru dulu.
// Aksi admin pada pengajuannya (ACC/tolak) tidak ikut, karena pelakunya admin.
func (s *userLogService) GetMyLogs(ctx context.Context, userID uint, query dto.UserLogListQuery) ([]dto.UserLogListItemResponse, *pagination.Pagination, error) {
	if err := checkRequestType(query.RequestType); err != nil {
		return nil, nil, err
	}

	page := pagination.New(query.Page, query.Limit)
	logs, total, err := s.repo.FindByUser(ctx, userID, query.RequestType, page.Offset(), page.Limit)
	if err != nil {
		return nil, nil, err
	}
	page.SetTotal(total)
	return toResponses(logs), page, nil
}

// checkRequestType: kosong = semua jenis; selain BORROWING/CONSIGNMENT → 400, supaya salah ketik di FE
// (mis. "borrowing" huruf kecil) langsung ketahuan.
func checkRequestType(requestType string) error {
	if requestType != "" && !slices.Contains(enums.RequestTypes, requestType) {
		return errx.ErrInvalidRequestType
	}
	return nil
}

func toResponses(logs []entity.UserLog) []dto.UserLogListItemResponse {
	res := make([]dto.UserLogListItemResponse, 0, len(logs))
	for i := range logs {
		res = append(res, dto.NewUserLogListItemResponse(&logs[i]))
	}
	return res
}
