package respbuilder_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/yusufsyaifudin/go-project-structure/pkg/respbuilder"
)

func TestOk(t *testing.T) {
	t.Run("all available codes", func(t *testing.T) {
		codes := respbuilder.GetAllRespCodeOk()
		for _, code := range codes {
			t.Run(fmt.Sprintf("code %d", code), func(t *testing.T) {
				resp := respbuilder.Ok(code, struct{}{}, "error warning")
				assert.Equal(t, respbuilder.RespCodeOkStatus(code).Code, resp.Code)
				assert.Equal(t, respbuilder.RespCodeOkStatus(code).Status, resp.Status)
			})
		}
	})

	t.Run("unknown code", func(t *testing.T) {
		errCodeUnknown := respbuilder.UnknownOk
		resp := respbuilder.Ok(respbuilder.RespCodeOk(-1), nil)
		assert.Equal(t, respbuilder.RespCodeOkStatus(errCodeUnknown).Code, resp.Code)
		assert.Equal(t, respbuilder.RespCodeOkStatus(errCodeUnknown).Status, resp.Status)
	})
}
