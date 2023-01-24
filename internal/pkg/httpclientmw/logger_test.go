package httpclientmw

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace"

	"github.com/yusufsyaifudin/go-project-structure/pkg/ylog"
)

var noopTracer = trace.NewNoopTracerProvider()

var roundTripperInstance = &roundTripper{
	base:           http.DefaultTransport,
	msg:            "request logger",
	logger:         ylog.NewNoop(),
	tracerProvider: noopTracer,
	tracer:         noopTracer.Tracer(instrumentationName),
}

func TestWithBaseRoundTripper(t *testing.T) {
	t.Run("on nil", func(t *testing.T) {
		opt := WithBaseRoundTripper(nil)
		err := opt(roundTripperInstance)
		assert.NoError(t, err)
	})

	t.Run("on non-nil", func(t *testing.T) {
		opt := WithBaseRoundTripper(http.DefaultTransport)
		err := opt(roundTripperInstance)
		assert.NoError(t, err)
	})
}

func TestWithMessage(t *testing.T) {
	t.Run("on empty", func(t *testing.T) {
		opt := WithMessage("")
		err := opt(roundTripperInstance)
		assert.NoError(t, err)
	})

	t.Run("on non-empty", func(t *testing.T) {
		opt := WithMessage("message log")
		err := opt(roundTripperInstance)
		assert.NoError(t, err)
	})
}

func TestWithLogger(t *testing.T) {
	t.Run("on nil", func(t *testing.T) {
		opt := WithLogger(nil)
		err := opt(roundTripperInstance)
		assert.NoError(t, err)
	})

	t.Run("on non-nil", func(t *testing.T) {
		opt := WithLogger(ylog.NewNoop())
		err := opt(roundTripperInstance)
		assert.NoError(t, err)
	})
}

func TestWithTracer(t *testing.T) {
	t.Run("on nil", func(t *testing.T) {
		opt := WithTracer(nil)
		err := opt(roundTripperInstance)
		assert.NoError(t, err)
	})

	t.Run("on non-nil", func(t *testing.T) {
		opt := WithTracer(noopTracer)
		err := opt(roundTripperInstance)
		assert.NoError(t, err)
	})
}

func TestNewHttpRoundTripper(t *testing.T) {
	t.Run("ok", func(t *testing.T) {
		mw := NewHttpRoundTripper()
		assert.NotNil(t, mw)
	})
}

