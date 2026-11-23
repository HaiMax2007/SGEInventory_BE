package userlog

import (
	"github.com/gofiber/fiber/v3"

	"SGEInventory_BE/internal/middlewares"
)

// RegisterRoutes mendaftarkan URL riwayat aktivitas (log_users). Khusus admin.
// Riwayat milik satu pengajuan untuk peminjam ada di GET /api/borrowings/:id (timeline).
func RegisterRoutes(router fiber.Router, handler *UserLogHandler, protected fiber.Handler) {
	logs := router.Group("/logs", protected, middlewares.AdminOnly)
	logs.Get("/", handler.GetAll)
}
