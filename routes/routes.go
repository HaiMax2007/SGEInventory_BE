package routes

import (
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/limiter"

	"SGEInventory_BE/controllers"
	"SGEInventory_BE/middlewares"
)

func SetupRoutes(app *fiber.App) {
	api := app.Group("/api")

	// Auth. Login dibatasi 5x per menit per IP untuk menahan tebak-tebakan password.
	loginLimiter := limiter.New(limiter.Config{
		Max:        5,
		Expiration: time.Minute,
		LimitReached: func(c fiber.Ctx) error {
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{"message": "Terlalu banyak percobaan login, coba lagi 1 menit lagi"})
		},
	})

	auth := api.Group("/auth")
	auth.Post("/login", loginLimiter, controllers.Login)
	auth.Post("/logout", middlewares.Protected, controllers.Logout)
	auth.Get("/me", middlewares.Protected, controllers.Me)

	// User. Register terbuka; sisanya butuh token.
	users := api.Group("/users")
	users.Post("/", controllers.Register)
	users.Get("/", middlewares.Protected, middlewares.AdminOnly, controllers.GetUsers)
	users.Get("/:id", middlewares.Protected, controllers.GetUser)
	users.Put("/:id", middlewares.Protected, controllers.UpdateUser)
	users.Delete("/:id", middlewares.AdminOnly, controllers.DeleteUser)

	// Barang. Semua endpoint butuh login; tambah/ubah/hapus hanya admin.
	items := api.Group("/items", middlewares.Protected)
	items.Get("/", controllers.GetItems)
	items.Get("/:id", controllers.GetItem)
	items.Post("/", middlewares.AdminOnly, controllers.CreateItem)
	items.Put("/:id", middlewares.AdminOnly, controllers.UpdateItem)
	items.Delete("/:id", middlewares.AdminOnly, controllers.DeleteItem)
}
