package entity

import (
	"time"

	"SGEInventory_BE/domain/enums"
	"SGEInventory_BE/domain/errx"
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
	Status     string    `gorm:"size:20;not null;default:PENDING;index:idx_borrowings_user_status,priority:2;index:idx_borrowings_status_created,priority:1;check:chk_borrowings_status,status IN ('PENDING','APPROVED','REJECTED','RETURN_PENDING','RETURN_REJECTED','RETURNED')"`
	// ReturnPhoto: path foto bukti pengembalian di folder uploads, mis. "returns/8f3a...e1.jpg".
	// NULL = belum ada foto (belum mengembalikan, atau foto ditolak admin lalu dihapus).
	// Yang disimpan path relatif, bukan URL penuh, supaya tidak rusak jika alamat API berubah.
	ReturnPhoto *string `gorm:"size:255"`
	// Items: daftar barang di pengajuan ini (has-many). Ikut tersimpan otomatis saat Create.
	// Tag constraint di sini menjadi aturan FK borrowing_items.borrowing_id: header dihapus → detail ikut terhapus.
	Items     []BorrowingItem `gorm:"foreignKey:BorrowingID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
	CreatedAt time.Time       `gorm:"index:idx_borrowings_status_created,priority:2"`
	UpdatedAt time.Time
}

func (Borrowing) TableName() string {
	return "borrowings"
}

// Approve menyetujui pengajuan (tahap 2). Hanya pengajuan PENDING yang bisa diproses, dan hanya sekali.
// Pengurangan stok dilakukan service (butuh data barang), bukan di sini.
func (b *Borrowing) Approve() error {
	if b.Status != enums.BorrowingPending {
		return errx.NewBorrowingAlreadyProcessed(b.Status)
	}
	b.Status = enums.BorrowingApproved
	return nil
}

// Reject menolak pengajuan. Hanya pengajuan PENDING yang bisa diproses, dan hanya sekali.
func (b *Borrowing) Reject() error {
	if b.Status != enums.BorrowingPending {
		return errx.NewBorrowingAlreadyProcessed(b.Status)
	}
	b.Status = enums.BorrowingRejected
	return nil
}

// SubmitReturn dipanggil saat user mengunggah foto pengembalian. Hanya pengajuan yang sedang dipinjam
// (APPROVED) atau yang fotonya baru ditolak admin (RETURN_REJECTED) yang bisa dikembalikan.
func (b *Borrowing) SubmitReturn(photoPath string) error {
	if b.Status != enums.BorrowingApproved && b.Status != enums.BorrowingReturnRejected {
		return errx.NewBorrowingNotReturnable(b.Status)
	}
	b.Status = enums.BorrowingReturnPending
	b.ReturnPhoto = &photoPath
	return nil
}

// ApproveReturn menyetujui pengembalian (alur selesai). Foto tetap disimpan sebagai bukti.
// Penambahan stok dilakukan service (butuh data barang), bukan di sini.
func (b *Borrowing) ApproveReturn() error {
	if b.Status != enums.BorrowingReturnPending {
		return errx.NewReturnNotPending(b.Status)
	}
	b.Status = enums.BorrowingReturned
	return nil
}

// RejectReturn menolak foto pengembalian. Foto dilepas (file-nya dihapus service setelah tersimpan),
// lalu user harus mengunggah foto baru.
func (b *Borrowing) RejectReturn() error {
	if b.Status != enums.BorrowingReturnPending {
		return errx.NewReturnNotPending(b.Status)
	}
	b.Status = enums.BorrowingReturnRejected
	b.ReturnPhoto = nil
	return nil
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
