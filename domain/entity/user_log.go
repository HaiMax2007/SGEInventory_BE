package entity

import "time"

// UserLog adalah cermin tabel log_users: catatan setiap aktivitas pada sebuah pengajuan
// (diajukan, di-ACC/ditolak admin, pengembalian, ...). Hanya di-INSERT, tidak pernah diubah.
//
//   - UserID = PELAKU aktivitas (user saat mengajukan, admin saat ACC/menolak; untuk AUTO_REJECTED
//     = admin yang tindakannya membuat stok berkurang).
//     Pemilik pengajuan disimpan di tabel pengajuannya (mis. borrowings.user_id).
//   - RequestType + RequestID = pengajuan yang dimaksud, mis. ("BORROWING", 12).
//     Satu tabel log dipakai dua jenis form (peminjaman & penitipan), jadi RequestID tidak bisa
//     diberi FOREIGN KEY ke satu tabel tertentu; kebenarannya dijaga oleh aplikasi.
//
// Riwayat satu pengajuan: WHERE request_type = ? AND request_id = ? (index idx_log_users_request).
type UserLog struct {
	ID              uint      `gorm:"primaryKey"`
	UserID          uint      `gorm:"not null;index:idx_log_users_user_id"`
	User            *User     `gorm:"constraint:OnUpdate:CASCADE,OnDelete:RESTRICT"`
	RequestType     string    `gorm:"size:20;not null;index:idx_log_users_request,priority:1;check:chk_log_users_request_type,request_type IN ('BORROWING','CONSIGNMENT')"`
	RequestID       uint      `gorm:"not null;index:idx_log_users_request,priority:2"`
	RequestActivity string    `gorm:"size:30;not null;check:chk_log_users_request_activity,request_activity IN ('SUBMITTED','APPROVED','REJECTED','AUTO_REJECTED','RETURN_SUBMITTED','RETURN_APPROVED','RETURN_REJECTED')"`
	CreatedAt       time.Time // waktu aktivitas terjadi
}

func (UserLog) TableName() string {
	return "log_users"
}

// NewUserLog membuat catatan aktivitas baru.
func NewUserLog(userID uint, requestType string, requestID uint, activity string) *UserLog {
	return &UserLog{
		UserID:          userID,
		RequestType:     requestType,
		RequestID:       requestID,
		RequestActivity: activity,
	}
}
