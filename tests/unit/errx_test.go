package unit

import (
	"errors"
	"fmt"
	"testing"

	"SGEInventory_BE/domain/errx"
)

func TestDomainErrorIs_MatchesByCode(t *testing.T) {
	t.Parallel()
	wrapped := fmt.Errorf("find item 7: %w", errx.ErrItemNotFound) // dibungkus seperti di repository

	if !errors.Is(wrapped, errx.ErrItemNotFound) {
		t.Error("errors.Is(wrapped, ErrItemNotFound) = false; want true (error terbungkus tetap dikenali)")
	}
	if errors.Is(wrapped, errx.ErrUserNotFound) {
		t.Error("errors.Is(ErrItemNotFound, ErrUserNotFound) = true; want false (Code berbeda)")
	}
	// Error buatan fungsi (pesan berbeda) tetap cocok karena yang dibandingkan Code-nya.
	if !errors.Is(errx.NewBorrowItemIDRequired(2), errx.NewBorrowItemIDRequired(5)) {
		t.Error("errors.Is dengan Code sama tapi pesan beda = false; want true")
	}
}

func TestDomainErrorAs_UnwrapsAndKeepsMessage(t *testing.T) {
	t.Parallel()
	err := fmt.Errorf("service: %w", errx.ErrEmailTaken)

	de, ok := errx.As(err)
	if !ok {
		t.Fatalf("errx.As(%v) ok = false; want true", err)
	}
	if de.Kind != errx.KindConflict || de.Message != "Email sudah terdaftar" {
		t.Errorf("As() = kind %d pesan %q; want KindConflict \"Email sudah terdaftar\"", de.Kind, de.Message)
	}
	if de.Error() != de.Message {
		t.Errorf("Error() = %q; want sama dengan Message %q", de.Error(), de.Message)
	}

	if _, ok := errx.As(errors.New("database mati")); ok {
		t.Error("errx.As(error biasa) ok = true; want false")
	}
}

func TestValidation_WrapsDetail(t *testing.T) {
	t.Parallel()
	detail := errors.New("Key: 'Name' Error:Field validation for 'Name' failed on the 'required' tag")
	err := errx.Validation(detail)

	if err.Kind != errx.KindInvalid || err.Code != "VALIDATION_ERROR" {
		t.Errorf("Validation() = kind %d code %s; want KindInvalid VALIDATION_ERROR", err.Kind, err.Code)
	}
	if !errors.Is(err, detail) {
		t.Error("errors.Is(Validation(detail), detail) = false; want true (Unwrap mengembalikan detail)")
	}
}
