package entity

import (
	"time"

	"SGEInventory_BE/domain/enums"
)

// Consignment adalah cermin tabel consignments: satu pengajuan penitipan barang dari user.
// Barang-barang yang dititipkan disimpan di tabel items (type CONSIGNMENT, items.consignment_id = id ini).
// Seperti Borrowing, tidak memakai soft delete karena riwayat pengajuan harus tetap utuh; waktu tiap tahap
// disimpan di log_users (request_type CONSIGNMENT).
type Consignment struct {
	ID uint `gorm:"primaryKey"`
	// penitip (pemilik pengajuan), diambil dari token login
	UserID uint  `gorm:"not null;index:idx_consignments_user_status,priority:1"`
	User   *User `gorm:"constraint:OnUpdate:CASCADE,OnDelete:RESTRICT"`
	// lembaga asal penitip, dipilih di form
	LembagaID uint     `gorm:"not null;index:idx_consignments_lembaga_id"`
	Lembaga   *Lembaga `gorm:"constraint:OnUpdate:CASCADE,OnDelete:RESTRICT"`
	EventName string   `gorm:"size:255;not null"`
	Status    string   `gorm:"size:20;not null;default:PENDING;index:idx_consignments_user_status,priority:2;index:idx_consignments_status_created,priority:1;check:chk_consignments_status,status IN ('PENDING','APPROVED','REJECTED')"`
	// type:date = hanya tanggal, tanpa jam
	StartDate time.Time `gorm:"type:date;not null"`
	EndDate   time.Time `gorm:"type:date;not null;check:chk_consignments_end_after_start,end_date > start_date"`
	// Photo: key foto barang di FileStorage (Cloudinary / folder lokal), mis. "consignments/8f3a...e1.jpg".
	// Yang disimpan key, bukan URL penuh; URL publik dibuat storage saat membalas (response: photoUrl).
	Photo string `gorm:"size:255;not null"`
	// Notes: catatan dari user, boleh kosong.
	Notes string `gorm:"type:text"`
	// Items: barang titipan (has-many lewat items.consignment_id). Ikut tersimpan otomatis saat Create.
	// Tag constraint di sini menjadi aturan FK fk_consignments_items: penitipan yang masih punya barang
	// tidak bisa dihapus permanen (RESTRICT).
	Items     []Item    `gorm:"foreignKey:ConsignmentID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT"`
	CreatedAt time.Time `gorm:"index:idx_consignments_status_created,priority:2"`
	UpdatedAt time.Time
}

func (Consignment) TableName() string {
	return "consignments"
}

// NewConsignment membuat pengajuan penitipan baru (status PENDING) beserta barang-barangnya.
// items dibuat dengan NewConsignmentItem; consignment_id-nya diisi GORM saat Create.
func NewConsignment(userID, lembagaID uint, eventName string, startDate, endDate time.Time, photo, notes string, items []Item) *Consignment {
	return &Consignment{
		UserID:    userID,
		LembagaID: lembagaID,
		EventName: eventName,
		Status:    enums.ConsignmentPending,
		StartDate: startDate,
		EndDate:   endDate,
		Photo:     photo,
		Notes:     notes,
		Items:     items,
	}
}
