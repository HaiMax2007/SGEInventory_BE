package unit

import (
	"errors"
	"slices"
	"strings"
	"testing"

	"SGEInventory_BE/domain/dto"
	"SGEInventory_BE/domain/errx"
	"SGEInventory_BE/pkg/validator"
)

// Tes ini memastikan tag `validate:"..."` di DTO menolak/menerima data seperti yang diharapkan,
// dan field yang gagal diterjemahkan ke errx miliknya lewat FieldErrors() (mis. Email → ErrInvalidEmail).

func ptr[T any](v T) *T { return &v }

func TestValidateStruct_MapsFieldToError(t *testing.T) {
	t.Parallel()
	validUser := dto.RegisterUserRequest{
		FullName: "Budi Santoso", NIM: "2210511001", Email: "budi@mail.com", Password: "rahasia123",
	}
	validItem := dto.CreateItemRequest{Name: "Proyektor", LembagaID: 1, TotalQuantity: 5}
	validBorrow := dto.CreateBorrowingRequest{
		Items:     []dto.BorrowingItemRequest{{ItemID: 1, Quantity: 2}},
		LembagaID: 1, Purpose: "Seminar nasional", PickUpDate: "2026-12-01", ReturnDate: "2026-12-03",
	}

	tests := []struct {
		name    string
		input   any // selalu pointer, seperti di handler: validator.ValidateStruct(&req)
		wantErr *errx.DomainError
	}{
		// registrasi user
		{name: "register valid", input: &validUser},
		{name: "register nama kosong", input: withUser(validUser, func(u *dto.RegisterUserRequest) { u.FullName = "" }), wantErr: errx.ErrInvalidFullName},
		{name: "register nama hanya spasi", input: withUser(validUser, func(u *dto.RegisterUserRequest) { u.FullName = "   " }), wantErr: errx.ErrInvalidFullName},
		{name: "register nama 2 karakter + spasi", input: withUser(validUser, func(u *dto.RegisterUserRequest) { u.FullName = "  Bu  " }), wantErr: errx.ErrInvalidFullName},
		{name: "register NIM huruf", input: withUser(validUser, func(u *dto.RegisterUserRequest) { u.NIM = "22105A1001" }), wantErr: errx.ErrInvalidNIM},
		{name: "register NIM 7 digit", input: withUser(validUser, func(u *dto.RegisterUserRequest) { u.NIM = "1234567" }), wantErr: errx.ErrInvalidNIM},
		{name: "register NIM 21 digit", input: withUser(validUser, func(u *dto.RegisterUserRequest) { u.NIM = strings.Repeat("1", 21) }), wantErr: errx.ErrInvalidNIM},
		{name: "register email salah", input: withUser(validUser, func(u *dto.RegisterUserRequest) { u.Email = "budi" }), wantErr: errx.ErrInvalidEmail},
		{name: "register password 7 karakter", input: withUser(validUser, func(u *dto.RegisterUserRequest) { u.Password = "1234567" }), wantErr: errx.ErrInvalidPassword},
		{name: "register password 73 karakter", input: withUser(validUser, func(u *dto.RegisterUserRequest) { u.Password = strings.Repeat("a", 73) }), wantErr: errx.ErrInvalidPassword},
		{name: "register kontak 7 karakter", input: withUser(validUser, func(u *dto.RegisterUserRequest) { u.Contact = "0812345" }), wantErr: errx.ErrInvalidContact},
		{name: "register kontak kosong boleh", input: withUser(validUser, func(u *dto.RegisterUserRequest) { u.Contact = "" })},
		// yang dikirim error field PERTAMA yang gagal (urutan field di struct)
		{name: "register semua kosong → nama dulu", input: &dto.RegisterUserRequest{}, wantErr: errx.ErrInvalidFullName},

		// ubah user
		{name: "ubah user kosong boleh (dicek service)", input: &dto.UpdateUserRequest{}},
		{name: "ubah user email salah", input: &dto.UpdateUserRequest{Email: "budi@"}, wantErr: errx.ErrInvalidEmail},
		{name: "ubah user kontak pendek", input: &dto.UpdateUserRequest{Contact: "123"}, wantErr: errx.ErrInvalidContact},

		// login
		{name: "login valid", input: &dto.LoginRequest{Email: "budi@mail.com", Password: "x"}},
		{name: "login email kosong", input: &dto.LoginRequest{Password: "x"}, wantErr: errx.ErrInvalidEmail},
		{name: "login tanpa password", input: &dto.LoginRequest{Email: "budi@mail.com"}, wantErr: errx.ErrInvalidPassword},

		// lembaga
		{name: "lembaga valid", input: &dto.CreateLembagaRequest{Name: "BEM"}},
		{name: "lembaga nama 1 huruf", input: &dto.CreateLembagaRequest{Name: "B"}, wantErr: errx.ErrInvalidLembagaName},
		{name: "lembaga nama hanya spasi", input: &dto.CreateLembagaRequest{Name: "   "}, wantErr: errx.ErrInvalidLembagaName},
		{name: "ubah lembaga tanpa nama", input: &dto.UpdateLembagaRequest{}, wantErr: errx.ErrInvalidLembagaName},

		// barang
		{name: "barang valid", input: &validItem},
		{name: "barang total 0 boleh", input: withItem(validItem, func(i *dto.CreateItemRequest) { i.TotalQuantity = 0 })},
		{name: "barang nama hanya spasi", input: withItem(validItem, func(i *dto.CreateItemRequest) { i.Name = "  " }), wantErr: errx.ErrInvalidItemName},
		{name: "barang keterangan 1001 karakter", input: withItem(validItem, func(i *dto.CreateItemRequest) { i.Description = strings.Repeat("a", 1001) }), wantErr: errx.ErrInvalidItemDescription},
		{name: "barang tanpa lembaga", input: withItem(validItem, func(i *dto.CreateItemRequest) { i.LembagaID = 0 }), wantErr: errx.ErrInvalidItemLembaga},
		// dulu: -5 lolos validasi lalu ditolak CHECK database → 500. Sekarang 400.
		{name: "barang total negatif", input: withItem(validItem, func(i *dto.CreateItemRequest) { i.TotalQuantity = -5 }), wantErr: errx.ErrInvalidItemTotal},
		{name: "barang total lebih dari 1 juta", input: withItem(validItem, func(i *dto.CreateItemRequest) { i.TotalQuantity = 1_000_001 }), wantErr: errx.ErrInvalidItemTotal},
		{name: "ubah barang kosong boleh (dicek service)", input: &dto.UpdateItemRequest{}},
		{name: "ubah total jadi 0 boleh", input: &dto.UpdateItemRequest{TotalQuantity: ptr(0)}},
		{name: "ubah total negatif", input: &dto.UpdateItemRequest{TotalQuantity: ptr(-1)}, wantErr: errx.ErrInvalidItemTotal},
		{name: "ubah lembagaId 0", input: &dto.UpdateItemRequest{LembagaID: ptr(uint(0))}, wantErr: errx.ErrInvalidItemLembaga},
		{name: "ubah nama 1 huruf", input: &dto.UpdateItemRequest{Name: "A"}, wantErr: errx.ErrInvalidItemName},
		{name: "ubah keterangan jadi kosong boleh", input: &dto.UpdateItemRequest{Description: ptr("")}},

		// peminjaman (aturan antar-field seperti barang dobel & tanggal diuji di borrowing_create_test.go)
		{name: "pinjam valid", input: &validBorrow},
		{name: "pinjam tanpa barang", input: withBorrow(validBorrow, func(b *dto.CreateBorrowingRequest) { b.Items = nil }), wantErr: errx.ErrInvalidBorrowItems},
		{name: "pinjam daftar barang kosong", input: withBorrow(validBorrow, func(b *dto.CreateBorrowingRequest) { b.Items = []dto.BorrowingItemRequest{} }), wantErr: errx.ErrInvalidBorrowItems},
		{name: "pinjam 21 jenis barang", input: withBorrow(validBorrow, func(b *dto.CreateBorrowingRequest) {
			b.Items = nil
			for i := range 21 {
				b.Items = append(b.Items, dto.BorrowingItemRequest{ItemID: uint(i + 1), Quantity: 1})
			}
		}), wantErr: errx.ErrInvalidBorrowItems},
		{name: "pinjam 20 jenis barang boleh", input: withBorrow(validBorrow, func(b *dto.CreateBorrowingRequest) {
			b.Items = nil
			for i := range 20 {
				b.Items = append(b.Items, dto.BorrowingItemRequest{ItemID: uint(i + 1), Quantity: 1})
			}
		})},
		{name: "pinjam itemId kosong di barang ke-2", input: withBorrow(validBorrow, func(b *dto.CreateBorrowingRequest) {
			b.Items = []dto.BorrowingItemRequest{{ItemID: 1, Quantity: 1}, {ItemID: 0, Quantity: 1}}
		}), wantErr: errx.ErrInvalidBorrowItems},
		{name: "pinjam quantity 0", input: withBorrow(validBorrow, func(b *dto.CreateBorrowingRequest) { b.Items[0].Quantity = 0 }), wantErr: errx.ErrInvalidBorrowItems},
		{name: "pinjam quantity negatif", input: withBorrow(validBorrow, func(b *dto.CreateBorrowingRequest) { b.Items[0].Quantity = -2 }), wantErr: errx.ErrInvalidBorrowItems},
		{name: "pinjam tanpa lembaga", input: withBorrow(validBorrow, func(b *dto.CreateBorrowingRequest) { b.LembagaID = 0 }), wantErr: errx.ErrInvalidBorrowLembaga},
		{name: "pinjam tujuan hanya spasi", input: withBorrow(validBorrow, func(b *dto.CreateBorrowingRequest) { b.Purpose = "   " }), wantErr: errx.ErrInvalidPurpose},
		{name: "pinjam tujuan 501 karakter", input: withBorrow(validBorrow, func(b *dto.CreateBorrowingRequest) { b.Purpose = strings.Repeat("a", 501) }), wantErr: errx.ErrInvalidPurpose},
		// 500 huruf "é" = 500 karakter tapi 1000 byte: tag max menghitung karakter, jadi boleh.
		{name: "pinjam tujuan 500 karakter non-ASCII boleh", input: withBorrow(validBorrow, func(b *dto.CreateBorrowingRequest) { b.Purpose = strings.Repeat("é", 500) })},
		{name: "pinjam tanggal ambil kosong", input: withBorrow(validBorrow, func(b *dto.CreateBorrowingRequest) { b.PickUpDate = "" }), wantErr: errx.ErrInvalidPickupDate},
		{name: "pinjam format tanggal ambil salah", input: withBorrow(validBorrow, func(b *dto.CreateBorrowingRequest) { b.PickUpDate = "25-11-2026" }), wantErr: errx.ErrInvalidPickupDate},
		{name: "pinjam tanggal ambil tidak ada (30 Feb)", input: withBorrow(validBorrow, func(b *dto.CreateBorrowingRequest) { b.PickUpDate = "2026-02-30" }), wantErr: errx.ErrInvalidPickupDate},
		{name: "pinjam tanggal kembali hanya spasi", input: withBorrow(validBorrow, func(b *dto.CreateBorrowingRequest) { b.ReturnDate = " " }), wantErr: errx.ErrInvalidReturnDate},
		{name: "pinjam format tanggal kembali salah", input: withBorrow(validBorrow, func(b *dto.CreateBorrowingRequest) { b.ReturnDate = "besok" }), wantErr: errx.ErrInvalidReturnDate},
		// pengembalian (ukuran & jenis gambar dicek service, lihat borrowing_return_test.go)
		{name: "pengembalian tanpa foto", input: &dto.SubmitReturnRequest{}, wantErr: errx.ErrInvalidReturnPhoto},

		// yang dikirim error field PERTAMA: barang → lembaga → tujuan → tanggal
		{name: "pinjam semua kosong → barang dulu", input: &dto.CreateBorrowingRequest{}, wantErr: errx.ErrInvalidBorrowItems},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := validator.ValidateStruct(tt.input)
			if tt.wantErr == nil {
				if err != nil {
					t.Errorf("ValidateStruct(%+v) error = %v; want nil", tt.input, err)
				}
				return
			}
			de := assertDomainError(t, err, tt.wantErr.Code, errx.KindInvalid)
			if de != nil && de.Message != tt.wantErr.Message {
				t.Errorf("pesan = %q; want %q", de.Message, tt.wantErr.Message)
			}
		})
	}
}

