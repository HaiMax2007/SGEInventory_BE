package dto

import "time"

// HistoryListQuery adalah query string GET /api/history/me?page=1&limit=10.
type HistoryListQuery struct {
	Page  int
	Limit int
}

// HistoryResponse adalah satu baris riwayat pengajuan user: peminjaman ATAU penitipan.
// Type menentukan field mana yang terisi; field lainnya null. Isi borrowing/consignment sama dengan
// GET /api/borrowings/me dan GET /api/consignments/me (tanpa timeline).
type HistoryResponse struct {
	Type        string               `json:"type"`        // BORROWING | CONSIGNMENT (enums.RequestType*)
	CreatedAt   time.Time            `json:"createdAt"`   // waktu pengajuan dibuat; dasar urutan riwayat
	Borrowing   *BorrowingResponse   `json:"borrowing"`   // null jika type CONSIGNMENT
	Consignment *ConsignmentResponse `json:"consignment"` // null jika type BORROWING
}
