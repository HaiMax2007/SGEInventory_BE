package contracts

import (
	"context"

	"SGEInventory_BE/domain/dto"
)

// ConsignmentService berisi aturan bisnis penitipan. Implementasinya ada di internal/app/consignment/service.go.
type ConsignmentService interface {
	// CreateConsignment: userID = penitip (pemilik token).
	CreateConsignment(ctx context.Context, userID uint, req dto.CreateConsignmentRequest) (*dto.ConsignmentResponse, error)
}
