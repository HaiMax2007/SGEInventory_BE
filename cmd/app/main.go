package main

import (
	"log"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/cors"
	"github.com/gofiber/fiber/v3/middleware/logger"
	"github.com/gofiber/fiber/v3/middleware/recover"

	"SGEInventory_BE/internal/app/auth"
	"SGEInventory_BE/internal/app/borrowing"
	"SGEInventory_BE/internal/app/item"
	"SGEInventory_BE/internal/app/lembaga"
	"SGEInventory_BE/internal/app/user"
	"SGEInventory_BE/internal/app/userlog"
	"SGEInventory_BE/internal/config"
	"SGEInventory_BE/internal/infra/database"
	"SGEInventory_BE/internal/infra/repository"
	"SGEInventory_BE/internal/middlewares"
	"SGEInventory_BE/pkg/response"
)

func main() {
	// 1. Baca konfigurasi dari environment (PORT, DB_*, JWT_SECRET, CORS_ORIGINS)
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatal(err)
	}

	// 2. Sambungkan ke database (PostgreSQL)
	db, err := database.NewPostgres(cfg)
	if err != nil {
		log.Fatal(err)
	}

	// 3. Samakan struktur tabel dengan entity (AutoMigrate),
	//    lalu isi data awal: user Max & Raka + lembaga (khusus development)
	if err := database.Migrate(db); err != nil {
		log.Fatal(err)
	}
	if cfg.IsDevelopment() {
		if err := database.Seed(db); err != nil {
			log.Fatal(err)
		}
	}

	// 4. Rakit lapisan aplikasi (dependency injection):
	//    repository (akses database) → service (aturan bisnis) → handler (HTTP)
	userRepo := repository.NewUserRepository(db)
	itemRepo := repository.NewItemRepository(db)
	lembagaRepo := repository.NewLembagaRepository(db)
	borrowingRepo := repository.NewBorrowingRepository(db)
	userLogRepo := repository.NewUserLogRepository(db)

	userSvc := user.NewUserService(userRepo)
	itemSvc := item.NewItemService(itemRepo, lembagaRepo)
	lembagaSvc := lembaga.NewLembagaService(lembagaRepo)
	borrowingSvc := borrowing.NewBorrowingService(borrowingRepo, userLogRepo, itemRepo, lembagaRepo)
	userLogSvc := userlog.NewUserLogService(userLogRepo)
	authSvc := auth.NewAuthService(userRepo, cfg.JWTSecret)

	userHandler := user.NewUserHandler(userSvc)
	itemHandler := item.NewItemHandler(itemSvc)
	lembagaHandler := lembaga.NewLembagaHandler(lembagaSvc)
	borrowingHandler := borrowing.NewBorrowingHandler(borrowingSvc)
	userLogHandler := userlog.NewUserLogHandler(userLogSvc)
	authHandler := auth.NewAuthHandler(authSvc)

	// 5. Buat aplikasi Fiber. Semua error dari handler diubah menjadi JSON oleh ErrorHandler.
	app := fiber.New(fiber.Config{
		ErrorHandler: middlewares.ErrorHandler,
	})

	// 6. Middleware global: catat request, cegah server mati saat panic,
	//    dan izinkan FE (beda port/domain) memanggil API ini (CORS)
	app.Use(logger.New())
	app.Use(recover.New())
	app.Use(cors.New(cors.Config{
		AllowOrigins: cfg.CORSOrigins,
		AllowHeaders: []string{"Origin", "Content-Type", "Authorization"},
	}))

	// 7. Daftarkan route
	app.Get("/", func(c fiber.Ctx) error {
		return response.Message(c, fiber.StatusOK, "This app is running")
	})

	protected := middlewares.Protected(cfg.JWTSecret) // middleware cek token, dipakai di banyak route
	api := app.Group("/api")
	auth.RegisterRoutes(api, authHandler, protected)
	user.RegisterRoutes(api, userHandler, protected)
	item.RegisterRoutes(api, itemHandler, protected)
	lembaga.RegisterRoutes(api, lembagaHandler, protected)
	borrowing.RegisterRoutes(api, borrowingHandler, protected)
	userlog.RegisterRoutes(api, userLogHandler, protected)

	// 8. Jalankan server
	log.Fatal(app.Listen(":" + cfg.AppPort))
}