func TestValidateStruct_TrimsText(t *testing.T) {
	t.Parallel()
	req := dto.RegisterUserRequest{
		FullName: "  Ani  ", NIM: " 12345678 ", Email: " ani@mail.com ", Password: " pass 12 ", Contact: " 08123456789 ",
	}
	mustNoErr(t, validator.ValidateStruct(&req))
	if req.FullName != "Ani" || req.NIM != "12345678" || req.Email != "ani@mail.com" || req.Contact != "08123456789" {
		t.Errorf("hasil = %+v; want nama, NIM, email, kontak sudah di-trim", req)
	}
	// Password diberi tag trim:"-": spasi bisa bagian dari password, jadi tidak dibuang.
	if req.Password != " pass 12 " {
		t.Errorf("password = %q; want tetap \" pass 12 \"", req.Password)
	}

	// *string juga di-trim (keterangan barang saat diubah).
	update := dto.UpdateItemRequest{Description: ptr("  cadangan  ")}
	mustNoErr(t, validator.ValidateStruct(&update))
	if *update.Description != "cadangan" {
		t.Errorf("keterangan = %q; want cadangan", *update.Description)
	}

	// Tujuan & tanggal peminjaman juga di-trim sebelum dicek.
	borrow := dto.CreateBorrowingRequest{
		Items:     []dto.BorrowingItemRequest{{ItemID: 1, Quantity: 1}},
		LembagaID: 1, Purpose: "  Seminar nasional  ", PickUpDate: " 2026-12-01 ", ReturnDate: "2026-12-03 ",
	}
	mustNoErr(t, validator.ValidateStruct(&borrow))
	if borrow.Purpose != "Seminar nasional" || borrow.PickUpDate != "2026-12-01" || borrow.ReturnDate != "2026-12-03" {
		t.Errorf("hasil = %+v; want tujuan & tanggal sudah di-trim", borrow)
	}
}

