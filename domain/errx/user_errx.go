package errx

var (
	ErrUserNotFound       = New(KindNotFound, "USER_NOT_FOUND", "User tidak ditemukan")
	ErrEmailTaken         = New(KindConflict, "EMAIL_TAKEN", "Email sudah terdaftar")
	ErrNIMTaken           = New(KindConflict, "NIM_TAKEN", "NIM sudah terdaftar")
	ErrForbiddenOtherUser = New(KindForbidden, "FORBIDDEN_OTHER_USER", "Kamu tidak boleh mengakses data user lain")
)

// Error validasi form user (register, ubah profil) dan login.
// Satu field = satu error; pesannya langsung menjelaskan aturan field tersebut.
var (
	ErrInvalidFullName = New(KindInvalid, "INVALID_FULL_NAME", "Nama lengkap wajib diisi, 3-255 karakter")
	ErrInvalidNIM      = New(KindInvalid, "INVALID_NIM", "NIM wajib diisi, berupa angka 8-20 digit")
	ErrInvalidEmail    = New(KindInvalid, "INVALID_EMAIL", "Email wajib diisi dengan format yang benar, mis. nama@mail.com")
	ErrInvalidPassword = New(KindInvalid, "INVALID_PASSWORD", "Password wajib diisi, 8-72 karakter")
	ErrInvalidContact  = New(KindInvalid, "INVALID_CONTACT", "Kontak harus 8-20 karakter")
)
