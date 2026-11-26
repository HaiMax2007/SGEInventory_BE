package borrowing

import (
	"github.com/gofiber/fiber/v3"

	"SGEInventory_BE/internal/middlewares"
)

// RegisterRoutes mendaftarkan URL peminjaman. Semua butuh login; mengajukan & mengembalikan hanya role USER,
// melihat semua pengajuan serta ACC/menolak (peminjaman & pengembalian) hanya ADMIN.
// "/count" & "/me" didaftarkan sebelum "/:id", kalau tidak keduanya akan dianggap sebagai :id.
func RegisterRoutes(router fiber.Router, handler *BorrowingHandler, protected fiber.Handler) {
	borrowings := router.Group("/borrowings", protected)
	borrowings.Post("/", middlewares.UserOnly, handler.Create)
	borrowings.Get("/", middlewares.AdminOnly, handler.GetAll)
	borrowings.Get("/count", middlewares.AdminOnly, handler.Count)
	borrowings.Get("/me", handler.GetMine)
	borrowings.Get("/:id", handler.GetByID)

	// tahap 2: admin memproses pengajuan
	borrowings.Patch("/:id/approve", middlewares.AdminOnly, handler.Approve)
	borrowings.Patch("/:id/reject", middlewares.AdminOnly, handler.Reject)

	// tahap 3: user mengunggah foto pengembalian, admin memprosesnya
	borrowings.Post("/:id/return", middlewares.UserOnly, handler.SubmitReturn)
	borrowings.Patch("/:id/return/approve", middlewares.AdminOnly, handler.ApproveReturn)
	borrowings.Patch("/:id/return/reject", middlewares.AdminOnly, handler.RejectReturn)
}
