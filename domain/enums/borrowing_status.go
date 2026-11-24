package enums

// Status peminjaman. Disimpan di kolom borrowings.status (VARCHAR + CHECK chk_borrowings_status).
//
//	PENDING ─ACC─→ APPROVED ─user unggah foto─→ RETURN_PENDING ─ACC─→ RETURNED (stok kembali, selesai)
//	   └─REJECT─→ REJECTED            ▲                        │
//	              (final)             └──── RETURN_REJECTED ◄───┘ admin tolak foto (foto lama dihapus,
//	                                        user unggah foto baru)
const (
	BorrowingPending        = "PENDING"
	BorrowingApproved       = "APPROVED"
	BorrowingRejected       = "REJECTED"
	BorrowingReturnPending  = "RETURN_PENDING"
	BorrowingReturnRejected = "RETURN_REJECTED"
	BorrowingReturned       = "RETURNED"
)

// BorrowingStatuses berisi semua status yang sah, dipakai untuk mengecek filter ?status= dari admin.
var BorrowingStatuses = []string{
	BorrowingPending, BorrowingApproved, BorrowingRejected,
	BorrowingReturnPending, BorrowingReturnRejected, BorrowingReturned,
}
