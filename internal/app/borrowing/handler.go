package borrowing

import (
	"strconv"

	"github.com/gofiber/fiber/v3"

	"SGEInventory_BE/domain/contracts"
	"SGEInventory_BE/domain/dto"
	"SGEInventory_BE/domain/enums"
	"SGEInventory_BE/domain/errx"
	"SGEInventory_BE/pkg/response"
)

// BorrowingHandler menerima request HTTP peminjaman, memanggil service, lalu membalas JSON.
type BorrowingHandler struct {
	svc contracts.BorrowingService
}

func NewBorrowingHandler(svc contracts.BorrowingService) *BorrowingHandler {
	return &BorrowingHandler{svc: svc}
}

// Create: POST /api/borrowings (khusus role USER)
// Tidak memakai validator.ValidateStruct: semua aturan dicek di service dengan pesan error masing-masing.
func (h *BorrowingHandler) Create(c fiber.Ctx) error {
	var req dto.CreateBorrowingRequest
	if err := c.Bind().Body(&req); err != nil {
		return errx.Validation(err) // JSON rusak atau tipe data salah (mis. quantity berupa teks)
	}

	userID := fiber.Locals[uint](c, "userID") // peminjam = pemilik token, diisi middlewares.Protected
	res, err := h.svc.CreateBorrowing(c.Context(), userID, req)
	if err != nil {
		return err
	}
	return response.Success(c, fiber.StatusCreated, "Pengajuan peminjaman berhasil dikirim", res)
}

// GetMine: GET /api/borrowings/me?page=1&limit=10 (pengajuan milik user yang login)
func (h *BorrowingHandler) GetMine(c fiber.Ctx) error {
	query := dto.BorrowingListQuery{
		Page:  fiber.Query[int](c, "page", 1),
		Limit: fiber.Query[int](c, "limit", 10),
	}

	userID := fiber.Locals[uint](c, "userID")
	res, page, err := h.svc.GetMyBorrowings(c.Context(), userID, query)
	if err != nil {
		return err
	}
	return response.SuccessWithMeta(c, "Daftar pengajuan peminjaman berhasil diambil", res, page)
}

// GetAll: GET /api/borrowings?page=1&limit=10&status=PENDING (admin; semua pengajuan)
func (h *BorrowingHandler) GetAll(c fiber.Ctx) error {
	query := dto.BorrowingListQuery{
		Page:   fiber.Query[int](c, "page", 1),
		Limit:  fiber.Query[int](c, "limit", 10),
		Status: c.Query("status"),
	}

	res, page, err := h.svc.GetBorrowings(c.Context(), query)
	if err != nil {
		return err
	}
	return response.SuccessWithMeta(c, "Daftar semua pengajuan peminjaman berhasil diambil", res, page)
}

// GetByID: GET /api/borrowings/:id (pemilik pengajuan atau admin; berisi timeline)
func (h *BorrowingHandler) GetByID(c fiber.Ctx) error {
	id, err := parseID(c)
	if err != nil {
		return err
	}

	userID := fiber.Locals[uint](c, "userID")
	isAdmin := fiber.Locals[string](c, "role") == enums.RoleAdmin
	res, err := h.svc.GetBorrowing(c.Context(), id, userID, isAdmin)
	if err != nil {
		return err
	}
	return response.Success(c, fiber.StatusOK, "Pengajuan peminjaman ditemukan", res)
}

// parseID mengubah :id di URL menjadi angka.
func parseID(c fiber.Ctx) (uint, error) {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil || id < 1 {
		return 0, errx.ErrInvalidID
	}
	return uint(id), nil
}
