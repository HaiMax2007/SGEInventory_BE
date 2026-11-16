package auth

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"

	"SGEInventory_BE/domain/contracts"
	"SGEInventory_BE/domain/dto"
	"SGEInventory_BE/domain/errx"
)

// Token berlaku 24 jam; setelah itu user harus login ulang.
const tokenDuration = 24 * time.Hour

type authService struct {
	userRepo  contracts.UserRepository
	jwtSecret string
}

// NewAuthService membuat service login. Secret JWT dioper dari config di main.go.
func NewAuthService(userRepo contracts.UserRepository, jwtSecret string) contracts.AuthService {
	return &authService{userRepo: userRepo, jwtSecret: jwtSecret}
}

func (s *authService) Login(ctx context.Context, req dto.LoginRequest) (*dto.LoginResponse, error) {
	user, err := s.userRepo.FindByEmail(ctx, strings.ToLower(req.Email))
	if err != nil {
		if errors.Is(err, errx.ErrUserNotFound) {
			return nil, errx.ErrInvalidCredentials // jangan beri tahu bahwa email tidak terdaftar
		}
		return nil, err
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		return nil, errx.ErrInvalidCredentials
	}

	// Buat token berisi id & role user, lalu tanda tangani dengan JWT secret.
	now := time.Now()
	claims := dto.JWTClaims{
		UserID: user.ID,
		Role:   user.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   strconv.Itoa(int(user.ID)),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(tokenDuration)),
		},
	}
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(s.jwtSecret))
	if err != nil {
		return nil, fmt.Errorf("sign token: %w", err)
	}

	return &dto.LoginResponse{
		Token:     token,
		TokenType: "Bearer",
		ExpiresIn: int(tokenDuration.Seconds()),
		User:      dto.NewUserResponse(user),
	}, nil
}

func (s *authService) Me(ctx context.Context, userID uint) (*dto.UserResponse, error) {
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	res := dto.NewUserResponse(user)
	return &res, nil
}
