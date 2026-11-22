package enums

// Status peminjaman. Disimpan di kolom borrowings.status (VARCHAR + CHECK chk_borrowings_status).
// Status hanya bergerak maju dan tiap tahap terjadi satu kali:
//
//	PENDING ──ACC──→ APPROVED ──user ajukan pengembalian──→ RETURN_PENDING ──ACC──→ RETURNED
//	   └──REJECT──→ REJECTED (final; user membuat pengajuan baru)
const (
	BorrowingPending       = "PENDING"
	BorrowingApproved      = "APPROVED"
	BorrowingRejected      = "REJECTED"
	BorrowingReturnPending = "RETURN_PENDING"
	BorrowingReturned      = "RETURNED"
)

// BorrowingStatuses berisi semua status yang sah, dipakai untuk mengecek filter ?status= dari admin.
var BorrowingStatuses = []string{
	BorrowingPending, BorrowingApproved, BorrowingRejected, BorrowingReturnPending, BorrowingReturned,
}
