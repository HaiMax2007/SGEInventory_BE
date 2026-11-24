package borrowing

import (
	"github.com/gofiber/fiber/v3"

	"SGEInventory_BE/internal/middlewares"
)

// RegisterRoutes mendaftarkan URL peminjaman. Semua butuh login; mengajukan hanya role USER,
// melihat semua pengajuan serta ACC/menolak hanya ADMIN.
// "/me" didaftarkan sebelum "/:id", kalau tidak "me" akan dianggap sebagai :id.
func RegisterRoutes(router fiber.Router, handler *BorrowingHandler, protected fiber.Handler) {
	borrowings := router.Group("/borrowings", protected)
	borrowings.Post("/", middlewares.UserOnly, handler.Create)
	borrowings.Get("/", middlewares.AdminOnly, handler.GetAll)
	borrowings.Get("/me", handler.GetMine)
	borrowings.Get("/:id", handler.GetByID)
	borrowings.Patch("/:id/approve", middlewares.AdminOnly, handler.Approve)
	borrowings.Patch("/:id/reject", middlewares.AdminOnly, handler.Reject)
}
