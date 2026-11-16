package auth

import (
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/limiter"

	"SGEInventory_BE/pkg/response"
)

// RegisterRoutes mendaftarkan URL auth. protected adalah middleware cek token (middlewares.Protected).
func RegisterRoutes(router fiber.Router, handler *AuthHandler, protected fiber.Handler) {
	// Login dibatasi 5x per menit per IP untuk menahan tebak-tebakan password.
	loginLimiter := limiter.New(limiter.Config{
		Max:        5,
		Expiration: time.Minute,
		LimitReached: func(c fiber.Ctx) error {
			return response.Message(c, fiber.StatusTooManyRequests, "Terlalu banyak percobaan login, coba lagi 1 menit lagi")
		},
	})

	auth := router.Group("/auth")
	auth.Post("/login", loginLimiter, handler.Login)
	auth.Post("/logout", protected, handler.Logout)
	auth.Get("/me", protected, handler.Me)
}
