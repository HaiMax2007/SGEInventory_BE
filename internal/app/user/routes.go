package user

import (
	"github.com/gofiber/fiber/v3"

	"SGEInventory_BE/internal/middlewares"
)

// RegisterRoutes mendaftarkan URL user. protected adalah middleware cek token (middlewares.Protected).
func RegisterRoutes(router fiber.Router, handler *UserHandler, protected fiber.Handler) {
	users := router.Group("/users")
	users.Post("/", handler.Register) // terbuka, tanpa token
	users.Get("/", protected, middlewares.AdminOnly, handler.GetAll)
	users.Get("/:id", protected, handler.GetByID)
	users.Put("/:id", protected, handler.Update)
	users.Delete("/:id", protected, middlewares.AdminOnly, handler.Delete)
}
