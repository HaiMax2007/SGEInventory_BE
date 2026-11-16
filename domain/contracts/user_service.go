package contracts

import (
	"context"

	"SGEInventory_BE/domain/dto"
	"SGEInventory_BE/helpers/pagination"
)

// UserService berisi aturan bisnis user. Implementasinya ada di internal/app/user/service.go.
type UserService interface {
	Register(ctx context.Context, req dto.RegisterUserRequest) (*dto.UserResponse, error)
	GetUsers(ctx context.Context, query dto.UserListQuery) ([]dto.UserResponse, *pagination.Pagination, error)
	GetUser(ctx context.Context, id uint) (*dto.UserResponse, error)
	UpdateUser(ctx context.Context, id uint, req dto.UpdateUserRequest) (*dto.UserResponse, error)
	DeleteUser(ctx context.Context, id uint) error
}
