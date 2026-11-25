package consignment

import (
	"github.com/gofiber/fiber/v3"

	"SGEInventory_BE/internal/middlewares"
)

// RegisterRoutes mendaftarkan URL penitipan. Semua butuh login; mengajukan penitipan hanya role USER.
// (ACC/tolak oleh admin menyusul di tahap berikutnya.)
func RegisterRoutes(router fiber.Router, handler *ConsignmentHandler, protected fiber.Handler) {
	consignments := router.Group("/consignments", protected)
	consignments.Post("/", middlewares.UserOnly, handler.Create)
}
