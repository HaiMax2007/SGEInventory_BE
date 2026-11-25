package dto

import (
	"time"

	"SGEInventory_BE/domain/entity"
)

// UserLogResponse adalah satu baris riwayat (timeline) sebuah pengajuan.
// Dipakai di detail pengajuan, jadi jenis & id pengajuan tidak perlu ditulis ulang.
type UserLogResponse struct {
	Activity  string       `json:"activity"`
	Actor     *UserSummary `json:"actor"` // pelaku: user saat mengajukan, admin saat ACC/menolak
	CreatedAt time.Time    `json:"createdAt"`
}

// NewUserLogResponse mengubah entity.UserLog menjadi UserLogResponse.
// Actor bernilai null jika User belum di-Preload.
func NewUserLogResponse(l *entity.UserLog) UserLogResponse {
	return UserLogResponse{
		Activity:  l.RequestActivity,
		Actor:     NewUserSummary(l.User),
		CreatedAt: l.CreatedAt,
	}
}

// UserLogListQuery adalah query string GET /api/logs?page=1&limit=10&requestType=BORROWING&search=raka.
type UserLogListQuery struct {
	Page        int
	Limit       int
	RequestType string // kosong = semua jenis
	Search      string // cocokkan sebagian nama lengkap (full_name) pelaku, tanpa beda huruf besar/kecil
}

// UserLogListItemResponse adalah satu baris di daftar semua log (admin).
// Berbeda dengan UserLogResponse, di sini jenis & id pengajuan ikut dikirim karena
// daftarnya berisi log dari banyak pengajuan sekaligus.
type UserLogListItemResponse struct {
	ID          uint         `json:"id"`
	RequestType string       `json:"requestType"`
	RequestID   uint         `json:"requestId"`
	Activity    string       `json:"activity"`
	Actor       *UserSummary `json:"actor"`
	CreatedAt   time.Time    `json:"createdAt"`
}

// NewUserLogListItemResponse mengubah entity.UserLog menjadi UserLogListItemResponse.
func NewUserLogListItemResponse(l *entity.UserLog) UserLogListItemResponse {
	return UserLogListItemResponse{
		ID:          l.ID,
		RequestType: l.RequestType,
		RequestID:   l.RequestID,
		Activity:    l.RequestActivity,
		Actor:       NewUserSummary(l.User),
		CreatedAt:   l.CreatedAt,
	}
}