// noMapperRequest tidak punya FieldErrors(), jadi error-nya memakai cadangan errx.Validation.
type noMapperRequest struct {
	Code string `validate:"required"`
}

func TestValidateStruct_FallbackWithoutMapping(t *testing.T) {
	t.Parallel()
	err := validator.ValidateStruct(&noMapperRequest{})
	de := assertDomainError(t, err, "VALIDATION_ERROR", errx.KindInvalid)
	if de != nil && (de.Message != "Data tidak valid" || de.Err == nil) {
		t.Errorf("error = %q (detail %v); want \"Data tidak valid\" + detail validator", de.Message, de.Err)
	}

	// Bukan struct = kesalahan kode, bukan kesalahan user: tidak boleh jadi DomainError (dibalas 500).
	err = validator.ValidateStruct("bukan struct")
	if _, ok := errx.As(err); ok || err == nil {
		t.Errorf("ValidateStruct(string) = %v; want error biasa (500)", err)
	}
	var de2 *errx.DomainError
	if errors.As(err, &de2) {
		t.Error("error bukan-struct terbaca sebagai DomainError")
	}
}

func withUser(u dto.RegisterUserRequest, change func(*dto.RegisterUserRequest)) *dto.RegisterUserRequest {
	change(&u)
	return &u
}

func withItem(i dto.CreateItemRequest, change func(*dto.CreateItemRequest)) *dto.CreateItemRequest {
	change(&i)
	return &i
}

// withBorrow menyalin form (termasuk daftar barangnya) lalu mengubahnya, supaya kasus lain tidak ikut berubah.
func withBorrow(b dto.CreateBorrowingRequest, change func(*dto.CreateBorrowingRequest)) *dto.CreateBorrowingRequest {
	b.Items = slices.Clone(b.Items)
	change(&b)
	return &b
}
