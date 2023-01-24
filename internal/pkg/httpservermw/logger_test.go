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

func TestLoggingMiddleware(t *testing.T) {
	handlerMock := &mockHandler{
		responseCode: http.StatusOK,
		responseHeader: map[string]string{
			"Content-Type": "application/json",
		},
		responseBody: `{"FOO":"BAR"}`,
	}

	t.Run("normal case", func(t *testing.T) {
		handler := httpservermw.LoggingMiddleware(handlerMock)

		req, err := http.NewRequest(http.MethodPost, "http://localhost", bytes.NewBufferString(`{"foo":"bar"}`))
		assert.NotNil(t, req)
		assert.NoError(t, err)

		resp := httptest.NewRecorder()
		handler.ServeHTTP(resp, req)
		assert.Equal(t, resp.Code, handlerMock.responseCode)
		assert.Equal(t, httpservermw.HttpHeaderToSimpleMap(resp.Header()), handlerMock.responseHeader)
	})

	t.Run("non-valid json request body", func(t *testing.T) {
		handler := httpservermw.LoggingMiddleware(handlerMock)

		req, err := http.NewRequest(http.MethodPost, "http://localhost", bytes.NewBufferString(`<foo>bar</foo>`))
		assert.NotNil(t, req)
		assert.NoError(t, err)

		resp := httptest.NewRecorder()
		handler.ServeHTTP(resp, req)
		assert.Equal(t, resp.Code, handlerMock.responseCode)
		assert.Equal(t, httpservermw.HttpHeaderToSimpleMap(resp.Header()), handlerMock.responseHeader)
	})

	t.Run("failed copy request body", func(t *testing.T) {
		handler := httpservermw.LoggingMiddleware(handlerMock)

		req, err := http.NewRequest(http.MethodPost, "http://localhost", bytes.NewBufferString(`{"foo":"bar"}`))
		req.Body = io.NopCloser(newBuf(fmt.Errorf("mock buffer error")))
		assert.NotNil(t, req)
		assert.NoError(t, err)

		resp := httptest.NewRecorder()
		handler.ServeHTTP(resp, req)
		assert.Equal(t, resp.Code, handlerMock.responseCode)
		assert.Equal(t, httpservermw.HttpHeaderToSimpleMap(resp.Header()), handlerMock.responseHeader)
	})

	t.Run("failed close request body", func(t *testing.T) {
		handler := httpservermw.LoggingMiddleware(handlerMock)

		req, err := http.NewRequest(http.MethodPost, "http://localhost", bytes.NewBufferString(`{"foo":"bar"}`))
		req.Body = newCloser(bytes.NewBufferString(`{"foo":"bar"}`), fmt.Errorf("mock error closing request body"))
		assert.NotNil(t, req)
		assert.NoError(t, err)

		resp := httptest.NewRecorder()
		handler.ServeHTTP(resp, req)
		assert.Equal(t, resp.Code, handlerMock.responseCode)
		assert.Equal(t, httpservermw.HttpHeaderToSimpleMap(resp.Header()), handlerMock.responseHeader)
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
		assert.Equal(t, resp.Code, handlerMock.responseCode)
		assert.Equal(t, httpservermw.HttpHeaderToSimpleMap(resp.Header()), handlerMock.responseHeader)
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
		assert.Equal(t, resp.code, handlerMock.responseCode)
		assert.Equal(t, httpservermw.HttpHeaderToSimpleMap(resp.Header()), handlerMock.responseHeader)
	})
}

type mockHandler struct {
	responseCode   int
	responseHeader map[string]string
	responseBody   string
}

func (m *mockHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	for k, v := range m.responseHeader {
		w.Header().Set(k, v)
	}

	w.WriteHeader(m.responseCode)
	_, _ = w.Write([]byte(m.responseBody))

}

type buf struct {
	err error
}

var _ io.Reader = (*buf)(nil)

func newBuf(err error) io.Reader {
	return &buf{
		err: err,
	}
}

func (b *buf) Read(p []byte) (n int, err error) {
	if b.err != nil {
		return 0, b.err
	}

	return len(p), nil
}

type closer struct {
	buf io.Reader
	err error
}

var _ io.ReadCloser = (*closer)(nil)

func newCloser(buf io.Reader, err error) *closer {
	return &closer{
		buf: buf,
		err: err,
	}
}

func (c *closer) Read(p []byte) (n int, err error) {
	return c.buf.Read(p)
}

func (c *closer) Close() error {
	return c.err
}

type mockRespWriter struct {
	header http.Header
	write  func(b []byte) (int, error)
	code   int
}

var _ http.ResponseWriter = (*mockRespWriter)(nil)

func (m *mockRespWriter) Header() http.Header {
	return m.header
}

func (m *mockRespWriter) Write(i []byte) (int, error) {
	return m.write(i)
}

func (m *mockRespWriter) WriteHeader(_ int) {}
