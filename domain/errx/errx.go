// Package errx berisi error bisnis (domain error) aplikasi.
//
// Service & repository cukup mengembalikan error dari package ini (mis. errx.ErrUserNotFound).
// middlewares.ErrorHandler yang menerjemahkannya menjadi status HTTP + JSON,
// jadi handler tidak perlu menulis c.Status(...).JSON(...) untuk setiap error.
package errx

import "errors"

// Kind menentukan status HTTP dari sebuah error (lihat middlewares.ErrorHandler).
type Kind int

const (
	KindInvalid      Kind = iota + 1 // 400 Bad Request
	KindUnauthorized                 // 401 Unauthorized
	KindForbidden                    // 403 Forbidden
	KindNotFound                     // 404 Not Found
	KindConflict                     // 409 Conflict
)

// DomainError adalah error yang pesannya aman ditampilkan ke client.
type DomainError struct {
	Kind    Kind
	Code    string // kode singkat untuk dibaca programmer, mis. "USER_NOT_FOUND"
	Message string // pesan untuk user/FE
	Err     error  // detail tambahan (opsional), mis. hasil validator
}

func (e *DomainError) Error() string {
	return e.Message
}

func (e *DomainError) Unwrap() error {
	return e.Err
}

// Is membuat errors.Is(err, errx.ErrUserNotFound) bernilai true berdasarkan Code.
func (e *DomainError) Is(target error) bool {
	if t, ok := target.(*DomainError); ok {
		return e.Code == t.Code
	}
	return false
}

// New membuat DomainError baru.
func New(kind Kind, code, message string) *DomainError {
	return &DomainError{Kind: kind, Code: code, Message: message}
}

// As mengambil DomainError dari sebuah error (termasuk error yang dibungkus).
func As(err error) (*DomainError, bool) {
	var de *DomainError
	if errors.As(err, &de) {
		return de, true
	}
	return nil, false
}

// Validation membungkus error dari validator menjadi 400 "Data tidak valid".
// Dipakai sebagai cadangan untuk field yang belum dipetakan di FieldErrors() DTO.
// Detailnya (err) ikut dikirim ke client di field "error".
func Validation(err error) *DomainError {
	return &DomainError{Kind: KindInvalid, Code: "VALIDATION_ERROR", Message: "Data tidak valid", Err: err}
}