func TestRoundTripper_RoundTrip(t *testing.T) {

	t.Run("good condition", func(t *testing.T) {
		transport := newMockHTTPRoundTripper()
		transport.CallRoundTrip = func(request *http.Request) (*http.Response, error) {
			return &http.Response{
				Header: map[string][]string{
					"Content-Type": {"application/json"},
				},
				Body: io.NopCloser(bytes.NewBufferString(`{"FOO":"BAR"}`)),
			}, nil
		}

		mw := NewHttpRoundTripper(WithBaseRoundTripper(transport))
		assert.NotNil(t, mw)

		host, _ := url.Parse("https://localhost")
		req := &http.Request{
			URL: host,
			Header: map[string][]string{
				"Content-Type": {"application/json"},
			},
			Body: io.NopCloser(bytes.NewBufferString(`{"foo":"bar"}`)),
		}
		resp, err := mw.RoundTrip(req)
		assert.NotNil(t, resp)
		assert.NoError(t, err)
	})

	t.Run("nil request", func(t *testing.T) {
		mw := NewHttpRoundTripper()
		assert.NotNil(t, mw)

		resp, err := mw.RoundTrip(nil)
		assert.Nil(t, resp)
		assert.Error(t, err)
	})

	t.Run("empty request body", func(t *testing.T) {
		transport := newMockHTTPRoundTripper()

		mw := NewHttpRoundTripper(WithBaseRoundTripper(transport))
		assert.NotNil(t, mw)

		host, _ := url.Parse("https://localhost")
		req := &http.Request{
			URL:    host,
			Header: map[string][]string{},
		}
		resp, err := mw.RoundTrip(req)
		assert.NotNil(t, resp)
		assert.NoError(t, err)
	})

	t.Run("non-empty request body", func(t *testing.T) {
		transport := newMockHTTPRoundTripper()

		mw := NewHttpRoundTripper(WithBaseRoundTripper(transport))
		assert.NotNil(t, mw)

		host, _ := url.Parse("https://localhost")
		req := &http.Request{
			URL:    host,
			Header: map[string][]string{},
			Body:   io.NopCloser(bytes.NewBufferString(`{"foo":"bar"}`)),
		}
		resp, err := mw.RoundTrip(req)
		assert.NotNil(t, resp)
		assert.NoError(t, err)
	})

	t.Run("non-empty request body non-json", func(t *testing.T) {
		transport := newMockHTTPRoundTripper()

		mw := NewHttpRoundTripper(WithBaseRoundTripper(transport))
		assert.NotNil(t, mw)

		host, _ := url.Parse("https://localhost")
		req := &http.Request{
			URL:    host,
			Header: map[string][]string{},
			Body:   io.NopCloser(bytes.NewBufferString(`<html></html>`)),
		}
		resp, err := mw.RoundTrip(req)
		assert.NotNil(t, resp)
		assert.NoError(t, err)
	})

	t.Run("non-empty request body but error read", func(t *testing.T) {
		transport := newMockHTTPRoundTripper()

		mw := NewHttpRoundTripper(WithBaseRoundTripper(transport))
		assert.NotNil(t, mw)

		host, _ := url.Parse("https://localhost")
		req := &http.Request{
			URL:    host,
			Header: map[string][]string{},
			Body:   io.NopCloser(newBuf(fmt.Errorf("mock error buffer"))),
		}

		resp, err := mw.RoundTrip(req)
		assert.NotNil(t, resp)
		assert.NoError(t, err)
	})

	t.Run("non-empty request body but error close", func(t *testing.T) {
		transport := newMockHTTPRoundTripper()

		mw := NewHttpRoundTripper(WithBaseRoundTripper(transport))
		assert.NotNil(t, mw)

		host, _ := url.Parse("https://localhost")
		req := &http.Request{
			URL:    host,
			Header: map[string][]string{},
			Body:   newCloser(bytes.NewBufferString(`{"foo":"bar"}`), fmt.Errorf("mock close error")),
		}

		resp, err := mw.RoundTrip(req)
		require.NotNil(t, resp)
		require.NoError(t, err)
	})

	t.Run("not-nil response with empty resp body", func(t *testing.T) {
		transport := newMockHTTPRoundTripper()
		transport.CallRoundTrip = func(request *http.Request) (*http.Response, error) {
			return &http.Response{}, nil
		}

		mw := NewHttpRoundTripper(WithBaseRoundTripper(transport))
		assert.NotNil(t, mw)

		host, _ := url.Parse("https://localhost")
		req := &http.Request{
			URL:    host,
			Header: map[string][]string{},
			Body:   io.NopCloser(bytes.NewBufferString(`{"foo":"bar"}`)),
		}
		resp, err := mw.RoundTrip(req)
		assert.NotNil(t, resp)
		assert.NoError(t, err)
	})

	t.Run("not-nil response non-empty resp body but failed to read", func(t *testing.T) {
		transport := newMockHTTPRoundTripper()

		transport.CallRoundTrip = func(request *http.Request) (*http.Response, error) {
			return &http.Response{
				Body: io.NopCloser(newBuf(fmt.Errorf("mock error buffer"))),
			}, nil
		}

		mw := NewHttpRoundTripper(WithBaseRoundTripper(transport))
		assert.NotNil(t, mw)

		host, _ := url.Parse("https://localhost")
		req := &http.Request{
			URL:    host,
			Header: map[string][]string{},
			Body:   io.NopCloser(bytes.NewBufferString(`{"foo":"bar"}`)),
		}
		resp, err := mw.RoundTrip(req)
		assert.NotNil(t, resp)
		assert.NoError(t, err)
	})

	t.Run("not-nil response non-empty resp body", func(t *testing.T) {
		transport := newMockHTTPRoundTripper()
		transport.CallRoundTrip = func(request *http.Request) (*http.Response, error) {
			return &http.Response{
				Body: io.NopCloser(bytes.NewBufferString(`{"FOO":"BAR"}`)),
			}, nil
		}

		mw := NewHttpRoundTripper(WithBaseRoundTripper(transport))
		assert.NotNil(t, mw)

		host, _ := url.Parse("https://localhost")
		req := &http.Request{
			URL:    host,
			Header: map[string][]string{},
			Body:   io.NopCloser(bytes.NewBufferString(`{"foo":"bar"}`)),
		}
		resp, err := mw.RoundTrip(req)
		assert.NotNil(t, resp)
		assert.NoError(t, err)
	})

	t.Run("not-nil response body fail on close", func(t *testing.T) {
		transport := newMockHTTPRoundTripper()
		transport.CallRoundTrip = func(request *http.Request) (*http.Response, error) {
			return &http.Response{
				Body: newCloser(bytes.NewBufferString(`{"FOO":"BAR"}`), fmt.Errorf("mock close error")),
			}, nil
		}

		mw := NewHttpRoundTripper(WithBaseRoundTripper(transport))
		assert.NotNil(t, mw)

		host, _ := url.Parse("https://localhost")
		req := &http.Request{
			URL:    host,
			Header: map[string][]string{},
			Body:   io.NopCloser(bytes.NewBufferString(`{"foo":"bar"}`)),
		}
		resp, err := mw.RoundTrip(req)
		assert.NotNil(t, resp)
		assert.NoError(t, err)
	})

	t.Run("not-nil response non-empty resp body with non-valid json", func(t *testing.T) {
		transport := newMockHTTPRoundTripper()
		transport.CallRoundTrip = func(request *http.Request) (*http.Response, error) {
			return &http.Response{
				Body: io.NopCloser(bytes.NewBufferString(`<FOO>BAR</FOO>`)),
			}, nil
		}

		mw := NewHttpRoundTripper(WithBaseRoundTripper(transport))
		assert.NotNil(t, mw)

		host, _ := url.Parse("https://localhost")
		req := &http.Request{
			URL:    host,
			Header: map[string][]string{},
			Body:   io.NopCloser(bytes.NewBufferString(`{"foo":"bar"}`)),
		}
		resp, err := mw.RoundTrip(req)
		assert.NotNil(t, resp)
		assert.NoError(t, err)
	})
}

type mockHTTPRoundTrip struct {
	Error         error
	CallRoundTrip func(request *http.Request) (*http.Response, error)
}

var _ http.RoundTripper = (*mockHTTPRoundTrip)(nil)

func newMockHTTPRoundTripper() *mockHTTPRoundTrip {
	return &mockHTTPRoundTrip{}
}

func (m *mockHTTPRoundTrip) RoundTrip(request *http.Request) (*http.Response, error) {
	if m.Error != nil {
		return nil, m.Error
	}

	if request == nil {
		return nil, fmt.Errorf("nil *http.Request")
	}

	// by default, non-nil error returned with empty http.Response
	if m.CallRoundTrip == nil {
		return &http.Response{}, nil
	}

	return m.CallRoundTrip(request)
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
