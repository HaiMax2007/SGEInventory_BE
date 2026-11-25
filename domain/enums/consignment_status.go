package enums

// Status penitipan. Disimpan di kolom consignments.status (VARCHAR + CHECK chk_consignments_status).
//
//	PENDING ─ACC─→ APPROVED (final)
//	   └─tolak─→ REJECTED (final; barang titipan & foto dihapus)
//
// Alur penitipan selesai di sini: tidak ada form lanjutan, tenggat (end_date) cukup ditampilkan/diatur FE.
const (
	ConsignmentPending  = "PENDING"
	ConsignmentApproved = "APPROVED"
	ConsignmentRejected = "REJECTED"
)

// ConsignmentStatuses berisi semua status penitipan yang sah.
var ConsignmentStatuses = []string{ConsignmentPending, ConsignmentApproved, ConsignmentRejected}
