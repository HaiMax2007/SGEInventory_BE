package item

import (
	"strconv"

	"github.com/gofiber/fiber/v3"

	"SGEInventory_BE/domain/contracts"
	"SGEInventory_BE/domain/dto"
	"SGEInventory_BE/domain/enums"
	"SGEInventory_BE/domain/errx"
	"SGEInventory_BE/pkg/response"
	"SGEInventory_BE/pkg/validator"
)

// ItemHandler menerima request HTTP barang, memanggil service, lalu membalas JSON.
type ItemHandler struct {
	svc contracts.ItemService
}

func NewItemHandler(svc contracts.ItemService) *ItemHandler {
	return &ItemHandler{svc: svc}
}

// Create: POST /api/items (admin)
func (h *ItemHandler) Create(c fiber.Ctx) error {
	var req dto.CreateItemRequest
	if err := c.Bind().Body(&req); err != nil {
		return errx.InvalidBody(err)
	}
	if err := validator.ValidateStruct(&req); err != nil {
		return err // sudah berupa errx (mis. errx.ErrInvalidEmail)
	}

	res, err := h.svc.CreateItem(c.Context(), req)
	if err != nil {
		return err
	}
	return response.Success(c, fiber.StatusCreated, "Barang berhasil ditambahkan", res)
}

// GetAll: GET /api/items?page=1&limit=10&search=kabel&available=true&type=CONSIGNMENT
// User hanya melihat barang BORROWABLE; ?type= hanya berlaku untuk admin.
func (h *ItemHandler) GetAll(c fiber.Ctx) error {
	query := dto.ItemListQuery{
		Page:          fiber.Query[int](c, "page", 1),
		Limit:         fiber.Query[int](c, "limit", 10),
		Search:        c.Query("search"),
		OnlyAvailable: c.Query("available") == "true",
		Type:          c.Query("type"),
	}

	res, page, err := h.svc.GetItems(c.Context(), query, isAdmin(c))
	if err != nil {
		return err
	}
	return response.SuccessWithMeta(c, "Daftar barang berhasil diambil", res, page)
}

// GetByID: GET /api/items/:id (barang titipan hanya untuk admin; user → 404)
func (h *ItemHandler) GetByID(c fiber.Ctx) error {
	id, err := parseID(c)
	if err != nil {
		return err
	}

	res, err := h.svc.GetItem(c.Context(), id, isAdmin(c))
	if err != nil {
		return err
	}
	return response.Success(c, fiber.StatusOK, "Barang ditemukan", res)
}

// Update: PUT /api/items/:id (admin; hanya field yang diisi yang berubah)
func (h *ItemHandler) Update(c fiber.Ctx) error {
	id, err := parseID(c)
	if err != nil {
		return err
	}

	var req dto.UpdateItemRequest
	if err := c.Bind().Body(&req); err != nil {
		return errx.InvalidBody(err)
	}
	if err := validator.ValidateStruct(&req); err != nil {
		return err // sudah berupa errx (mis. errx.ErrInvalidEmail)
	}

	adminID := fiber.Locals[uint](c, "userID") // pelaku jika ada pengajuan yang ditolak otomatis
	res, err := h.svc.UpdateItem(c.Context(), id, adminID, req)
	if err != nil {
		return err
	}
	return response.Success(c, fiber.StatusOK, "Barang berhasil diperbarui", res)
}

// Delete: DELETE /api/items/:id (admin; soft delete)
func (h *ItemHandler) Delete(c fiber.Ctx) error {
	id, err := parseID(c)
	if err != nil {
		return err
	}

	if err := h.svc.DeleteItem(c.Context(), id); err != nil {
		return err
	}
	return response.Message(c, fiber.StatusOK, "Barang berhasil dihapus")
}

// isAdmin bernilai true jika yang login ADMIN (role diisi middlewares.Protected).
func isAdmin(c fiber.Ctx) bool {
	return fiber.Locals[string](c, "role") == enums.RoleAdmin
}

// parseID mengubah :id di URL menjadi angka.
func parseID(c fiber.Ctx) (uint, error) {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil || id < 1 {
		return 0, errx.ErrInvalidID
	}
	return uint(id), nil
}
