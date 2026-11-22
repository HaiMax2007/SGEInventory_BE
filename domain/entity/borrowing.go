package entity

import (
	"time"

	"SGEInventory_BE/domain/enums"
)

// Borrowing adalah cermin tabel borrowings: satu pengajuan peminjaman (header).
// Setelah dikirim, isi pengajuan tidak bisa diubah; yang berubah hanya status lewat alur admin.
// Tidak memakai soft delete karena riwayat peminjaman harus tetap utuh.
//
// Riwayat waktu tiap tahap (diajukan, di-ACC, dikembalikan, ...) tidak disimpan di sini,
// tapi di log_users (lihat UserLog).
type Borrowing struct {
	ID uint `gorm:"primaryKey"`
	// peminjam (pemilik pengajuan), diambil dari token login
	UserID uint  `gorm:"not null;index:idx_borrowings_user_status,priority:1"`
	User   *User `gorm:"constraint:OnUpdate:CASCADE,OnDelete:RESTRICT"`
	// lembaga yang meminjam, dipilih di form
	LembagaID uint     `gorm:"not null;index:idx_borrowings_lembaga_id"`
	Lembaga   *Lembaga `gorm:"constraint:OnUpdate:CASCADE,OnDelete:RESTRICT"`
	Purpose   string   `gorm:"type:text;not null"`
	// type:date = hanya tanggal, tanpa jam
	PickupDate time.Time `gorm:"type:date;not null"`
	ReturnDate time.Time `gorm:"type:date;not null;check:chk_borrowings_return_after_pickup,return_date > pickup_date"`
	Status     string    `gorm:"size:20;not null;default:PENDING;index:idx_borrowings_user_status,priority:2;index:idx_borrowings_status_created,priority:1;check:chk_borrowings_status,status IN ('PENDING','APPROVED','REJECTED','RETURN_PENDING','RETURNED')"`
	// Items: daftar barang di pengajuan ini (has-many). Ikut tersimpan otomatis saat Create.
	// Tag constraint di sini menjadi aturan FK borrowing_items.borrowing_id: header dihapus → detail ikut terhapus.
	Items     []BorrowingItem `gorm:"foreignKey:BorrowingID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
	CreatedAt time.Time       `gorm:"index:idx_borrowings_status_created,priority:2"`
	UpdatedAt time.Time
}

func (Borrowing) TableName() string {
	return "borrowings"
}

// NewBorrowing membuat pengajuan baru dengan status PENDING.
func NewBorrowing(userID, lembagaID uint, purpose string, pickupDate, returnDate time.Time, items []BorrowingItem) *Borrowing {
	return &Borrowing{
		UserID:     userID,
		LembagaID:  lembagaID,
		Purpose:    purpose,
		PickupDate: pickupDate,
		ReturnDate: returnDate,
		Status:     enums.BorrowingPending,
		Items:      items,
	}
}
