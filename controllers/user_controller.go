package controllers

import (
	"errors"
	"log"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v3"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"SGEInventory_BE/config"
	"SGEInventory_BE/models"
)

// bcrypt hanya bisa menerima password maksimal 72 byte.
const maxPasswordBytes = 72

// Register: POST /api/users
func Register(c fiber.Ctx) error {
	var req models.RegisterUserRequest
	if err := c.Bind().Body(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Data tidak valid", "error": err.Error()})
	}
	if len(req.Password) > maxPasswordBytes {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Password terlalu panjang (maksimal 72 karakter)"})
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return serverError(c, err)
	}

	user := models.User{
		FullName: req.FullName,
		Email:    strings.ToLower(req.Email),
		Password: string(hashedPassword),
		Contact:  req.Contact,
		Role:     models.RoleUser, // role selalu USER, tidak bisa dipilih sendiri
	}
	if err := config.DB.Create(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{"message": "Email sudah terdaftar"})
		}
		return serverError(c, err)
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"message": "User berhasil didaftarkan", "data": user})
}

// GetUsers: GET /api/users?page=1&limit=10&search=budi&role=USER
func GetUsers(c fiber.Ctx) error {
	page := fiber.Query[int](c, "page", 1)
	limit := fiber.Query[int](c, "limit", 10)
	search := c.Query("search")
	role := c.Query("role")

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 10
	}

	query := config.DB.Model(&models.User{})
	if search != "" {
		query = query.Where("full_name LIKE ? OR email LIKE ?", "%"+search+"%", "%"+search+"%")
	}
	if role != "" {
		query = query.Where("role = ?", role)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return serverError(c, err)
	}

	users := []models.User{}
	if err := query.Order("id DESC").Offset((page - 1) * limit).Limit(limit).Find(&users).Error; err != nil {
		return serverError(c, err)
	}

	return c.JSON(fiber.Map{
		"message": "Daftar user berhasil diambil",
		"data":    users,
		"meta": fiber.Map{
			"page":       page,
			"limit":      limit,
			"total":      total,
			"totalPages": (int(total) + limit - 1) / limit,
		},
	})
}

// GetUser: GET /api/users/:id
func GetUser(c fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil || id < 1 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "ID tidak valid"})
	}
	if !isAdminOrSelf(c, id) {
		return forbidden(c)
	}

	var user models.User
	if err := config.DB.First(&user, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"message": "User tidak ditemukan"})
		}
		return serverError(c, err)
	}

	return c.JSON(fiber.Map{"message": "User ditemukan", "data": user})
}

// UpdateUser: PUT /api/users/:id (hanya field yang diisi yang berubah)
func UpdateUser(c fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil || id < 1 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "ID tidak valid"})
	}
	if !isAdminOrSelf(c, id) {
		return forbidden(c)
	}

	var req models.UpdateUserRequest
	if err := c.Bind().Body(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Data tidak valid", "error": err.Error()})
	}
	if req == (models.UpdateUserRequest{}) {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Tidak ada data yang diperbarui"})
	}

	var user models.User
	if err := config.DB.First(&user, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"message": "User tidak ditemukan"})
		}
		return serverError(c, err)
	}

	// Updates dengan struct otomatis melewati field yang kosong.
	updates := models.User{
		FullName: req.FullName,
		Email:    strings.ToLower(req.Email),
		Contact:  req.Contact,
	}

	if err := config.DB.Model(&user).Updates(updates).Error; err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{"message": "Email sudah terdaftar"})
		}
		return serverError(c, err)
	}

	return c.JSON(fiber.Map{"message": "User berhasil diperbarui", "data": user})
}

// DeleteUser: DELETE /api/users/:id (soft delete, baris tetap ada dengan deleted_at terisi)
func DeleteUser(c fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil || id < 1 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "ID tidak valid"})
	}
	if !isAdminOrSelf(c, id) {
		return forbidden(c)
	}

	result := config.DB.Delete(&models.User{}, id)
	if result.Error != nil {
		return serverError(c, result.Error)
	}
	if result.RowsAffected == 0 {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"message": "User tidak ditemukan"})
	}

	return c.JSON(fiber.Map{"message": "User berhasil dihapus"})
}

// isAdminOrSelf bernilai true jika yang login adalah ADMIN atau pemilik data (id sama).
// userID & role diisi oleh middlewares.Protected.
func isAdminOrSelf(c fiber.Ctx, id int) bool {
	if fiber.Locals[string](c, "role") == models.RoleAdmin {
		return true
	}
	return fiber.Locals[uint](c, "userID") == uint(id)
}

func forbidden(c fiber.Ctx) error {
	return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"message": "Kamu tidak boleh mengakses data user lain"})
}

// serverError mencatat error ke log lalu membalas 500 tanpa membocorkan detailnya ke client.
func serverError(c fiber.Ctx, err error) error {
	log.Println("server error:", err)
	return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Terjadi kesalahan pada server"})
}
