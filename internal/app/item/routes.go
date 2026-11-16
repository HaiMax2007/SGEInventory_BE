package item

import (
	"github.com/gofiber/fiber/v3"

	"SGEInventory_BE/internal/middlewares"
)

// RegisterRoutes mendaftarkan URL barang. Semua butuh login; tambah/ubah/hapus hanya admin.
func RegisterRoutes(router fiber.Router, handler *ItemHandler, protected fiber.Handler) {
	items := router.Group("/items", protected)
	items.Get("/", handler.GetAll)
	items.Get("/:id", handler.GetByID)
	items.Post("/", middlewares.AdminOnly, handler.Create)
	items.Put("/:id", middlewares.AdminOnly, handler.Update)
	items.Delete("/:id", middlewares.AdminOnly, handler.Delete)
}
