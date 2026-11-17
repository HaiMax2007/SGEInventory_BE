package errx

var (
	ErrUserNotFound       = New(KindNotFound, "USER_NOT_FOUND", "User tidak ditemukan")
	ErrEmailTaken         = New(KindConflict, "EMAIL_TAKEN", "Email sudah terdaftar")
	ErrNIMTaken           = New(KindConflict, "NIM_TAKEN", "NIM sudah terdaftar")
	ErrPasswordTooLong    = New(KindInvalid, "PASSWORD_TOO_LONG", "Password terlalu panjang (maksimal 72 karakter)")
	ErrForbiddenOtherUser = New(KindForbidden, "FORBIDDEN_OTHER_USER", "Kamu tidak boleh mengakses data user lain")
)
