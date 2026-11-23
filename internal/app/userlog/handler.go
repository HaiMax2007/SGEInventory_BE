package userlog

import (
	"github.com/gofiber/fiber/v3"

	"SGEInventory_BE/domain/contracts"
	"SGEInventory_BE/domain/dto"
	"SGEInventory_BE/pkg/response"
)

// UserLogHandler menerima request HTTP riwayat aktivitas, memanggil service, lalu membalas JSON.
type UserLogHandler struct {
	svc contracts.UserLogService
}

func NewUserLogHandler(svc contracts.UserLogService) *UserLogHandler {
	return &UserLogHandler{svc: svc}
}

// GetAll: GET /api/logs?page=1&limit=10&requestType=BORROWING (admin; semua log, terbaru dulu)
func (h *UserLogHandler) GetAll(c fiber.Ctx) error {
	query := dto.UserLogListQuery{
		Page:        fiber.Query[int](c, "page", 1),
		Limit:       fiber.Query[int](c, "limit", 10),
		RequestType: c.Query("requestType"),
	}

	res, page, err := h.svc.GetLogs(c.Context(), query)
	if err != nil {
		return err
	}
	return response.SuccessWithMeta(c, "Daftar riwayat aktivitas berhasil diambil", res, page)
}
