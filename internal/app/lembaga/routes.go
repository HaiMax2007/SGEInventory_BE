package lembaga

import (
	"github.com/gofiber/fiber/v3"

	"SGEInventory_BE/internal/middlewares"
)

// RegisterRoutes mendaftarkan URL lembaga. Semua butuh login (FE memakai GET untuk pilihan lembaga);
// tambah/ubah/hapus hanya admin.
func RegisterRoutes(router fiber.Router, handler *LembagaHandler, protected fiber.Handler) {
	lembaga := router.Group("/lembaga", protected)
	lembaga.Get("/", handler.GetAll)
	lembaga.Get("/:id", handler.GetByID)
	lembaga.Post("/", middlewares.AdminOnly, handler.Create)
	lembaga.Put("/:id", middlewares.AdminOnly, handler.Update)
	lembaga.Delete("/:id", middlewares.AdminOnly, handler.Delete)
}
