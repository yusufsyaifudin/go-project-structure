package validator_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yusufsyaifudin/go-project-structure/pkg/validator"
)

func TestValidate(t *testing.T) {

	testCases := []struct {
		Str string `validate:"required"`
		Err bool
	}{
		{
			Str: "",
			Err: true,
		},
		{
			Str: "abc",
			Err: false,
		},
		{
			Str: "abc-abc",
			Err: false,
		},
		{
			Str: "123-abc",
			Err: false,
		},
		{
			Str: "123-Abc",
			Err: false,
		},
		{
			Str: "123-Abc name-label",
			Err: false,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.Str, func(t *testing.T) {
			err := validator.Validate(testCase)
			if !testCase.Err {
				assert.NoError(t, err)
				return
			}

			assert.Error(t, err)
		})
	}

}
