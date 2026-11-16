package user

import (
	"context"
	"fmt"

	"golang.org/x/crypto/bcrypt"

	"SGEInventory_BE/domain/contracts"
	"SGEInventory_BE/domain/dto"
	"SGEInventory_BE/domain/entity"
	"SGEInventory_BE/domain/errx"
	"SGEInventory_BE/helpers/pagination"
)

// bcrypt hanya bisa menerima password maksimal 72 byte.
const maxPasswordBytes = 72

type userService struct {
	repo contracts.UserRepository
}

// NewUserService membuat service user. Repository dioper dari main.go (dependency injection).
func NewUserService(repo contracts.UserRepository) contracts.UserService {
	return &userService{repo: repo}
}

func (s *userService) Register(ctx context.Context, req dto.RegisterUserRequest) (*dto.UserResponse, error) {
	if len(req.Password) > maxPasswordBytes {
		return nil, errx.ErrPasswordTooLong
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	// NewUser memastikan role selalu USER, tidak bisa dipilih sendiri.
	user := entity.NewUser(req.FullName, req.Email, string(hashedPassword), req.Contact)
	if err := s.repo.Create(ctx, user); err != nil {
		return nil, err // errx.ErrEmailTaken jika email sudah dipakai
	}

	res := dto.NewUserResponse(user)
	return &res, nil
}

func (s *userService) GetUsers(ctx context.Context, query dto.UserListQuery) ([]dto.UserResponse, *pagination.Pagination, error) {
	page := pagination.New(query.Page, query.Limit)

	users, total, err := s.repo.FindAll(ctx, query.Search, query.Role, page.Offset(), page.Limit)
	if err != nil {
		return nil, nil, err
	}
	page.SetTotal(total)

	res := make([]dto.UserResponse, 0, len(users))
	for i := range users {
		res = append(res, dto.NewUserResponse(&users[i]))
	}
	return res, page, nil
}

func (s *userService) GetUser(ctx context.Context, id uint) (*dto.UserResponse, error) {
	user, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	res := dto.NewUserResponse(user)
	return &res, nil
}

func (s *userService) UpdateUser(ctx context.Context, id uint, req dto.UpdateUserRequest) (*dto.UserResponse, error) {
	if req == (dto.UpdateUserRequest{}) {
		return nil, errx.ErrNothingToUpdate
	}

	user, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	user.ApplyUpdate(req.FullName, req.Email, req.Contact)
	if err := s.repo.Update(ctx, user); err != nil {
		return nil, err // errx.ErrEmailTaken jika email sudah dipakai user lain
	}

	res := dto.NewUserResponse(user)
	return &res, nil
}

func (s *userService) DeleteUser(ctx context.Context, id uint) error {
	return s.repo.Delete(ctx, id)
}
