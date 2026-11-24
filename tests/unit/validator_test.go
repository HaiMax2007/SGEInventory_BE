package unit

import (
	"strings"
	"testing"

	"SGEInventory_BE/domain/dto"
	"SGEInventory_BE/pkg/validator"
)

// Tes ini memastikan tag `validate:"..."` di DTO menolak/menerima data seperti yang diharapkan.

func ptr[T any](v T) *T { return &v }

func TestValidateDTO(t *testing.T) {
	t.Parallel()
	validUser := dto.RegisterUserRequest{
		FullName: "Budi Santoso", NIM: "2210511001", Email: "budi@mail.com", Password: "rahasia123",
	}

	tests := []struct {
		name    string
		input   any
		wantErr bool
	}{
		// registrasi user
		{name: "register valid", input: validUser},
		{name: "register NIM huruf", input: withUser(validUser, func(u *dto.RegisterUserRequest) { u.NIM = "22105A1001" }), wantErr: true},
		{name: "register NIM 7 digit", input: withUser(validUser, func(u *dto.RegisterUserRequest) { u.NIM = "1234567" }), wantErr: true},
		{name: "register NIM 21 digit", input: withUser(validUser, func(u *dto.RegisterUserRequest) { u.NIM = strings.Repeat("1", 21) }), wantErr: true},
		{name: "register email salah", input: withUser(validUser, func(u *dto.RegisterUserRequest) { u.Email = "budi" }), wantErr: true},
		{name: "register password 7 karakter", input: withUser(validUser, func(u *dto.RegisterUserRequest) { u.Password = "1234567" }), wantErr: true},
		{name: "register nama 2 karakter", input: withUser(validUser, func(u *dto.RegisterUserRequest) { u.FullName = "Bu" }), wantErr: true},
		{name: "register kontak kosong boleh", input: withUser(validUser, func(u *dto.RegisterUserRequest) { u.Contact = "" })},

		// login
		{name: "login valid", input: dto.LoginRequest{Email: "budi@mail.com", Password: "x"}},
		{name: "login tanpa password", input: dto.LoginRequest{Email: "budi@mail.com"}, wantErr: true},

		// lembaga
		{name: "lembaga valid", input: dto.CreateLembagaRequest{Name: "BEM"}},
		{name: "lembaga nama 1 huruf", input: dto.CreateLembagaRequest{Name: "B"}, wantErr: true},
		{name: "ubah lembaga tanpa nama", input: dto.UpdateLembagaRequest{}, wantErr: true},

		// barang
		{name: "barang valid", input: dto.CreateItemRequest{Name: "Proyektor", LembagaID: 1, TotalQuantity: 5}},
		{name: "barang tanpa lembaga", input: dto.CreateItemRequest{Name: "Proyektor", TotalQuantity: 5}, wantErr: true},
		{name: "barang keterangan 1001 karakter", input: dto.CreateItemRequest{Name: "Proyektor", LembagaID: 1, TotalQuantity: 5, Description: strings.Repeat("a", 1001)}, wantErr: true},
		{name: "ubah barang kosong boleh (dicek service)", input: dto.UpdateItemRequest{}},
		{name: "ubah total jadi 0 boleh", input: dto.UpdateItemRequest{TotalQuantity: ptr(0)}},
		{name: "ubah total negatif", input: dto.UpdateItemRequest{TotalQuantity: ptr(-1)}, wantErr: true},
		{name: "ubah lembagaId 0", input: dto.UpdateItemRequest{LembagaID: ptr(uint(0))}, wantErr: true},
		{name: "ubah keterangan jadi kosong boleh", input: dto.UpdateItemRequest{Description: ptr("")}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := validator.ValidateStruct(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateStruct(%+v) error = %v; wantErr %v", tt.input, err, tt.wantErr)
			}
		})
	}
}

func withUser(u dto.RegisterUserRequest, change func(*dto.RegisterUserRequest)) dto.RegisterUserRequest {
	change(&u)
	return u
}
