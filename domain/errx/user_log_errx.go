package errx

// Error untuk riwayat aktivitas (log_users).
var (
	ErrInvalidRequestType = New(KindInvalid, "INVALID_REQUEST_TYPE", "Jenis pengajuan harus BORROWING atau CONSIGNMENT")
)
