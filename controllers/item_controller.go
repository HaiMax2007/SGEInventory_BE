package controllers

import (
	"errors"
	"strconv"

	"github.com/gofiber/fiber/v3"
	"gorm.io/gorm"

	"SGEInventory_BE/config"
	"SGEInventory_BE/models"
)

// CreateItem: POST /api/items (admin)
func CreateItem(c fiber.Ctx) error {
	var req models.CreateItemRequest
	if err := c.Bind().Body(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Data tidak valid", "error": err.Error()})
	}

	// Barang baru belum ada yang dipinjam, jadi sisa = total.
	item := models.Item{
		Name:              req.Name,
		TotalQuantity:     req.TotalQuantity,
		AvailableQuantity: req.TotalQuantity,
	}
	if err := config.DB.Create(&item).Error; err != nil {
		return serverError(c, err)
	}

	item.SetStatus()
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"message": "Barang berhasil ditambahkan", "data": item})
}

// GetItems: GET /api/items?page=1&limit=10&search=kabel&available=true
func GetItems(c fiber.Ctx) error {
	page := fiber.Query[int](c, "page", 1)
	limit := fiber.Query[int](c, "limit", 10)
	search := c.Query("search")
	onlyAvailable := c.Query("available") == "true"

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 10
	}

	query := config.DB.Model(&models.Item{})
	if search != "" {
		query = query.Where("name LIKE ?", "%"+search+"%")
	}
	if onlyAvailable {
		query = query.Where("available_quantity > 0")
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return serverError(c, err)
	}

	items := []models.Item{}
	if err := query.Order("id DESC").Offset((page - 1) * limit).Limit(limit).Find(&items).Error; err != nil {
		return serverError(c, err)
	}
	for i := range items {
		items[i].SetStatus()
	}

	return c.JSON(fiber.Map{
		"message": "Daftar barang berhasil diambil",
		"data":    items,
		"meta": fiber.Map{
			"page":       page,
			"limit":      limit,
			"total":      total,
			"totalPages": (int(total) + limit - 1) / limit,
		},
	})
}

// GetItem: GET /api/items/:id
func GetItem(c fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil || id < 1 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "ID tidak valid"})
	}

	var item models.Item
	if err := config.DB.First(&item, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"message": "Barang tidak ditemukan"})
		}
		return serverError(c, err)
	}

	item.SetStatus()
	return c.JSON(fiber.Map{"message": "Barang ditemukan", "data": item})
}

// UpdateItem: PUT /api/items/:id (admin; hanya field yang diisi yang berubah)
func UpdateItem(c fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil || id < 1 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "ID tidak valid"})
	}

	var req models.UpdateItemRequest
	if err := c.Bind().Body(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Data tidak valid", "error": err.Error()})
	}
	if req.Name == "" && req.TotalQuantity == nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Tidak ada data yang diperbarui"})
	}

	var item models.Item
	if err := config.DB.First(&item, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"message": "Barang tidak ditemukan"})
		}
		return serverError(c, err)
	}

	// Pakai map (bukan struct) supaya nilai 0 tetap ikut disimpan.
	// Updates dengan struct akan melewati field yang bernilai 0.
	updates := map[string]any{}
	if req.Name != "" {
		updates["name"] = req.Name
	}
	if req.TotalQuantity != nil {
		newTotal := *req.TotalQuantity
		borrowed := item.TotalQuantity - item.AvailableQuantity // jumlah yang sedang dipinjam

		if newTotal < borrowed {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{
				"message": "Total tidak boleh kurang dari jumlah yang sedang dipinjam (" + strconv.Itoa(borrowed) + ")",
			})
		}
		updates["total_quantity"] = newTotal
		updates["available_quantity"] = newTotal - borrowed
	}

	if err := config.DB.Model(&item).Updates(updates).Error; err != nil {
		return serverError(c, err)
	}

	item.SetStatus()
	return c.JSON(fiber.Map{"message": "Barang berhasil diperbarui", "data": item})
}

// DeleteItem: DELETE /api/items/:id (admin; soft delete)
func DeleteItem(c fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil || id < 1 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "ID tidak valid"})
	}

	var item models.Item
	if err := config.DB.First(&item, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"message": "Barang tidak ditemukan"})
		}
		return serverError(c, err)
	}

	// Barang yang sebagian masih dipinjam tidak boleh dihapus.
	if item.AvailableQuantity < item.TotalQuantity {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{"message": "Barang masih dipinjam, tidak bisa dihapus"})
	}

	if err := config.DB.Delete(&item).Error; err != nil {
		return serverError(c, err)
	}

	return c.JSON(fiber.Map{"message": "Barang berhasil dihapus"})
}
