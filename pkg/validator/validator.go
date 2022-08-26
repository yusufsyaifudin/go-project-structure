package validator

import (
	"github.com/go-playground/validator/v10"
)

var (
	v *validator.Validate
)

func init() {
	v = validator.New()
}

func Validate(i interface{}) error {
	return v.Struct(i)
}
