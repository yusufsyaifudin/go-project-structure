package httpservermw_test

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yusufsyaifudin/go-project-structure/internal/pkg/httpservermw"
)

func TestRemoveTrailingSlash(t *testing.T) {
	handlerMock := &mockHandler{
		responseCode: http.StatusOK,
		responseHeader: map[string]string{
			"Content-Type": "application/json",
		},
		responseBody: `{"FOO":"BAR"}`,
	}

	t.Run("normal case", func(t *testing.T) {
		handler := httpservermw.RemoveTrailingSlash(handlerMock)

		req, err := http.NewRequest(http.MethodPost, "http://localhost/ping/////", bytes.NewBufferString(`{"foo":"bar"}`))
		assert.NotNil(t, req)
		assert.NoError(t, err)
		assert.Equal(t, "/ping/////", req.URL.Path)

		resp := httptest.NewRecorder()
		handler.ServeHTTP(resp, req)
		assert.Equal(t, "/ping", req.URL.Path)
	})
}
