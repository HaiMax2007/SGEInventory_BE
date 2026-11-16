package auth

import (
	"github.com/gofiber/fiber/v3"

	"SGEInventory_BE/domain/contracts"
	"SGEInventory_BE/domain/dto"
	"SGEInventory_BE/domain/errx"
	"SGEInventory_BE/pkg/response"
	"SGEInventory_BE/pkg/validator"
)

// AuthHandler menerima request login/logout/me.
type AuthHandler struct {
	svc contracts.AuthService
}

func NewAuthHandler(svc contracts.AuthService) *AuthHandler {
	return &AuthHandler{svc: svc}
}

// Login: POST /api/auth/login
func (h *AuthHandler) Login(c fiber.Ctx) error {
	var req dto.LoginRequest
	if err := c.Bind().Body(&req); err != nil {
		return errx.Validation(err)
	}
	if err := validator.ValidateStruct(req); err != nil {
		return errx.Validation(err)
	}

	res, err := h.svc.Login(c.Context(), req)
	if err != nil {
		return err
	}
	return response.Success(c, fiber.StatusOK, "Login berhasil", res)
}

// Logout: POST /api/auth/logout (butuh token)
// JWT tidak disimpan di server, jadi tidak ada yang perlu dihapus di sini.
// FE yang menghapus token dari penyimpanannya (cookie) setelah endpoint ini berhasil.
// Token yang sempat tersalin masih berlaku sampai kedaluwarsa (24 jam).
func (h *AuthHandler) Logout(c fiber.Ctx) error {
	return response.Message(c, fiber.StatusOK, "Logout berhasil")
}

// Me: GET /api/auth/me (data user yang sedang login, butuh token)
func (h *AuthHandler) Me(c fiber.Ctx) error {
	userID := fiber.Locals[uint](c, "userID") // diisi oleh middlewares.Protected

	res, err := h.svc.Me(c.Context(), userID)
	if err != nil {
		return err
	}
	return response.Success(c, fiber.StatusOK, "User ditemukan", res)
}
