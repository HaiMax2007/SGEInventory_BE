package dto

import (
	"github.com/golang-jwt/jwt/v5"

	"SGEInventory_BE/domain/errx"
)

// LoginRequest adalah isi body POST /api/auth/login.
// Password diberi `trim:"-"`: spasi tidak dibuang karena bisa saja bagian dari password.
type LoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required" trim:"-"`
}

// FieldErrors memetakan field yang gagal validasi ke error-nya (dipakai validator.ValidateStruct).
func (LoginRequest) FieldErrors() map[string]error {
	return map[string]error{
		"Email":    errx.ErrInvalidEmail,
		"Password": errx.ErrInvalidPassword,
	}
}

// LoginResponse adalah isi "data" saat login berhasil.
type LoginResponse struct {
	Token     string       `json:"token"`
	TokenType string       `json:"tokenType"`
	ExpiresIn int          `json:"expiresIn"` // dalam detik
	User      UserResponse `json:"user"`
}

// JWTClaims adalah data yang disimpan di dalam token.
// RegisteredClaims berisi field standar JWT seperti exp (kedaluwarsa) dan iat (dibuat).
type JWTClaims struct {
	UserID uint   `json:"userId"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}
