package httpclientmw

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"go.opentelemetry.io/collector/pdata/ptrace"

	"github.com/yusufsyaifudin/go-project-structure/pkg/ylog"
	"go.uber.org/multierr"
)

type OptLogMw func(*loggerRoundTripper) error

// LogMwWithBaseRoundTripper replace base http.RoundTripper with new one.
func LogMwWithBaseRoundTripper(h http.RoundTripper) OptLogMw {
	return func(tripper *loggerRoundTripper) error {
		if h == nil {
			tripper.base = http.DefaultTransport
			return nil
		}

		tripper.base = h
		return nil
	}
}

// LogMwWithMessage replaces the message on the log.
func LogMwWithMessage(msg string) OptLogMw {
	return func(tripper *loggerRoundTripper) error {
		if msg == "" {
			return nil
		}

		tripper.msg = msg
		return nil
	}
}

// LogMwWithLogger set logger instance for this http.Client logger.
func LogMwWithLogger(logger ylog.Logger) OptLogMw {
	return func(tripper *loggerRoundTripper) error {
		if logger == nil {
			tripper.logger = ylog.NewNoop()
			return nil
		}

		tripper.logger = logger
		return nil
	}
}

// loggerRoundTripper hold an implementation of http.RoundTripper
type loggerRoundTripper struct {
	base   http.RoundTripper
	msg    string
	logger ylog.Logger
}

var _ http.RoundTripper = (*loggerRoundTripper)(nil)

// NewLogMw return http.RoundTripper to be used as "middleware" in http.Client
// to log any outgoing http request.
func NewLogMw(opts ...OptLogMw) http.RoundTripper {
	instance := &loggerRoundTripper{
		base:   http.DefaultTransport,
		msg:    "outgoing request logger",
		logger: ylog.NewNoop(),
	}

	for _, opt := range opts {
		err := opt(instance)
		if err != nil {
			panic(err)
		}
	}

	return instance
}

// RoundTrip do a http.RoundTrip and log the request/response body.
func (r *loggerRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	t0 := time.Now()

	ctx := context.Background()
	if req != nil && req.Context() != nil {
		ctx = req.Context()
	}

	var (
		errCum error // final error

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

	// TODO: this is to log the request payload sent to OTLP protocol using protobuf
	var tracesUnmarshaler = &ptrace.ProtoUnmarshaler{}
	_, _ = tracesUnmarshaler.UnmarshalTraces(reqBodyBuf.Bytes())

	var (
		respOriginal *http.Response // round trip response
		roundTripErr error          // round trip error
	)

	if req != nil {
		respOriginal, roundTripErr = r.base.RoundTrip(req)
	} else {
		roundTripErr = fmt.Errorf("cannot do round-tripper request because *http.Request is nil")
	}

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
	_errMarshalRespBody := json.Unmarshal(respBodyBuf.Bytes(), &respBodyCaptured)
	if _errMarshalRespBody != nil && respBodyBuf.Len() > 0 {
		respBodyCaptured = respBodyBuf.String()
	}

	accessLog := ylog.AccessLogData{
		ElapsedTime: time.Since(t0).Nanoseconds(),
	}

	// append to map only when the http.Request is not nil
	if req != nil {
		accessLog.Method = req.Method
		accessLog.Request = &ylog.HTTPData{
			Header: toSimpleMap(req.Header),
			Body:   reqBodyCaptured,
		}
	}

	if req != nil && req.URL != nil {
		accessLog.Host = req.URL.Host
		accessLog.Path = req.URL.EscapedPath()
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

	r.logger.Access(ctx, r.msg, accessLog)

	return respOriginal, roundTripErr
}
