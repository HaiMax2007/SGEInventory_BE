package contracts

import (
	"context"

	"SGEInventory_BE/domain/dto"
)

// AuthService berisi aturan login. Implementasinya ada di internal/app/auth/service.go.
type AuthService interface {
	Login(ctx context.Context, req dto.LoginRequest) (*dto.LoginResponse, error)
	Me(ctx context.Context, userID uint) (*dto.UserResponse, error)
}
