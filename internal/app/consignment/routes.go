package consignment

import (
	"github.com/gofiber/fiber/v3"

	"SGEInventory_BE/internal/middlewares"
)

// RegisterRoutes mendaftarkan URL penitipan. Semua butuh login; mengajukan penitipan hanya role USER,
// melihat semua penitipan serta ACC/menolak hanya ADMIN.
// "/count" & "/me" didaftarkan sebelum "/:id", kalau tidak keduanya akan dianggap sebagai :id.
func RegisterRoutes(router fiber.Router, handler *ConsignmentHandler, protected fiber.Handler) {
	consignments := router.Group("/consignments", protected)
	consignments.Post("/", middlewares.UserOnly, handler.Create)
	consignments.Get("/", middlewares.AdminOnly, handler.GetAll)
	consignments.Get("/count", middlewares.AdminOnly, handler.Count)
	consignments.Get("/me", middlewares.UserOnly, handler.GetMine)
	consignments.Get("/:id", middlewares.AdminOnly, handler.GetByID)

	// tahap 2: admin memproses penitipan
	consignments.Patch("/:id/approve", middlewares.AdminOnly, handler.Approve)
	consignments.Patch("/:id/reject", middlewares.AdminOnly, handler.Reject)
}
