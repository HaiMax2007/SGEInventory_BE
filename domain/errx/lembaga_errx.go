package errx

var (
	ErrLembagaNotFound  = New(KindNotFound, "LEMBAGA_NOT_FOUND", "Lembaga tidak ditemukan")
	ErrLembagaNameTaken = New(KindConflict, "LEMBAGA_NAME_TAKEN", "Nama lembaga sudah dipakai")
	ErrLembagaInUse     = New(KindConflict, "LEMBAGA_IN_USE", "Lembaga masih menjadi asal barang, tidak bisa dihapus")

	// validasi form lembaga
	ErrInvalidLembagaName = New(KindInvalid, "INVALID_LEMBAGA_NAME", "Nama lembaga wajib diisi, 2-150 karakter")
)
