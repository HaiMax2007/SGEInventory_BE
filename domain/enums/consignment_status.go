package enums

// Status penitipan. Disimpan di kolom consignments.status (VARCHAR + CHECK chk_consignments_status).
// Tahap 1 hanya membuat PENDING; ACC/tolak admin menyusul di tahap berikutnya.
const (
	ConsignmentPending  = "PENDING"
	ConsignmentApproved = "APPROVED"
	ConsignmentRejected = "REJECTED"
)

// ConsignmentStatuses berisi semua status penitipan yang sah.
var ConsignmentStatuses = []string{ConsignmentPending, ConsignmentApproved, ConsignmentRejected}
