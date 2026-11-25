package errx

import "fmt"

var (
	ErrConsignmentNotFound = New(KindNotFound, "CONSIGNMENT_NOT_FOUND", "Pengajuan penitipan tidak ditemukan")

	// filter ?status= di daftar penitipan (admin)
	ErrInvalidConsignmentStatus = New(KindInvalid, "INVALID_CONSIGNMENT_STATUS",
		"Status harus salah satu dari PENDING, APPROVED, REJECTED")
)

// Error validasi form penitipan per field (dari tag `validate` di dto.CreateConsignmentRequest).
// Satu field = satu error; pesannya langsung menjelaskan aturan field tersebut.
var (
	ErrInvalidConsignmentLembaga = New(KindInvalid, "INVALID_CONSIGNMENT_LEMBAGA", "Lembaga asal wajib dipilih")
	ErrInvalidEventName          = New(KindInvalid, "INVALID_EVENT_NAME", "Nama acara wajib diisi, 3-255 karakter")
	ErrInvalidConsignmentItems   = New(KindInvalid, "INVALID_CONSIGNMENT_ITEMS",
		"Isi 1-20 barang titipan, setiap barang wajib punya nama 2-255 karakter dan jumlah 1-1000000")
	ErrInvalidStartDate        = New(KindInvalid, "INVALID_START_DATE", "Tanggal mulai wajib diisi dengan format YYYY-MM-DD")
	ErrInvalidEndDate          = New(KindInvalid, "INVALID_END_DATE", "Tanggal selesai wajib diisi dengan format YYYY-MM-DD")
	ErrInvalidConsignmentPhoto = New(KindInvalid, "INVALID_CONSIGNMENT_PHOTO",
		"Foto barang titipan wajib diunggah: JPG, PNG, atau WebP, maksimal 5 MB")
	ErrInvalidConsignmentNotes = New(KindInvalid, "INVALID_CONSIGNMENT_NOTES", "Catatan maksimal 1000 karakter")
)

// Error aturan yang membandingkan beberapa field sekaligus, dicek di consignment service.
var (
	ErrStartDatePast    = New(KindInvalid, "START_DATE_PAST", "Tanggal mulai tidak boleh sebelum hari ini")
	ErrEndNotAfterStart = New(KindInvalid, "END_NOT_AFTER_START", "Tanggal selesai harus setelah tanggal mulai")
)

// NewDuplicateConsignmentItem: nama barang yang sama ditulis dua kali. position & firstPosition mulai dari 1.
func NewDuplicateConsignmentItem(position, firstPosition int) *DomainError {
	return New(KindInvalid, "DUPLICATE_CONSIGNMENT_ITEM",
		fmt.Sprintf("Barang ke-%d sama dengan barang ke-%d, gabungkan jumlahnya", position, firstPosition))
}

// NewConsignmentTooLong: tanggal selesai lebih dari maxDays hari setelah tanggal mulai.
func NewConsignmentTooLong(maxDays int) *DomainError {
	return New(KindInvalid, "CONSIGNMENT_TOO_LONG",
		fmt.Sprintf("Lama penitipan maksimal %d hari dari tanggal mulai", maxDays))
}
