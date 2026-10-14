package validator

import (
	"errors"

	"github.com/go-playground/validator/v10"
)

var v *validator.Validate

func init() {
	v = validator.New()
}

// Validate runs struct-level validation using the validate struct tags.
func Validate(i interface{}) error {
	return v.Struct(i)
}

// Reasons extracts each field-level error message from a ValidationErrors
// returned by Validate. Returns nil when err is not a ValidationErrors.
func Reasons(err error) []string {
	if err == nil {
		return nil
	}
	var ve validator.ValidationErrors
	if !errors.As(err, &ve) {
		return nil
	}
	reasons := make([]string, len(ve))
	for i, fe := range ve {
		reasons[i] = fe.Error()
	}
	return reasons
}
