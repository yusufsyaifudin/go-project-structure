package httpclientmw

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/yusufsyaifudin/go-project-structure/pkg/ylog"
	"go.uber.org/multierr"
)

type Opt func(*roundTripper) error

func WithBaseRoundTripper(h http.RoundTripper) Opt {
	return func(tripper *roundTripper) error {
		if tripper == nil {
			tripper.base = http.DefaultTransport
			return nil
		}

		tripper.base = h
		return nil
	}
}

func WithMessage(msg string) Opt {
	return func(tripper *roundTripper) error {
		if msg == "" {
			return nil
		}

		tripper.msg = msg
		return nil
	}
}

func WithLogger(logger ylog.Logger) Opt {
	return func(tripper *roundTripper) error {
		if logger == nil {
			tripper.logger = &ylog.Noop{}
			return nil
		}

		tripper.logger = logger
		return nil
	}
}

type roundTripper struct {
	base   http.RoundTripper
	msg    string
	logger ylog.Logger
}

var _ http.RoundTripper = (*roundTripper)(nil)

func NewHttpRoundTripper(opts ...Opt) http.RoundTripper {
	instance := &roundTripper{
		base:   http.DefaultTransport,
		msg:    "request logger",
		logger: &ylog.Noop{},
	}

	for _, opt := range opts {
		err := opt(instance)
		if err != nil {
			panic(err)
		}
	}

	return instance
}

func (r *roundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	t0 := time.Now()

	var (
		respOriginal *http.Response // final response
		errCum       error          // final error

		reqCtx          = req.Context()
		reqBodyBuf      = &bytes.Buffer{}
		reqBodyErr      error
		reqBodyCaptured interface{}
	)

	if req != nil && req.Body != nil {
		_, reqBodyErr = io.Copy(reqBodyBuf, req.Body)
		if reqBodyErr != nil {
			errCum = multierr.Append(errCum, fmt.Errorf("error copy request body: %w", reqBodyErr))
			reqBodyBuf = &bytes.Buffer{}
		}

		if _err := req.Body.Close(); _err != nil {
			errCum = multierr.Append(errCum, fmt.Errorf("error closing request body: %w", _err))
		}

		req.Body = io.NopCloser(reqBodyBuf)
	}

	// use json.Unmarshal instead of json.NewDecoder to make sure we can re-read the buffer
	if _err := json.Unmarshal(reqBodyBuf.Bytes(), &reqBodyCaptured); _err != nil && reqBodyBuf.Len() > 0 {
		reqBodyCaptured = reqBodyBuf.String()
	}

	var roundTripErr error
	respOriginal, roundTripErr = r.base.RoundTrip(req)
	if roundTripErr != nil {
		errCum = multierr.Append(errCum, fmt.Errorf("error doing actual request: %w", roundTripErr))
	}

	var (
		respBodyCaptured interface{}
		respBodyBuf      = &bytes.Buffer{}
		respErrBody      error
	)

	if respOriginal != nil && respOriginal.Body != nil {
		_, respErrBody = io.Copy(respBodyBuf, respOriginal.Body)
		if respErrBody != nil {
			errCum = multierr.Append(errCum, fmt.Errorf("error copy response body: %w", respErrBody))
			respBodyBuf = &bytes.Buffer{}
		}

		if _err := respOriginal.Body.Close(); _err != nil {
			errCum = multierr.Append(errCum, fmt.Errorf("error closing response body: %w", _err))
		}

		respOriginal.Body = io.NopCloser(respBodyBuf)
	}

	// use json.Unmarshal instead of json.NewDecoder to make sure we can re-read the buffer
	if _err := json.Unmarshal(respBodyBuf.Bytes(), &respBodyCaptured); _err != nil && respBodyBuf.Len() > 0 {
		respBodyCaptured = respBodyBuf.String()
	}

	accessLog := ylog.AccessLogData{
		ElapsedTime: time.Since(t0).Nanoseconds(),
	}

	// append to map only when the http.Request is not nil
	if req != nil {
		accessLog.Method = req.Method
		accessLog.Host = req.URL.Host
		accessLog.Path = req.URL.EscapedPath()
		accessLog.Request = &ylog.HTTPData{
			Header: toSimpleMap(req.Header),
			Body:   reqBodyCaptured,
		}
	}

	// append to map only when the http.Response is not nil
	if respOriginal != nil {
		accessLog.Response = &ylog.HTTPData{
			StatusCode: respOriginal.StatusCode,
			Header:     toSimpleMap(respOriginal.Header),
			Body:       respBodyCaptured,
		}
	}

	// append error if any
	if errCum != nil {
		accessLog.Error = errCum.Error()
	}

	r.logger.Access(reqCtx, r.msg, accessLog)

	return respOriginal, roundTripErr
}

func toSimpleMap(h http.Header) map[string]string {
	out := map[string]string{}
	for k, v := range h {
		out[k] = strings.Join(v, " ")
	}

	return out
}
