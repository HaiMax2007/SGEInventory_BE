package controllers

import (
	"errors"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"SGEInventory_BE/config"
	"SGEInventory_BE/models"
)

// Token berlaku 24 jam; setelah itu user harus login ulang.
const tokenDuration = 24 * time.Hour

// Login: POST /api/auth/login
func Login(c fiber.Ctx) error {
	var req models.LoginRequest
	if err := c.Bind().Body(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Data tidak valid", "error": err.Error()})
	}

	// Pesan error sengaja sama untuk "email tidak ada" dan "password salah",
	// supaya orang lain tidak bisa menebak email mana yang terdaftar.
	var user models.User
	if err := config.DB.Where("email = ?", strings.ToLower(req.Email)).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"message": "Email atau password salah"})
		}
		return serverError(c, err)
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"message": "Email atau password salah"})
	}

	// Buat token berisi id & role user, lalu tanda tangani dengan JWT_SECRET.
	now := time.Now()
	claims := models.JWTClaims{
		UserID: user.ID,
		Role:   user.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   strconv.Itoa(int(user.ID)),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(tokenDuration)),
		},
	}
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(os.Getenv("JWT_SECRET")))
	if err != nil {
		return serverError(c, err)
	}

	return c.JSON(fiber.Map{
		"message": "Login berhasil",
		"data": fiber.Map{
			"token":     token,
			"tokenType": "Bearer",
			"expiresIn": int(tokenDuration.Seconds()),
			"user":      user,
		},
	})
}

// Me: GET /api/auth/me (data user yang sedang login, butuh token)
func Me(c fiber.Ctx) error {
	userID := fiber.Locals[uint](c, "userID") // diisi oleh middlewares.Protected

	var user models.User
	if err := config.DB.First(&user, userID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"message": "User tidak ditemukan"})
		}
		return serverError(c, err)
	}

	return c.JSON(fiber.Map{"message": "User ditemukan", "data": user})
}
