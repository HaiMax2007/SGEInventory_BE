package history

import (
	"github.com/gofiber/fiber/v3"

	"SGEInventory_BE/internal/middlewares"
)

// RegisterRoutes mendaftarkan URL riwayat pengajuan. Khusus role USER; user diambil dari token.
func RegisterRoutes(router fiber.Router, handler *HistoryHandler, protected fiber.Handler) {
	history := router.Group("/history", protected, middlewares.UserOnly)
	history.Get("/me", handler.GetMine)
}
