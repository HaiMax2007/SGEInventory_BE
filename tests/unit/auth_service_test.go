package unit

import (
	"testing"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"

	"SGEInventory_BE/domain/contracts"
	"SGEInventory_BE/domain/dto"
	"SGEInventory_BE/domain/entity"
	"SGEInventory_BE/domain/enums"
	"SGEInventory_BE/domain/errx"
	"SGEInventory_BE/internal/app/auth"
)

// newAuthService menyiapkan AuthService + satu user ADMIN dengan password "rahasia123".
func newAuthService(t *testing.T) (contracts.AuthService, *entity.User) {
	t.Helper()
	db := newFakeDB()
	u := db.addUser("Max", enums.RoleAdmin) // email max@example.com
	hash, err := bcrypt.GenerateFromPassword([]byte("rahasia123"), bcrypt.MinCost)
	mustNoErr(t, err)
	db.users[u.ID].PasswordHash = string(hash)
	return auth.NewAuthService(&fakeUserRepo{db}, testSecret), u
}

func TestLogin_Success_ReturnsValidToken(t *testing.T) {
	t.Parallel()
	svc, u := newAuthService(t)

	res, err := svc.Login(t.Context(), dto.LoginRequest{Email: "MAX@example.com", Password: "rahasia123"})
	mustNoErr(t, err)
	if res.TokenType != "Bearer" || res.ExpiresIn != 24*60*60 {
		t.Errorf("tokenType=%q expiresIn=%d; want Bearer dan 86400", res.TokenType, res.ExpiresIn)
	}

	// Token harus bisa dibaca dengan secret yang sama dan berisi id + role user.
	claims := &dto.JWTClaims{}
	_, err = jwt.ParseWithClaims(res.Token, claims, func(*jwt.Token) (any, error) { return []byte(testSecret), nil },
		jwt.WithValidMethods([]string{"HS256"}), jwt.WithExpirationRequired())
	mustNoErr(t, err)
	if claims.UserID != u.ID || claims.Role != enums.RoleAdmin {
		t.Errorf("isi token userID=%d role=%q; want %d ADMIN", claims.UserID, claims.Role, u.ID)
	}
}

func TestLogin_WrongCredentials_SameError(t *testing.T) {
	t.Parallel()
	svc, _ := newAuthService(t)

	// Email tidak terdaftar dan password salah harus menghasilkan error yang SAMA,
	// supaya orang lain tidak bisa menebak email mana yang terdaftar.
	for _, req := range []dto.LoginRequest{
		{Email: "max@example.com", Password: "salah"},
		{Email: "tidakada@example.com", Password: "rahasia123"},
	} {
		_, err := svc.Login(t.Context(), req)
		assertDomainError(t, err, "INVALID_CREDENTIALS", errx.KindUnauthorized)
	}
}

func TestMe(t *testing.T) {
	t.Parallel()
	svc, u := newAuthService(t)

	res, err := svc.Me(t.Context(), u.ID)
	mustNoErr(t, err)
	if res.ID != u.ID {
		t.Errorf("Me().ID = %d; want %d", res.ID, u.ID)
	}

	_, err = svc.Me(t.Context(), 999)
	assertDomainError(t, err, "USER_NOT_FOUND", errx.KindNotFound)
}
