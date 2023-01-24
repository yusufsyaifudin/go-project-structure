package respbuilder_test

import (
	"fmt"
	"testing"

	"github.com/labstack/echo/v4"

	"github.com/stretchr/testify/assert"

	"github.com/yusufsyaifudin/go-project-structure/pkg/respbuilder"
)

func TestError(t *testing.T) {
	t.Run("all available codes", func(t *testing.T) {
		codes := respbuilder.GetAllRespCodeErr()
		for _, code := range codes {
			t.Run(fmt.Sprintf("code %d", code), func(t *testing.T) {
				resp := respbuilder.Error(code, fmt.Errorf("error content"), "error warning")
				assert.Equal(t, respbuilder.RespCodeErrStatus(code).Code, resp.Code)
				assert.Equal(t, respbuilder.RespCodeErrStatus(code).Status, resp.Status)
			})
		}
	})

	t.Run("nil error", func(t *testing.T) {
		generalErrCode := respbuilder.ErrGeneral
		resp := respbuilder.Error(generalErrCode, nil)
		assert.Equal(t, respbuilder.RespCodeErrStatus(generalErrCode).Code, resp.Code)
		assert.Equal(t, respbuilder.RespCodeErrStatus(generalErrCode).Status, resp.Status)
	})

	t.Run("unknown code", func(t *testing.T) {
		errCodeUnknown := respbuilder.ErrUnknown
		resp := respbuilder.Error(respbuilder.RespCodeErr(-1), nil)
		assert.Equal(t, respbuilder.RespCodeErrStatus(errCodeUnknown).Code, resp.Code)
		assert.Equal(t, respbuilder.RespCodeErrStatus(errCodeUnknown).Status, resp.Status)
	})

	t.Run("echo http error", func(t *testing.T) {
		errCode := respbuilder.ErrGeneral
		resp := respbuilder.Error(errCode, &echo.HTTPError{
			Code:    404,
			Message: "Not Found",
		})
		assert.Equal(t, respbuilder.RespCodeErrStatus(errCode).Code, resp.Code)
		assert.Equal(t, respbuilder.RespCodeErrStatus(errCode).Status, resp.Status)
	})

	t.Run("echo binding error", func(t *testing.T) {
		errCode := respbuilder.ErrGeneral
		resp := respbuilder.Error(errCode, &echo.BindingError{
			Field:  "key",
			Values: nil,
			HTTPError: &echo.HTTPError{
				Code:    404,
				Message: "Not Found",
			},
		})
		assert.Equal(t, respbuilder.RespCodeErrStatus(errCode).Code, resp.Code)
		assert.Equal(t, respbuilder.RespCodeErrStatus(errCode).Status, resp.Status)
	})
}
