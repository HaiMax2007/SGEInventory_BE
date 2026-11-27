package userlog

import (
	"github.com/gofiber/fiber/v3"

	"SGEInventory_BE/internal/middlewares"
)

// RegisterRoutes mendaftarkan URL riwayat aktivitas (log_users):
//   - GET /api/logs        : semua log (khusus admin)
//   - GET /api/history/me  : aktivitas yang dilakukan user itu sendiri (khusus role USER, dari token)
//
// Riwayat lengkap satu pengajuan (termasuk aksi admin) ada di GET /api/borrowings/:id (timeline).
func RegisterRoutes(router fiber.Router, handler *UserLogHandler, protected fiber.Handler) {
	logs := router.Group("/logs", protected, middlewares.AdminOnly)
	logs.Get("/", handler.GetAll)

	history := router.Group("/history", protected, middlewares.UserOnly)
	history.Get("/me", handler.GetMine)
}
