package contracts

import (
	"context"

	"SGEInventory_BE/domain/dto"
	"SGEInventory_BE/helpers/pagination"
)

// ConsignmentService berisi aturan bisnis penitipan. Implementasinya ada di internal/app/consignment/.
type ConsignmentService interface {
	// CreateConsignment: userID = penitip (pemilik token).
	CreateConsignment(ctx context.Context, userID uint, req dto.CreateConsignmentRequest) (*dto.ConsignmentResponse, error)
	// GetMyConsignments: penitipan milik userID saja.
	GetMyConsignments(ctx context.Context, userID uint, query dto.ConsignmentListQuery) ([]dto.ConsignmentResponse, *pagination.Pagination, error)
	// GetConsignments: semua penitipan (admin); query.Status tidak dikenal → 400.
	GetConsignments(ctx context.Context, query dto.ConsignmentListQuery) ([]dto.ConsignmentResponse, *pagination.Pagination, error)
	// GetConsignment: detail satu penitipan; bukan pemilik & bukan admin → 404.
	GetConsignment(ctx context.Context, id, userID uint, isAdmin bool) (*dto.ConsignmentResponse, error)

	// ACC / tolak (admin). adminID dicatat sebagai pelaku di log_users.
	ApproveConsignment(ctx context.Context, id, adminID uint) (*dto.ConsignmentResponse, error)
	RejectConsignment(ctx context.Context, id, adminID uint) (*dto.ConsignmentResponse, error)
}
