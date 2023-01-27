package httpservermw_test

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel/trace"

	"github.com/yusufsyaifudin/go-project-structure/internal/pkg/httpservermw"
	"github.com/yusufsyaifudin/go-project-structure/pkg/ylog"
)

var logMwTest = &httpservermw.LogMiddleware{}

func TestLogMwWithMessage(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		opt := httpservermw.LogMwWithMessage("")
		err := opt(logMwTest)
		assert.NoError(t, err)
	})

	t.Run("non-empty", func(t *testing.T) {
		opt := httpservermw.LogMwWithMessage("message")
		err := opt(logMwTest)
		assert.NoError(t, err)
	})
}

func TestLogMwWithLogger(t *testing.T) {
	t.Run("nil", func(t *testing.T) {
		opt := httpservermw.LogMwWithLogger(nil)
		err := opt(logMwTest)
		assert.NoError(t, err)
	})

	t.Run("non-nil", func(t *testing.T) {
		opt := httpservermw.LogMwWithLogger(ylog.NewNoop())
		err := opt(logMwTest)
		assert.NoError(t, err)
	})
}

func TestLogMwWithTracer(t *testing.T) {
	t.Run("nil", func(t *testing.T) {
		opt := httpservermw.LogMwWithTracer(nil)
		err := opt(logMwTest)
		assert.NoError(t, err)
	})

	t.Run("non-nil", func(t *testing.T) {
		opt := httpservermw.LogMwWithTracer(trace.NewNoopTracerProvider())
		err := opt(logMwTest)
		assert.NoError(t, err)
	})
}

func TestLogMwWithFilter(t *testing.T) {
	t.Run("nil", func(t *testing.T) {
		opt := httpservermw.LogMwWithFilter(nil)
		err := opt(logMwTest)
		assert.NoError(t, err)
	})
}

func TestLoggingMiddleware(t *testing.T) {
	handlerMock := &mockHandler{
		responseCode: http.StatusOK,
		responseHeader: map[string]string{
			"Content-Type": "application/json",
		},
		responseBody: `{"FOO":"BAR"}`,
	}

	t.Run("with filter", func(t *testing.T) {
		handler := httpservermw.LoggingMiddleware(handlerMock, httpservermw.LogMwWithFilter(func(request *http.Request) bool {
			return false
		}))

		req, err := http.NewRequest(http.MethodPost, "http://localhost", bytes.NewBufferString(`{"foo":"bar"}`))
		assert.NotNil(t, req)
		assert.NoError(t, err)

		resp := httptest.NewRecorder()
		handler.ServeHTTP(resp, req)
		assert.Equal(t, handlerMock.responseCode, resp.Code)
		assert.Equal(t, handlerMock.responseHeader, httpservermw.HttpHeaderToSimpleMap(resp.Header()))
	})

	t.Run("normal case", func(t *testing.T) {
		handler := httpservermw.LoggingMiddleware(handlerMock)

		req, err := http.NewRequest(http.MethodPost, "http://localhost", bytes.NewBufferString(`{"foo":"bar"}`))
		assert.NotNil(t, req)
		assert.NoError(t, err)

		resp := httptest.NewRecorder()
		handler.ServeHTTP(resp, req)
		assert.Equal(t, handlerMock.responseCode, resp.Code)
		assert.Equal(t, handlerMock.responseHeader, httpservermw.HttpHeaderToSimpleMap(resp.Header()))
	})

	t.Run("non-valid json request body", func(t *testing.T) {
		handler := httpservermw.LoggingMiddleware(handlerMock)

		req, err := http.NewRequest(http.MethodPost, "http://localhost", bytes.NewBufferString(`<foo>bar</foo>`))
		assert.NotNil(t, req)
		assert.NoError(t, err)

		resp := httptest.NewRecorder()
		handler.ServeHTTP(resp, req)
		assert.Equal(t, handlerMock.responseCode, resp.Code)
		assert.Equal(t, handlerMock.responseHeader, httpservermw.HttpHeaderToSimpleMap(resp.Header()))
	})

	t.Run("failed copy request body", func(t *testing.T) {
		handler := httpservermw.LoggingMiddleware(handlerMock)

		req, err := http.NewRequest(http.MethodPost, "http://localhost", bytes.NewBufferString(`{"foo":"bar"}`))
		req.Body = io.NopCloser(newBuf(fmt.Errorf("mock buffer error")))
		assert.NotNil(t, req)
		assert.NoError(t, err)

		resp := httptest.NewRecorder()
		handler.ServeHTTP(resp, req)
		assert.Equal(t, handlerMock.responseCode, resp.Code)
		assert.Equal(t, handlerMock.responseHeader, httpservermw.HttpHeaderToSimpleMap(resp.Header()))
	})

	t.Run("failed close request body", func(t *testing.T) {
		handler := httpservermw.LoggingMiddleware(handlerMock)

		req, err := http.NewRequest(http.MethodPost, "http://localhost", bytes.NewBufferString(`{"foo":"bar"}`))
		req.Body = newCloser(bytes.NewBufferString(`{"foo":"bar"}`), fmt.Errorf("mock error closing request body"))
		assert.NotNil(t, req)
		assert.NoError(t, err)

		resp := httptest.NewRecorder()
		handler.ServeHTTP(resp, req)
		assert.Equal(t, handlerMock.responseCode, resp.Code)
		assert.Equal(t, handlerMock.responseHeader, httpservermw.HttpHeaderToSimpleMap(resp.Header()))
	})

	t.Run("invalid json response body", func(t *testing.T) {
		handlerMockInvalidJsonResp := &mockHandler{
			responseCode: http.StatusOK,
			responseHeader: map[string]string{
				"Content-Type": "application/json",
			},
			responseBody: `<FOO>BAR</FOO>`,
		}

		handler := httpservermw.LoggingMiddleware(handlerMockInvalidJsonResp)

		req, err := http.NewRequest(http.MethodPost, "http://localhost", bytes.NewBufferString(`{"foo":"bar"}`))
		assert.NotNil(t, req)
		assert.NoError(t, err)

		resp := httptest.NewRecorder()
		handler.ServeHTTP(resp, req)
		assert.Equal(t, handlerMock.responseCode, resp.Code)
		assert.Equal(t, handlerMock.responseHeader, httpservermw.HttpHeaderToSimpleMap(resp.Header()))
	})

	t.Run("failed to write response body", func(t *testing.T) {
		handler := httpservermw.LoggingMiddleware(handlerMock)

		req, err := http.NewRequest(http.MethodPost, "http://localhost", bytes.NewBufferString(`{"foo":"bar"}`))
		assert.NotNil(t, req)
		assert.NoError(t, err)

		errFailedWriteRespBody := fmt.Errorf("mock failed write")
		resp := &mockRespWriter{
			header: map[string][]string{},
			write: func(b []byte) (int, error) {
				return 0, errFailedWriteRespBody
			},
			code: http.StatusOK,
		}
		handler.ServeHTTP(resp, req)
		assert.Equal(t, handlerMock.responseCode, resp.code)
		assert.Equal(t, handlerMock.responseHeader, httpservermw.HttpHeaderToSimpleMap(resp.Header()))
	})
}
