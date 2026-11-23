package errx

var (
	// Pesan sengaja sama untuk "email tidak ada" dan "password salah",
	// supaya orang lain tidak bisa menebak email mana yang terdaftar.
	ErrInvalidCredentials = New(KindUnauthorized, "INVALID_CREDENTIALS", "Email atau password salah")
	ErrTokenMissing       = New(KindUnauthorized, "TOKEN_MISSING", "Token tidak ada")
	ErrTokenInvalid       = New(KindUnauthorized, "TOKEN_INVALID", "Token tidak valid atau sudah kedaluwarsa")
	ErrAdminOnly          = New(KindForbidden, "ADMIN_ONLY", "Hanya admin yang boleh mengakses")
	ErrUserOnly           = New(KindForbidden, "USER_ONLY", "Hanya user (bukan admin) yang boleh mengakses")
)
