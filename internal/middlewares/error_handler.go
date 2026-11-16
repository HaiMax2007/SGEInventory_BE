package middlewares

import (
	"errors"
	"log"

	"github.com/gofiber/fiber/v3"

	"SGEInventory_BE/domain/dto"
	"SGEInventory_BE/domain/errx"
)

// ErrorHandler adalah satu-satunya tempat error diubah menjadi respons JSON.
// Semua handler, service, dan middleware cukup `return err`.
//
//   - errx.DomainError → status sesuai Kind, pesan dari Message
//   - *fiber.Error (mis. 404 URL tidak ada) → status & pesan dari Fiber
//   - error lain (bug / database mati) → dicatat di log, client hanya menerima 500
func ErrorHandler(c fiber.Ctx, err error) error {
	if domainErr, ok := errx.As(err); ok {
		res := dto.APIResponse{Message: domainErr.Message}
		if domainErr.Err != nil {
			res.Error = domainErr.Err.Error() // detail validasi
		}
		return c.Status(statusFromKind(domainErr.Kind)).JSON(res)
	}

	var fiberErr *fiber.Error
	if errors.As(err, &fiberErr) {
		return c.Status(fiberErr.Code).JSON(dto.APIResponse{Message: fiberErr.Message})
	}

	log.Println("server error:", err)
	return c.Status(fiber.StatusInternalServerError).JSON(dto.APIResponse{Message: "Terjadi kesalahan pada server"})
}

func statusFromKind(kind errx.Kind) int {
	switch kind {
	case errx.KindInvalid:
		return fiber.StatusBadRequest
	case errx.KindUnauthorized:
		return fiber.StatusUnauthorized
	case errx.KindForbidden:
		return fiber.StatusForbidden
	case errx.KindNotFound:
		return fiber.StatusNotFound
	case errx.KindConflict:
		return fiber.StatusConflict
	default:
		return fiber.StatusInternalServerError
	}
}
