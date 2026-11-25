package history

import (
	"github.com/gofiber/fiber/v3"

	"SGEInventory_BE/domain/contracts"
	"SGEInventory_BE/domain/dto"
	"SGEInventory_BE/pkg/response"
)

// HistoryHandler menerima request HTTP riwayat pengajuan, memanggil service, lalu membalas JSON.
type HistoryHandler struct {
	svc contracts.HistoryService
}

func NewHistoryHandler(svc contracts.HistoryService) *HistoryHandler {
	return &HistoryHandler{svc: svc}
}

// GetMine: GET /api/history/me?page=1&limit=10 (user; peminjaman + penitipan miliknya, terbaru dulu)
func (h *HistoryHandler) GetMine(c fiber.Ctx) error {
	query := dto.HistoryListQuery{
		Page:  fiber.Query[int](c, "page", 1),
		Limit: fiber.Query[int](c, "limit", 10),
	}

	userID := fiber.Locals[uint](c, "userID") // pemilik token, diisi middlewares.Protected
	res, page, err := h.svc.GetMyHistory(c.Context(), userID, query)
	if err != nil {
		return err
	}
	return response.SuccessWithMeta(c, "Riwayat pengajuan berhasil diambil", res, page)
}
