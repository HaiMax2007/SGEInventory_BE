package errx

import (
	"fmt"
	"strings"
)

var (
	ErrBorrowingNotFound = New(KindNotFound, "BORROWING_NOT_FOUND", "Pengajuan peminjaman tidak ditemukan")

	// filter ?status= di daftar pengajuan (admin)
	ErrInvalidBorrowingStatus = New(KindInvalid, "INVALID_BORROWING_STATUS",
		"Status harus salah satu dari PENDING, APPROVED, REJECTED, RETURN_PENDING, RETURNED")
)

// Error validasi form peminjaman per field (dari tag `validate` di dto.CreateBorrowingRequest).
// Satu field = satu error; pesannya langsung menjelaskan aturan field tersebut.
var (
	ErrInvalidBorrowItems = New(KindInvalid, "INVALID_BORROW_ITEMS",
		"Pilih 1-20 barang, setiap barang wajib dipilih dengan jumlah lebih dari 0")
	ErrInvalidBorrowLembaga = New(KindInvalid, "INVALID_BORROW_LEMBAGA", "Lembaga wajib dipilih")
	ErrInvalidPurpose       = New(KindInvalid, "INVALID_PURPOSE", "Tujuan peminjaman wajib diisi, maksimal 500 karakter")
	ErrInvalidPickupDate    = New(KindInvalid, "INVALID_PICKUP_DATE", "Tanggal pengambilan wajib diisi dengan format YYYY-MM-DD")
	ErrInvalidReturnDate    = New(KindInvalid, "INVALID_RETURN_DATE", "Tanggal pengembalian wajib diisi dengan format YYYY-MM-DD")
)

// Error aturan yang membandingkan beberapa field sekaligus (tidak bisa ditulis sebagai tag),
// dicek di borrowing service.
var (
	ErrPickupDatePast       = New(KindInvalid, "PICKUP_DATE_PAST", "Tanggal pengambilan tidak boleh sebelum hari ini")
	ErrReturnNotAfterPickup = New(KindInvalid, "RETURN_NOT_AFTER_PICKUP", "Tanggal pengembalian harus setelah tanggal pengambilan")
)

// Error berikut pesannya berisi angka/nama, jadi dibuat lewat fungsi.

// NewDuplicateBorrowItem: barang yang sama dipilih dua kali. position & firstPosition mulai dari 1.
func NewDuplicateBorrowItem(position, firstPosition int) *DomainError {
	return New(KindInvalid, "DUPLICATE_BORROW_ITEM",
		fmt.Sprintf("Barang ke-%d sama dengan barang ke-%d, gabungkan jumlahnya", position, firstPosition))
}

// NewBorrowTooLong: tanggal pengembalian lebih dari maxDays hari setelah tanggal pengambilan.
func NewBorrowTooLong(maxDays int) *DomainError {
	return New(KindInvalid, "BORROW_TOO_LONG",
		fmt.Sprintf("Lama peminjaman maksimal %d hari dari tanggal pengambilan", maxDays))
}

// NewBorrowItemsNotFound: ada barang yang tidak ada atau sudah dihapus.
func NewBorrowItemsNotFound(ids []uint) *DomainError {
	parts := make([]string, len(ids))
	for i, id := range ids {
		parts[i] = fmt.Sprintf("%d", id)
	}
	return New(KindNotFound, "BORROW_ITEMS_NOT_FOUND",
		"Barang tidak ditemukan (id: "+strings.Join(parts, ", ")+")")
}

// NewInsufficientStock: jumlah yang diminta melebihi stok tersedia. details berisi semua barang
// yang bermasalah, mis. "Proyektor (diminta 5, tersedia 3)", supaya user bisa memperbaiki sekaligus.
func NewInsufficientStock(details []string) *DomainError {
	return New(KindConflict, "INSUFFICIENT_STOCK",
		"Stok tidak cukup: "+strings.Join(details, ", "))
}

// NewBorrowingAlreadyProcessed: admin mencoba ACC/menolak pengajuan yang statusnya bukan PENDING lagi
// (sudah diproses, mungkin oleh admin lain atau ditolak otomatis). Setiap tahap hanya terjadi sekali.
func NewBorrowingAlreadyProcessed(status string) *DomainError {
	return New(KindConflict, "BORROWING_ALREADY_PROCESSED",
		fmt.Sprintf("Pengajuan sudah diproses (status: %s)", status))
}

// NewBorrowItemsUnavailable: saat ACC, ada barang di pengajuan yang sudah dihapus admin.
// Pengajuan seperti ini tidak bisa di-ACC; admin cukup menolaknya.
func NewBorrowItemsUnavailable(names []string) *DomainError {
	return New(KindConflict, "BORROW_ITEMS_UNAVAILABLE",
		"Barang sudah dihapus: "+strings.Join(names, ", "))
}
