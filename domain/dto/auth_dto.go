package dto

import "github.com/golang-jwt/jwt/v5"

// LoginRequest adalah isi body POST /api/auth/login.
type LoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
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
