package config

import "github.com/go-playground/validator/v10"

// Validator membuat c.Bind().Body(&req) otomatis mengecek tag `validate:"..."`.
type Validator struct {
	validate *validator.Validate
}

func NewValidator() *Validator {
	return &Validator{validate: validator.New()}
}

func (v *Validator) Validate(out any) error {
	return v.validate.Struct(out)
}
