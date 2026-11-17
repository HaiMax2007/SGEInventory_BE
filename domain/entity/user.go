package entity

import (
	"strings"
	"time"

	"gorm.io/gorm"

	"SGEInventory_BE/domain/enums"
)

// User adalah cermin tabel users.
// Tag gorm memberi tahu GORM nama kolom/aturannya. Bentuk JSON ke FE diatur di dto.UserResponse,
// jadi password_hash tidak pernah ikut terkirim.
type User struct {
	ID           uint   `gorm:"primaryKey"`
	FullName     string `gorm:"size:255;not null"`
	NIM          string `gorm:"column:nim;size:20;not null;uniqueIndex"`
	Email        string `gorm:"size:255;not null;uniqueIndex"`
	PasswordHash string `gorm:"column:password_hash;size:255;not null"`
	Contact      string `gorm:"size:50"`
	Role         string `gorm:"size:10;not null;default:USER"`
	CreatedAt    time.Time
	UpdatedAt    time.Time
	DeletedAt    gorm.DeletedAt `gorm:"index"` // soft delete
}

func (User) TableName() string {
	return "users"
}

// NewUser membuat user baru. Role selalu USER dan email selalu huruf kecil.
func NewUser(fullName, nim, email, passwordHash, contact string) *User {
	return &User{
		FullName:     fullName,
		NIM:          nim,
		Email:        strings.ToLower(email),
		PasswordHash: passwordHash,
		Contact:      contact,
		Role:         enums.RoleUser,
	}
}

// ApplyUpdate mengubah data user. Field yang kosong ("") tidak diubah.
func (u *User) ApplyUpdate(fullName, email, contact string) {
	if fullName != "" {
		u.FullName = fullName
	}
	if email != "" {
		u.Email = strings.ToLower(email)
	}
	if contact != "" {
		u.Contact = contact
	}
}

func (u *User) IsAdmin() bool {
	return u.Role == enums.RoleAdmin
}
