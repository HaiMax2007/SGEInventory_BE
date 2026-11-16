package errx

var (
	ErrInvalidID       = New(KindInvalid, "INVALID_ID", "ID tidak valid")
	ErrNothingToUpdate = New(KindInvalid, "NOTHING_TO_UPDATE", "Tidak ada data yang diperbarui")
)
