package errx

import "fmt"

var (
	ErrItemNotFound      = New(KindNotFound, "ITEM_NOT_FOUND", "Barang tidak ditemukan")
	ErrItemStillBorrowed = New(KindConflict, "ITEM_STILL_BORROWED", "Barang masih dipinjam, tidak bisa dihapus")
)

// NewTotalBelowBorrowed dipakai saat admin mengubah total menjadi lebih kecil dari
// jumlah yang sedang dipinjam. Pesannya berisi angka, jadi dibuat lewat fungsi.
func NewTotalBelowBorrowed(borrowed int) *DomainError {
	return New(KindConflict, "TOTAL_BELOW_BORROWED",
		fmt.Sprintf("Total tidak boleh kurang dari jumlah yang sedang dipinjam (%d)", borrowed))
}
