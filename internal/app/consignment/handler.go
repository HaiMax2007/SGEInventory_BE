package consignment

import (
	"errors"

	"github.com/gofiber/fiber/v3"

	"SGEInventory_BE/domain/contracts"
	"SGEInventory_BE/domain/dto"
	"SGEInventory_BE/domain/errx"
	"SGEInventory_BE/pkg/response"
	"SGEInventory_BE/pkg/validator"
)

// ConsignmentHandler menerima request HTTP penitipan, memanggil service, lalu membalas JSON.
type ConsignmentHandler struct {
	svc contracts.ConsignmentService
}

func NewConsignmentHandler(svc contracts.ConsignmentService) *ConsignmentHandler {
	return &ConsignmentHandler{svc: svc}
}

// Create: POST /api/consignments (khusus role USER, multipart/form-data karena ada foto)
//
// Field: lembagaId, eventName, items (teks JSON: [{"name":"Kursi","quantity":3}]), startDate, endDate,
// photo (file), notes (opsional). Aturan per field dicek lewat tag `validate`; aturan antar-field di service.
func (h *ConsignmentHandler) Create(c fiber.Ctx) error {
	// Form wajib multipart (berisi file). Body JSON / form biasa langsung ditolak dengan pesan yang jelas.
	if _, err := c.MultipartForm(); err != nil {
		return errx.ErrInvalidForm
	}

	var req dto.CreateConsignmentRequest
	if err := c.Bind().Body(&req); err != nil {
		return formBindError(err)
	}
	req.ParseItems(c.FormValue("items"))
	// Error FormFile (field tidak ada) diabaikan: Photo tetap nil dan ValidateStruct membalasnya
	// dengan errx.ErrInvalidConsignmentPhoto.
	req.Photo, _ = c.FormFile("photo")
	if err := validator.ValidateStruct(&req); err != nil {
		return err // sudah berupa errx (mis. errx.ErrInvalidEventName)
	}

	userID := fiber.Locals[uint](c, "userID") // penitip = pemilik token, diisi middlewares.Protected
	res, err := h.svc.CreateConsignment(c.Context(), userID, req)
	if err != nil {
		return err
	}
	return response.Success(c, fiber.StatusCreated, "Pengajuan penitipan berhasil dikirim", res)
}

// formBindError mengubah error c.Bind().Body pada form menjadi error yang jelas untuk FE.
// Satu-satunya field angka di form ini adalah lembagaId, jadi field yang gagal diubah tipenya pasti angka
// (mis. lembagaId=abc → "Field lembagaId harus berupa angka bulat positif"). Pesan asli Fiber tidak dikirim.
func formBindError(err error) error {
	var bindErr *fiber.BindError
	if errors.As(err, &bindErr) && bindErr.Field != "" {
		return errx.NewInvalidFieldType(bindErr.Field, "angka bulat positif")
	}
	return errx.ErrInvalidForm
}

// GetMine: GET /api/consignments/me?page=1&limit=10 (penitipan milik user yang login, dari token)
func (h *ConsignmentHandler) GetMine(c fiber.Ctx) error {
	query := dto.ConsignmentListQuery{
		Page:  fiber.Query[int](c, "page", 1),
		Limit: fiber.Query[int](c, "limit", 10),
	}

	userID := fiber.Locals[uint](c, "userID")
	res, page, err := h.svc.GetMyConsignments(c.Context(), userID, query)
	if err != nil {
		return err
	}
	return response.SuccessWithMeta(c, "Daftar pengajuan penitipan berhasil diambil", res, page)
}

// GetAll: GET /api/consignments?page=1&limit=10&status=PENDING (admin; semua penitipan)
func (h *ConsignmentHandler) GetAll(c fiber.Ctx) error {
	query := dto.ConsignmentListQuery{
		Page:   fiber.Query[int](c, "page", 1),
		Limit:  fiber.Query[int](c, "limit", 10),
		Status: c.Query("status"),
	}

	res, page, err := h.svc.GetConsignments(c.Context(), query)
	if err != nil {
		return err
	}
	return response.SuccessWithMeta(c, "Daftar semua pengajuan penitipan berhasil diambil", res, page)
}
