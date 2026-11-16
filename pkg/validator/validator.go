package validator

import "github.com/go-playground/validator/v10"

var validate = validator.New()

// ValidateStruct mengecek tag `validate:"..."` pada sebuah struct (biasanya DTO request).
func ValidateStruct(s any) error {
	return validate.Struct(s)
}
