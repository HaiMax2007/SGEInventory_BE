package errx

import "fmt"

var (
	ErrItemNotFound      = New(KindNotFound, "ITEM_NOT_FOUND", "Barang tidak ditemukan")
	ErrItemStillBorrowed = New(KindConflict, "ITEM_STILL_BORROWED", "Barang masih dipinjam, tidak bisa dihapus")

	// Barang titipan user hanya diurus lewat alur penitipan, bukan lewat PUT/DELETE /api/items.
	ErrConsignmentItemReadOnly = New(KindConflict, "CONSIGNMENT_ITEM_READ_ONLY",
		"Barang titipan tidak bisa diubah atau dihapus di sini")

	// filter ?type= di daftar barang (admin)
	ErrInvalidItemType = New(KindInvalid, "INVALID_ITEM_TYPE", "Tipe barang harus BORROWABLE atau CONSIGNMENT")
)

// Error validasi form barang. Satu field = satu error; pesannya langsung menjelaskan aturannya.
var (
	ErrInvalidItemName        = New(KindInvalid, "INVALID_ITEM_NAME", "Nama barang wajib diisi, 2-255 karakter")
	ErrInvalidItemDescription = New(KindInvalid, "INVALID_ITEM_DESCRIPTION", "Keterangan barang maksimal 1000 karakter")
	ErrInvalidItemLembaga     = New(KindInvalid, "INVALID_ITEM_LEMBAGA", "Lembaga asal barang wajib dipilih")
	ErrInvalidItemTotal       = New(KindInvalid, "INVALID_ITEM_TOTAL", "Total barang harus 0 sampai 1000000")
)

// NewTotalBelowBorrowed dipakai saat admin mengubah total menjadi lebih kecil dari
// jumlah yang sedang dipinjam. Pesannya berisi angka, jadi dibuat lewat fungsi.
func NewTotalBelowBorrowed(borrowed int) *DomainError {
	return New(KindConflict, "TOTAL_BELOW_BORROWED",
		fmt.Sprintf("Total tidak boleh kurang dari jumlah yang sedang dipinjam (%d)", borrowed))
}
