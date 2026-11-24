package unit

import (
	"strings"
	"testing"

	"golang.org/x/crypto/bcrypt"

	"SGEInventory_BE/domain/contracts"
	"SGEInventory_BE/domain/dto"
	"SGEInventory_BE/domain/enums"
	"SGEInventory_BE/domain/errx"
	"SGEInventory_BE/internal/app/user"
)

func newUserService() (*fakeDB, contracts.UserService) {
	db := newFakeDB()
	return db, user.NewUserService(&fakeUserRepo{db})
}

func TestRegister_Success(t *testing.T) {
	t.Parallel()
	db, svc := newUserService()

	res, err := svc.Register(t.Context(), dto.RegisterUserRequest{
		FullName: "Budi", NIM: "2210511001", Email: "Budi@Mail.com", Password: "rahasia123",
	})
	mustNoErr(t, err)

	if res.Role != enums.RoleUser {
		t.Errorf("role = %q; want USER (registrasi tidak bisa memilih role)", res.Role)
	}
	if res.Email != "budi@mail.com" {
		t.Errorf("email = %q; want huruf kecil", res.Email)
	}
	stored := db.users[res.ID]
	if stored.PasswordHash == "rahasia123" {
		t.Fatal("password disimpan apa adanya; want di-hash")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(stored.PasswordHash), []byte("rahasia123")); err != nil {
		t.Errorf("hash tidak cocok dengan password asli: %v", err)
	}
}

func TestRegister_Errors(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		req      dto.RegisterUserRequest
		wantCode string
		wantKind errx.Kind
	}{
		{
			name:     "password lebih dari 72 byte",
			req:      dto.RegisterUserRequest{FullName: "Budi", NIM: "2210511001", Email: "budi@mail.com", Password: strings.Repeat("a", 73)},
			wantCode: "PASSWORD_TOO_LONG", wantKind: errx.KindInvalid,
		},
		{
			name:     "email sudah dipakai (beda huruf besar/kecil)",
			req:      dto.RegisterUserRequest{FullName: "Budi", NIM: "2210511001", Email: "RAKA@example.com", Password: "rahasia123"},
			wantCode: "EMAIL_TAKEN", wantKind: errx.KindConflict,
		},
		{
			name:     "NIM sudah dipakai",
			req:      dto.RegisterUserRequest{FullName: "Budi", NIM: "00000001", Email: "budi@mail.com", Password: "rahasia123"},
			wantCode: "NIM_TAKEN", wantKind: errx.KindConflict,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			db, svc := newUserService()
			db.addUser("Raka", enums.RoleUser) // email raka@example.com, NIM 00000001

			_, err := svc.Register(t.Context(), tt.req)
			assertDomainError(t, err, tt.wantCode, tt.wantKind)
			if len(db.users) != 1 {
				t.Errorf("jumlah user = %d; want tetap 1", len(db.users))
			}
		})
	}
}

func TestUpdateUser(t *testing.T) {
	t.Parallel()

	t.Run("body kosong ditolak", func(t *testing.T) {
		t.Parallel()
		db, svc := newUserService()
		u := db.addUser("Raka", enums.RoleUser)
		_, err := svc.UpdateUser(t.Context(), u.ID, dto.UpdateUserRequest{})
		assertDomainError(t, err, "NOTHING_TO_UPDATE", errx.KindInvalid)
	})

	t.Run("hanya field yang diisi yang berubah", func(t *testing.T) {
		t.Parallel()
		db, svc := newUserService()
		u := db.addUser("Raka", enums.RoleUser)
		res, err := svc.UpdateUser(t.Context(), u.ID, dto.UpdateUserRequest{Email: "Raka.Baru@Mail.com"})
		mustNoErr(t, err)
		if res.Email != "raka.baru@mail.com" || res.FullName != "Raka" {
			t.Errorf("hasil update: nama %q email %q; want nama tetap Raka, email huruf kecil", res.FullName, res.Email)
		}
	})

	t.Run("email milik user lain ditolak", func(t *testing.T) {
		t.Parallel()
		db, svc := newUserService()
		u := db.addUser("Raka", enums.RoleUser)
		db.addUser("Max", enums.RoleAdmin)
		_, err := svc.UpdateUser(t.Context(), u.ID, dto.UpdateUserRequest{Email: "max@example.com"})
		assertDomainError(t, err, "EMAIL_TAKEN", errx.KindConflict)
	})

	t.Run("user tidak ada", func(t *testing.T) {
		t.Parallel()
		_, svc := newUserService()
		_, err := svc.UpdateUser(t.Context(), 99, dto.UpdateUserRequest{FullName: "X"})
		assertDomainError(t, err, "USER_NOT_FOUND", errx.KindNotFound)
	})
}

func TestGetAndDeleteUser(t *testing.T) {
	t.Parallel()
	db, svc := newUserService()
	u := db.addUser("Raka", enums.RoleUser)

	res, err := svc.GetUser(t.Context(), u.ID)
	mustNoErr(t, err)
	if res.FullName != "Raka" {
		t.Errorf("GetUser nama = %q; want Raka", res.FullName)
	}

	mustNoErr(t, svc.DeleteUser(t.Context(), u.ID))
	_, err = svc.GetUser(t.Context(), u.ID)
	assertDomainError(t, err, "USER_NOT_FOUND", errx.KindNotFound) // soft delete: tidak terlihat lagi
}

func TestGetUsers_FilterAndPagination(t *testing.T) {
	t.Parallel()
	db, svc := newUserService()
	db.addUser("Raka", enums.RoleUser)
	db.addUser("Budi", enums.RoleUser)
	db.addUser("Max", enums.RoleAdmin)

	res, page, err := svc.GetUsers(t.Context(), dto.UserListQuery{Page: 1, Limit: 1, Role: enums.RoleUser})
	mustNoErr(t, err)
	if len(res) != 1 || page.Total != 2 || page.TotalPages != 2 {
		t.Errorf("GetUsers(role USER, limit 1) = %d data, total %d, %d halaman; want 1, 2, 2", len(res), page.Total, page.TotalPages)
	}
}
