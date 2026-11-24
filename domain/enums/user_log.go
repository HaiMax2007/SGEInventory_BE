package enums

// Jenis pengajuan yang dicatat di log_users (kolom request_type).
// Satu tabel log dipakai untuk dua form: peminjaman dan penitipan.
const (
	RequestTypeBorrowing   = "BORROWING"
	RequestTypeConsignment = "CONSIGNMENT"
)

// Aktivitas yang dicatat di log_users (kolom request_activity).
// Dibuat umum supaya bisa dipakai peminjaman maupun penitipan.
const (
	ActivitySubmitted       = "SUBMITTED"        // user mengirim pengajuan
	ActivityApproved        = "APPROVED"         // admin menyetujui pengajuan
	ActivityRejected        = "REJECTED"         // admin menolak pengajuan
	ActivityAutoRejected    = "AUTO_REJECTED"    // ditolak otomatis: stok tidak cukup lagi (pelaku = admin pemicunya)
	ActivityReturnSubmitted = "RETURN_SUBMITTED" // user mengajukan pengembalian
	ActivityReturnApproved  = "RETURN_APPROVED"  // admin menyetujui pengembalian
	ActivityReturnRejected  = "RETURN_REJECTED"  // admin menolak pengembalian
)

// RequestTypes berisi semua jenis pengajuan yang sah, dipakai untuk mengecek filter ?requestType=.
var RequestTypes = []string{RequestTypeBorrowing, RequestTypeConsignment}
