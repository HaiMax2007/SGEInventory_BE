package errx

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
)

var (
	ErrInvalidID       = New(KindInvalid, "INVALID_ID", "ID tidak valid")
	ErrNothingToUpdate = New(KindInvalid, "NOTHING_TO_UPDATE", "Tidak ada data yang diperbarui")

	// ErrInvalidBody: body bukan JSON yang benar (rusak, kosong, atau Content-Type bukan application/json).
	ErrInvalidBody = New(KindInvalid, "INVALID_BODY", "Format data tidak valid, kirim body JSON yang benar")
)

// NewInvalidFieldType: tipe data sebuah field salah, mis. totalQuantity dikirim "lima" (teks), bukan 5 (angka).
// field adalah nama field di JSON, want adalah tipe yang seharusnya (mis. "angka bulat").
func NewInvalidFieldType(field, want string) *DomainError {
	return New(KindInvalid, "INVALID_FIELD_TYPE",
		fmt.Sprintf("Field %s harus berupa %s", field, want))
}

// InvalidBody mengubah error dari c.Bind().Body(...) menjadi error yang aman & jelas untuk FE.
// Pesan asli dari Go (mis. "json: cannot unmarshal string into Go struct field ...") tidak dikirim,
// karena membocorkan nama struct di kode dan sulit dipahami user.
//
//   - tipe data field salah → NewInvalidFieldType (nama field disebut)
//   - selain itu (JSON rusak, body kosong, Content-Type salah) → ErrInvalidBody
func InvalidBody(err error) *DomainError {
	// errors.As tetap menemukan error JSON walaupun dibungkus Fiber (fiber.BindError punya Unwrap).
	var typeErr *json.UnmarshalTypeError
	if errors.As(err, &typeErr) && typeErr.Field != "" {
		return NewInvalidFieldType(typeErr.Field, typeName(typeErr.Type))
	}
	return ErrInvalidBody
}

// typeName menerjemahkan tipe Go menjadi nama yang mudah dipahami user.
func typeName(t reflect.Type) string {
	if t == nil {
		return "tipe yang benar"
	}
	switch t.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return "angka bulat" // mis. 5 (bukan "5" atau 1.5)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return "angka bulat positif" // dipakai untuk id, mis. lembagaId: -1 ditolak di sini
	case reflect.Float32, reflect.Float64:
		return "angka"
	case reflect.String:
		return "teks"
	case reflect.Bool:
		return "true/false"
	case reflect.Slice, reflect.Array:
		return "daftar (array)"
	case reflect.Struct, reflect.Map:
		return "objek"
	default:
		return "tipe yang benar"
	}
}
