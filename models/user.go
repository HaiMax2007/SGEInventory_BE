package models

import (
	"time"

	"gorm.io/gorm"
)

const (
	RoleAdmin = "ADMIN"
	RoleUser  = "USER"
)

// User adalah cermin tabel users. Tag json:"-" membuat field itu
// tidak pernah ikut terkirim di respons API.
type User struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	FullName  string         `gorm:"size:255;not null" json:"fullName"`
	Email     string         `gorm:"size:255;not null;uniqueIndex" json:"email"`
	Password  string         `gorm:"column:password_hash;size:255;not null" json:"-"`
	Contact   string         `gorm:"size:50" json:"contact"`
	Role      string         `gorm:"size:10;not null;default:USER" json:"role"`
	CreatedAt time.Time      `json:"createdAt"`
	UpdatedAt time.Time      `json:"updatedAt"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

func (User) TableName() string {
	return "users"
}

// RegisterUserRequest adalah isi body POST /api/users.
// Tidak ada field role: user baru selalu USER.
type RegisterUserRequest struct {
	FullName string `json:"fullName" validate:"required,min=3,max=255"`
	Email    string `json:"email" validate:"required,email,max=255"`
	Password string `json:"password" validate:"required,min=8"`
	Contact  string `json:"contact" validate:"omitempty,min=8,max=20"`
}

// UpdateUserRequest adalah isi body PUT /api/users/:id.
// Semua field opsional; field yang kosong tidak diubah.
type UpdateUserRequest struct {
	FullName string `json:"fullName" validate:"omitempty,min=3,max=255"`
	Email    string `json:"email" validate:"omitempty,email,max=255"`
	Password string `json:"password" validate:"omitempty,min=8"`
	Contact  string `json:"contact" validate:"omitempty,min=8,max=20"`
}
