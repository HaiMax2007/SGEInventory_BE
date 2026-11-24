// Package validator mengecek tag `validate:"..."` pada DTO request, lalu mengubah hasilnya menjadi
// error errx yang pesannya jelas untuk FE (mis. errx.ErrInvalidEmail), bukan pesan mentah dari library.
package validator

import (
	"errors"
	"reflect"
	"strings"

	"github.com/go-playground/validator/v10"

	"SGEInventory_BE/domain/errx"
)

var validate = validator.New()

// FieldErrorMapper dimiliki DTO yang memetakan nama field Go → error errx.
// Contoh: RegisterUserRequest.FieldErrors() berisi {"Email": errx.ErrInvalidEmail, ...}.
type FieldErrorMapper interface {
	FieldErrors() map[string]error
}

// ValidateStruct merapikan lalu mengecek DTO. s harus POINTER ke struct (mis. &req), supaya hasil trim tersimpan.
//
//  1. Spasi di awal/akhir semua field teks dibuang (lihat trimStrings), jadi "   " dianggap kosong.
//  2. Tag `validate:"..."` dicek.
//  3. Field PERTAMA yang gagal diterjemahkan lewat FieldErrors() milik DTO, mis. Email → errx.ErrInvalidEmail.
//     Field yang belum dipetakan → errx.Validation (400 "Data tidak valid" + detail di "error").
func ValidateStruct(s any) error {
	trimStrings(s)

	err := validate.Struct(s)
	if err == nil {
		return nil
	}

	var fieldErrs validator.ValidationErrors
	if !errors.As(err, &fieldErrs) {
		return err // s bukan struct: kesalahan kode (bug), jadi dibalas 500
	}
	if mapper, ok := s.(FieldErrorMapper); ok {
		if fieldErr, ok := mapper.FieldErrors()[fieldErrs[0].StructField()]; ok {
			return fieldErr
		}
	}
	return errx.Validation(err)
}

// trimStrings membuang spasi di awal/akhir setiap field string dan *string pada struct yang ditunjuk s.
// Field dengan tag `trim:"-"` dilewati (dipakai password, karena spasi bisa bagian dari password).
func trimStrings(s any) {
	v := reflect.ValueOf(s)
	if v.Kind() != reflect.Pointer || v.Elem().Kind() != reflect.Struct {
		return
	}
	v = v.Elem()
	for i := range v.NumField() {
		field := v.Field(i)
		if v.Type().Field(i).Tag.Get("trim") == "-" || !field.CanSet() {
			continue
		}
		switch {
		case field.Kind() == reflect.String:
			field.SetString(strings.TrimSpace(field.String()))
		case field.Kind() == reflect.Pointer && !field.IsNil() && field.Elem().Kind() == reflect.String:
			field.Elem().SetString(strings.TrimSpace(field.Elem().String()))
		}
	}
}
