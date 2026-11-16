// Package response membuat semua respons sukses berbentuk sama: {"message", "data", "meta"}.
// Respons error dibuat oleh middlewares.ErrorHandler.
package response

import (
	"github.com/gofiber/fiber/v3"

	"SGEInventory_BE/domain/dto"
)

// Success membalas {"message": ..., "data": ...}.
func Success(c fiber.Ctx, status int, message string, data any) error {
	return c.Status(status).JSON(dto.APIResponse{Message: message, Data: data})
}

// SuccessWithMeta membalas {"message": ..., "data": ..., "meta": ...} untuk endpoint daftar (list).
func SuccessWithMeta(c fiber.Ctx, message string, data, meta any) error {
	return c.Status(fiber.StatusOK).JSON(dto.APIResponse{Message: message, Data: data, Meta: meta})
}

// Message membalas {"message": ...} saja, tanpa data.
func Message(c fiber.Ctx, status int, message string) error {
	return c.Status(status).JSON(dto.APIResponse{Message: message})
}
