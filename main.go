package main

import (
	"log"
	"os"
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/cors"
	"github.com/gofiber/fiber/v3/middleware/logger"
	"github.com/gofiber/fiber/v3/middleware/recover"

	"SGEInventory_BE/config"
	"SGEInventory_BE/routes"
)

func main() {
	// 1. Sambungkan ke database
	if err := config.ConnectDatabase(); err != nil {
		log.Fatal(err)
	}

	// 2. Pastikan secret untuk menandatangani JWT ada dan cukup panjang
	if len(os.Getenv("JWT_SECRET")) < 32 {
		log.Fatal("JWT_SECRET wajib diisi, minimal 32 karakter")
	}

	// 3. Buat aplikasi Fiber; validator mengecek tag `validate` saat c.Bind().Body()
	app := fiber.New(fiber.Config{
		StructValidator: config.NewValidator(),
	})

	// 4. Middleware: catat setiap request, cegah server mati saat panic,
	//    dan izinkan FE (beda port/domain) memanggil API ini (CORS)
	app.Use(logger.New())
	app.Use(recover.New())

	corsOrigins := os.Getenv("CORS_ORIGINS")
	if corsOrigins == "" {
		corsOrigins = "http://localhost:5173,http://localhost:3001"
	}
	app.Use(cors.New(cors.Config{
		AllowOrigins: strings.Split(corsOrigins, ","),
		AllowHeaders: []string{"Origin", "Content-Type", "Authorization"},
	}))

	// 5. Daftarkan route
	app.Get("/", func(c fiber.Ctx) error {
		return c.JSON(fiber.Map{"message": "This app is running"})
	})
	routes.SetupRoutes(app)

	// 6. Jalankan server
	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}
	log.Fatal(app.Listen(":" + port))
}
