package main

import (
	"log"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/cors"
	"github.com/gofiber/fiber/v3/middleware/logger"
	"github.com/gofiber/fiber/v3/middleware/recover"
	"github.com/gofiber/fiber/v3/middleware/static"

	"SGEInventory_BE/domain/contracts"
	"SGEInventory_BE/internal/app/auth"
	"SGEInventory_BE/internal/app/borrowing"
	"SGEInventory_BE/internal/app/consignment"
	"SGEInventory_BE/internal/app/item"
	"SGEInventory_BE/internal/app/lembaga"
	"SGEInventory_BE/internal/app/user"
	"SGEInventory_BE/internal/app/userlog"
	"SGEInventory_BE/internal/config"
	"SGEInventory_BE/internal/infra/database"
	"SGEInventory_BE/internal/infra/repository"
	"SGEInventory_BE/internal/infra/storage"
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
	consignmentRepo := repository.NewConsignmentRepository(db)
	userLogRepo := repository.NewUserLogRepository(db)
	transactor := repository.NewTransactor(db) // menjalankan beberapa repository dalam satu transaksi

	// Tempat foto pengembalian: Cloudinary jika CLOUDINARY_URL diisi, folder lokal jika kosong (dev).
	fileStorage, err := newFileStorage(cfg)
	if err != nil {
		log.Fatal(err)
	}

	userSvc := user.NewUserService(userRepo)
	// borrowingSvc dibuat sebelum itemSvc: item service memakainya untuk menolak otomatis pengajuan
	// PENDING saat total barang dikurangi.
	borrowingSvc := borrowing.NewBorrowingService(borrowingRepo, userLogRepo, itemRepo, lembagaRepo, transactor, fileStorage)
	itemSvc := item.NewItemService(itemRepo, lembagaRepo, borrowingSvc, transactor)
	consignmentSvc := consignment.NewConsignmentService(consignmentRepo, userLogRepo, lembagaRepo, transactor, fileStorage)
	lembagaSvc := lembaga.NewLembagaService(lembagaRepo)
	userLogSvc := userlog.NewUserLogService(userLogRepo)
	authSvc := auth.NewAuthService(userRepo, cfg.JWTSecret)

	userHandler := user.NewUserHandler(userSvc)
	itemHandler := item.NewItemHandler(itemSvc)
	lembagaHandler := lembaga.NewLembagaHandler(lembagaSvc)
	borrowingHandler := borrowing.NewBorrowingHandler(borrowingSvc)
	consignmentHandler := consignment.NewConsignmentHandler(consignmentSvc)
	userLogHandler := userlog.NewUserLogHandler(userLogSvc)
	authHandler := auth.NewAuthHandler(authSvc)

	// 5. Buat aplikasi Fiber. Semua error dari handler diubah menjadi JSON oleh ErrorHandler.
	// BodyLimit 6 MB: foto pengembalian maksimal 5 MB + sedikit ruang untuk data multipart lainnya
	// (default Fiber 4 MB). Body yang lebih besar langsung ditolak Fiber dengan 413.
	app := fiber.New(fiber.Config{
		ErrorHandler: middlewares.ErrorHandler,
		BodyLimit:    6 << 20,
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

	// File unggahan di folder lokal (hanya jika TIDAK memakai Cloudinary): GET /uploads/returns/<nama acak>.jpg.
	// Nama file acak sehingga tidak bisa ditebak; daftar isi folder tidak ditampilkan (Browse: false).
	// nosniff: browser tidak boleh menebak jenis file sendiri. CacheDuration -1: file tidak disimpan di memori,
	// jadi foto yang ditolak admin (sudah dihapus) langsung 404.
	if cfg.CloudinaryURL == "" {
		app.Get(storage.URLPrefix+"*", static.New(cfg.UploadDir, static.Config{
			CacheDuration: -1,
			ModifyResponse: func(c fiber.Ctx) error {
				c.Set("X-Content-Type-Options", "nosniff")
				return nil
			},
		}))
	}

	protected := middlewares.Protected(cfg.JWTSecret) // middleware cek token, dipakai di banyak route
	api := app.Group("/api")
	auth.RegisterRoutes(api, authHandler, protected)
	user.RegisterRoutes(api, userHandler, protected)
	item.RegisterRoutes(api, itemHandler, protected)
	lembaga.RegisterRoutes(api, lembagaHandler, protected)
	borrowing.RegisterRoutes(api, borrowingHandler, protected)
	consignment.RegisterRoutes(api, consignmentHandler, protected)
	userlog.RegisterRoutes(api, userLogHandler, protected)

	// 8. Jalankan server
	log.Fatal(app.Listen(":" + cfg.AppPort))
}

// newFileStorage memilih tempat foto pengembalian. CLOUDINARY_URL kosong hanya mungkin di development
// (config.LoadConfig menolaknya di production). Nilai CLOUDINARY_URL tidak pernah dicetak.
func newFileStorage(cfg *config.Config) (contracts.FileStorage, error) {
	if cfg.CloudinaryURL == "" {
		log.Printf("penyimpanan foto: folder lokal %q (CLOUDINARY_URL kosong)", cfg.UploadDir)
		return storage.NewLocalStorage(cfg.UploadDir)
	}
	log.Printf("penyimpanan foto: Cloudinary (folder %q)", cfg.CloudinaryFolder)
	return storage.NewCloudinaryStorage(cfg.CloudinaryURL, cfg.CloudinaryFolder)
}
