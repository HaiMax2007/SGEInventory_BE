package errx

import (
	"fmt"
	"strings"
)

// Error untuk pengajuan peminjaman. Setiap aturan validasi punya error (dan pesan) sendiri,
// supaya FE bisa menampilkan pesan yang jelas untuk tiap kesalahan di form.
var (
	ErrBorrowingNotFound = New(KindNotFound, "BORROWING_NOT_FOUND", "Pengajuan peminjaman tidak ditemukan")

	// filter ?status= di daftar pengajuan (admin)
	ErrInvalidBorrowingStatus = New(KindInvalid, "INVALID_BORROWING_STATUS",
		"Status harus salah satu dari PENDING, APPROVED, REJECTED, RETURN_PENDING, RETURNED")

	// items
	ErrBorrowItemsRequired = New(KindInvalid, "BORROW_ITEMS_REQUIRED", "Pilih minimal 1 barang yang ingin dipinjam")

	// lembaga
	ErrBorrowLembagaRequired = New(KindInvalid, "BORROW_LEMBAGA_REQUIRED", "Lembaga wajib dipilih")

	// tujuan
	ErrPurposeRequired = New(KindInvalid, "PURPOSE_REQUIRED", "Tujuan peminjaman wajib diisi")

	// tanggal
	ErrPickupDateRequired   = New(KindInvalid, "PICKUP_DATE_REQUIRED", "Tanggal pengambilan wajib diisi")
	ErrPickupDateInvalid    = New(KindInvalid, "PICKUP_DATE_INVALID", "Format tanggal pengambilan harus YYYY-MM-DD")
	ErrReturnDateRequired   = New(KindInvalid, "RETURN_DATE_REQUIRED", "Tanggal pengembalian wajib diisi")
	ErrReturnDateInvalid    = New(KindInvalid, "RETURN_DATE_INVALID", "Format tanggal pengembalian harus YYYY-MM-DD")
	ErrPickupDatePast       = New(KindInvalid, "PICKUP_DATE_PAST", "Tanggal pengambilan tidak boleh sebelum hari ini")
	ErrReturnNotAfterPickup = New(KindInvalid, "RETURN_NOT_AFTER_PICKUP", "Tanggal pengembalian harus setelah tanggal pengambilan")
)

// Error berikut pesannya berisi angka/nama, jadi dibuat lewat fungsi.

// NewBorrowTooManyItems: jumlah jenis barang dalam satu pengajuan melebihi batas.
func NewBorrowTooManyItems(max int) *DomainError {
	return New(KindInvalid, "BORROW_TOO_MANY_ITEMS",
		fmt.Sprintf("Maksimal %d jenis barang dalam satu pengajuan", max))
}

// NewBorrowItemIDRequired: barang ke-n di daftar belum dipilih (itemId kosong/0). position mulai dari 1.
func NewBorrowItemIDRequired(position int) *DomainError {
	return New(KindInvalid, "BORROW_ITEM_ID_REQUIRED",
		fmt.Sprintf("Barang ke-%d belum dipilih", position))
}

// NewBorrowQuantityInvalid: jumlah barang ke-n kosong, 0, atau negatif. position mulai dari 1.
func NewBorrowQuantityInvalid(position int) *DomainError {
	return New(KindInvalid, "BORROW_QUANTITY_INVALID",
		fmt.Sprintf("Jumlah barang ke-%d harus lebih dari 0", position))
}

// NewDuplicateBorrowItem: barang yang sama dipilih dua kali. position & firstPosition mulai dari 1.
func NewDuplicateBorrowItem(position, firstPosition int) *DomainError {
	return New(KindInvalid, "DUPLICATE_BORROW_ITEM",
		fmt.Sprintf("Barang ke-%d sama dengan barang ke-%d, gabungkan jumlahnya", position, firstPosition))
}

// NewPurposeTooLong: tujuan peminjaman terlalu panjang.
func NewPurposeTooLong(max int) *DomainError {
	return New(KindInvalid, "PURPOSE_TOO_LONG",
		fmt.Sprintf("Tujuan peminjaman maksimal %d karakter", max))
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
