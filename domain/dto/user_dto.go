package dto

import (
	"time"

	"SGEInventory_BE/domain/entity"
)

// RegisterUserRequest adalah isi body POST /api/users.
// Tidak ada field role: user baru selalu USER.
type RegisterUserRequest struct {
	FullName string `json:"fullName" validate:"required,min=3,max=255"`
	NIM      string `json:"nim" validate:"required,numeric,min=8,max=20"` // angka saja, disimpan sebagai teks
	Email    string `json:"email" validate:"required,email,max=255"`
	Password string `json:"password" validate:"required,min=8"`
	Contact  string `json:"contact" validate:"omitempty,min=8,max=20"`
}

// UpdateUserRequest adalah isi body PUT /api/users/:id.
// Semua field opsional; field yang kosong tidak diubah.
type UpdateUserRequest struct {
	FullName string `json:"fullName" validate:"omitempty,min=3,max=255"`
	Email    string `json:"email" validate:"omitempty,email,max=255"`
	Contact  string `json:"contact" validate:"omitempty,min=8,max=20"`
}

// UserListQuery adalah query string GET /api/users?page=1&limit=10&search=budi&role=USER.
type UserListQuery struct {
	Page   int
	Limit  int
	Search string
	Role   string
}

// UserResponse adalah bentuk data user yang dikirim ke FE (tanpa password).
type UserResponse struct {
	ID        uint      `json:"id"`
	FullName  string    `json:"fullName"`
	NIM       string    `json:"nim"`
	Email     string    `json:"email"`
	Contact   string    `json:"contact"`
	Role      string    `json:"role"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// NewUserResponse mengubah entity.User menjadi UserResponse.
func NewUserResponse(u *entity.User) UserResponse {
	return UserResponse{
		ID:        u.ID,
		FullName:  u.FullName,
		NIM:       u.NIM,
		Email:     u.Email,
		Contact:   u.Contact,
		Role:      u.Role,
		CreatedAt: u.CreatedAt,
		UpdatedAt: u.UpdatedAt,
	}
}

// UserSummary adalah ringkasan user yang ikut ditampilkan di data lain, mis. peminjam sebuah
// pengajuan atau pelaku sebuah aktivitas di log.
type UserSummary struct {
	ID       uint   `json:"id"`
	FullName string `json:"fullName"`
}

// NewUserSummary mengubah entity.User menjadi UserSummary.
// Mengembalikan nil (null di JSON) jika user belum di-Preload.
func NewUserSummary(u *entity.User) *UserSummary {
	if u == nil {
		return nil
	}
	return &UserSummary{ID: u.ID, FullName: u.FullName}
}
