package lembaga

import (
	"strconv"

	"github.com/gofiber/fiber/v3"

	"SGEInventory_BE/domain/contracts"
	"SGEInventory_BE/domain/dto"
	"SGEInventory_BE/domain/errx"
	"SGEInventory_BE/pkg/response"
	"SGEInventory_BE/pkg/validator"
)

// LembagaHandler menerima request HTTP lembaga, memanggil service, lalu membalas JSON.
type LembagaHandler struct {
	svc contracts.LembagaService
}

func NewLembagaHandler(svc contracts.LembagaService) *LembagaHandler {
	return &LembagaHandler{svc: svc}
}

// Create: POST /api/lembaga (admin)
func (h *LembagaHandler) Create(c fiber.Ctx) error {
	var req dto.CreateLembagaRequest
	if err := c.Bind().Body(&req); err != nil {
		return errx.Validation(err)
	}
	if err := validator.ValidateStruct(req); err != nil {
		return errx.Validation(err)
	}

	res, err := h.svc.CreateLembaga(c.Context(), req)
	if err != nil {
		return err
	}
	return response.Success(c, fiber.StatusCreated, "Lembaga berhasil ditambahkan", res)
}

// GetAll: GET /api/lembaga?page=1&limit=10&search=bem
func (h *LembagaHandler) GetAll(c fiber.Ctx) error {
	query := dto.LembagaListQuery{
		Page:   fiber.Query[int](c, "page", 1),
		Limit:  fiber.Query[int](c, "limit", 10),
		Search: c.Query("search"),
	}

	res, page, err := h.svc.GetLembagas(c.Context(), query)
	if err != nil {
		return err
	}
	return response.SuccessWithMeta(c, "Daftar lembaga berhasil diambil", res, page)
}

// GetByID: GET /api/lembaga/:id
func (h *LembagaHandler) GetByID(c fiber.Ctx) error {
	id, err := parseID(c)
	if err != nil {
		return err
	}

	res, err := h.svc.GetLembaga(c.Context(), id)
	if err != nil {
		return err
	}
	return response.Success(c, fiber.StatusOK, "Lembaga ditemukan", res)
}

// Update: PUT /api/lembaga/:id (admin)
func (h *LembagaHandler) Update(c fiber.Ctx) error {
	id, err := parseID(c)
	if err != nil {
		return err
	}

	var req dto.UpdateLembagaRequest
	if err := c.Bind().Body(&req); err != nil {
		return errx.Validation(err)
	}
	if err := validator.ValidateStruct(req); err != nil {
		return errx.Validation(err)
	}

	res, err := h.svc.UpdateLembaga(c.Context(), id, req)
	if err != nil {
		return err
	}
	return response.Success(c, fiber.StatusOK, "Lembaga berhasil diperbarui", res)
}

// Delete: DELETE /api/lembaga/:id (admin; soft delete)
func (h *LembagaHandler) Delete(c fiber.Ctx) error {
	id, err := parseID(c)
	if err != nil {
		return err
	}

	if err := h.svc.DeleteLembaga(c.Context(), id); err != nil {
		return err
	}
	return response.Message(c, fiber.StatusOK, "Lembaga berhasil dihapus")
}

// parseID mengubah :id di URL menjadi angka.
func parseID(c fiber.Ctx) (uint, error) {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil || id < 1 {
		return 0, errx.ErrInvalidID
	}
	return uint(id), nil
}
