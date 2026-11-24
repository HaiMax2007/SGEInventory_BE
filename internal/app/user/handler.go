package user

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

// UserHandler menerima request HTTP, memanggil service, lalu membalas JSON.
// Handler dibuat "tipis": tidak ada aturan bisnis atau query database di sini.
type UserHandler struct {
	svc contracts.UserService
}

func NewUserHandler(svc contracts.UserService) *UserHandler {
	return &UserHandler{svc: svc}
}

// Register: POST /api/users
func (h *UserHandler) Register(c fiber.Ctx) error {
	var req dto.RegisterUserRequest
	if err := c.Bind().Body(&req); err != nil {
		return errx.InvalidBody(err)
	}
	if err := validator.ValidateStruct(&req); err != nil {
		return err // sudah berupa errx (mis. errx.ErrInvalidEmail)
	}

	res, err := h.svc.Register(c.Context(), req)
	if err != nil {
		return err
	}
	return response.Success(c, fiber.StatusCreated, "User berhasil didaftarkan", res)
}

// GetAll: GET /api/users?page=1&limit=10&search=budi&role=USER (admin)
func (h *UserHandler) GetAll(c fiber.Ctx) error {
	query := dto.UserListQuery{
		Page:   fiber.Query[int](c, "page", 1),
		Limit:  fiber.Query[int](c, "limit", 10),
		Search: c.Query("search"),
		Role:   c.Query("role"),
	}

	res, page, err := h.svc.GetUsers(c.Context(), query)
	if err != nil {
		return err
	}
	return response.SuccessWithMeta(c, "Daftar user berhasil diambil", res, page)
}

// GetByID: GET /api/users/:id (admin atau user itu sendiri)
func (h *UserHandler) GetByID(c fiber.Ctx) error {
	id, err := parseID(c)
	if err != nil {
		return err
	}
	if !isAdminOrSelf(c, id) {
		return errx.ErrForbiddenOtherUser
	}

	res, err := h.svc.GetUser(c.Context(), id)
	if err != nil {
		return err
	}
	return response.Success(c, fiber.StatusOK, "User ditemukan", res)
}

// Update: PUT /api/users/:id (admin atau user itu sendiri; hanya field yang diisi yang berubah)
func (h *UserHandler) Update(c fiber.Ctx) error {
	id, err := parseID(c)
	if err != nil {
		return err
	}
	if !isAdminOrSelf(c, id) {
		return errx.ErrForbiddenOtherUser
	}

	var req dto.UpdateUserRequest
	if err := c.Bind().Body(&req); err != nil {
		return errx.InvalidBody(err)
	}
	if err := validator.ValidateStruct(&req); err != nil {
		return err // sudah berupa errx (mis. errx.ErrInvalidEmail)
	}

	res, err := h.svc.UpdateUser(c.Context(), id, req)
	if err != nil {
		return err
	}
	return response.Success(c, fiber.StatusOK, "User berhasil diperbarui", res)
}

// Delete: DELETE /api/users/:id (admin; soft delete)
func (h *UserHandler) Delete(c fiber.Ctx) error {
	id, err := parseID(c)
	if err != nil {
		return err
	}

	if err := h.svc.DeleteUser(c.Context(), id); err != nil {
		return err
	}
	return response.Message(c, fiber.StatusOK, "User berhasil dihapus")
}

// parseID mengubah :id di URL menjadi angka. Penting: jangan pernah mengoper teks mentah
// ke query GORM, karena GORM menganggapnya potongan SQL.
func parseID(c fiber.Ctx) (uint, error) {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil || id < 1 {
		return 0, errx.ErrInvalidID
	}
	return uint(id), nil
}

// isAdminOrSelf bernilai true jika yang login adalah ADMIN atau pemilik data (id sama).
// userID & role diisi oleh middlewares.Protected.
func isAdminOrSelf(c fiber.Ctx, id uint) bool {
	if fiber.Locals[string](c, "role") == enums.RoleAdmin {
		return true
	}
	return fiber.Locals[uint](c, "userID") == id
}
