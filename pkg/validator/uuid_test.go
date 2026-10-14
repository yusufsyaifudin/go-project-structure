package validator_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/yusufsyaifudin/go-project-structure/pkg/validator"
)

func TestIsStrictUUIDv4(t *testing.T) {
	cases := []struct {
		name  string
		input string
		want  bool
	}{
		{"valid v4 lowercase", "f47ac10b-58cc-4372-a567-0e02b2c3d479", true},
		{"valid v4 uppercase", "F47AC10B-58CC-4372-A567-0E02B2C3D479", true},
		{"valid v4 new random", "550e8400-e29b-4372-a716-446655440000", true},
		{"v1 uuid rejected", "6ba7b810-9dad-11d1-80b4-00c04fd430c8", false},
		{"v5 uuid rejected", "886313e1-3b8a-5372-9b90-0c9aee199e5d", false},
		{"empty string", "", false},
		{"not a uuid", "not-a-uuid", false},
		{"too short", "f47ac10b-58cc-4372", false},
		{"all zeros", "00000000-0000-0000-0000-000000000000", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, validator.IsStrictUUIDv4(tc.input))
		})
	}
}

func TestReasons_Nil(t *testing.T) {
	assert.Nil(t, validator.Reasons(nil))
}

func TestReasons_NonValidationError(t *testing.T) {
	assert.Nil(t, validator.Reasons(errors.New("plain error")))
}

func TestReasons_ValidationErrors(t *testing.T) {
	type req struct {
		Name string `validate:"required"`
		Age  int    `validate:"gt=0"`
	}
	err := validator.Validate(req{})
	assert.Error(t, err)

	reasons := validator.Reasons(err)
	assert.NotNil(t, reasons)
	assert.Len(t, reasons, 2)
	for _, r := range reasons {
		assert.NotEmpty(t, r)
	}
}

func TestReasons_SingleValidationError(t *testing.T) {
	type req struct {
		Name string `validate:"required"`
	}
	err := validator.Validate(req{})
	assert.Error(t, err)
	reasons := validator.Reasons(err)
	assert.Len(t, reasons, 1)
}
