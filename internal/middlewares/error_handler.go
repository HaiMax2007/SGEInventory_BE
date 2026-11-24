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
//   - errx.DomainError → status sesuai Kind, pesan dari Message (+ "error" jika ada detail validasi)
//   - *fiber.Error (mis. 404 URL tidak ada) → status dari Fiber, pesan Bahasa Indonesia (lihat fiberMessage)
//   - error lain (bug / database mati) → dicatat di log, client hanya menerima 500
func ErrorHandler(c fiber.Ctx, err error) error {
	if domainErr, ok := errx.As(err); ok {
		res := dto.APIResponse{Message: domainErr.Message}
		if domainErr.Err != nil {
			res.Error = domainErr.Err.Error() // detail validasi (hanya dari errx.Validation)
		}
		return c.Status(statusFromKind(domainErr.Kind)).JSON(res)
	}

	var fiberErr *fiber.Error
	if errors.As(err, &fiberErr) {
		return c.Status(fiberErr.Code).JSON(dto.APIResponse{Message: fiberMessage(fiberErr)})
	}

	log.Println("server error:", err)
	return c.Status(fiber.StatusInternalServerError).JSON(dto.APIResponse{Message: "Terjadi kesalahan pada server"})
}

// fiberMessage menerjemahkan error bawaan Fiber yang paling sering muncul.
// Kode lain memakai pesan asli dari Fiber.
func fiberMessage(err *fiber.Error) string {
	switch err.Code {
	case fiber.StatusNotFound:
		return "Endpoint tidak ditemukan"
	case fiber.StatusMethodNotAllowed:
		return "Method tidak diizinkan"
	case fiber.StatusRequestEntityTooLarge:
		return "Ukuran data terlalu besar"
	default:
		return err.Message
	}
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
