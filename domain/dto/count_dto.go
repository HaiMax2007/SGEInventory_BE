package dto

// CountResponse adalah balasan endpoint yang hanya mengirim jumlah data, mis. GET /api/borrowings/count
// → {"total": 12}. Dipakai bersama oleh peminjaman & penitipan.
type CountResponse struct {
	Total int64 `json:"total"`
}
