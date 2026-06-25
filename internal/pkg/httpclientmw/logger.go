package httpclientmw

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
)

const (
	instrumentationName = "github.com/yusufsyaifudin/go-project-structure/internal/pkg/httpclientmw"
)

// OutgoingLog holds log data for an outgoing HTTP request or response.
type OutgoingLog struct {
	Method      string            `json:"method,omitempty"`
	Host        string            `json:"host,omitempty"`
	Path        string            `json:"path,omitempty"`
	StatusCode  int               `json:"statusCode,omitempty"`
	Header      map[string]string `json:"header,omitempty"`
	Body        any               `json:"body,omitempty"`
	BodyLen     int64             `json:"bodyLen,omitempty"`
	Error       string            `json:"error,omitempty"`
	ElapsedTime int64             `json:"elapsedTime,omitempty"`
}

type Opt func(*roundTripper) error

// WithBaseRoundTripper replace base http.RoundTripper with new one.
func WithBaseRoundTripper(h http.RoundTripper) Opt {
	return func(tripper *roundTripper) error {
		if h == nil {
			tripper.base = http.DefaultTransport
			return nil
		}

		tripper.base = h
		return nil
	}
}

// WithMessage replaces the message on the log.
func WithMessage(msg string) Opt {
	return func(tripper *roundTripper) error {
		if msg == "" {
			return nil
		}

		tripper.msg = msg
		return nil
	}
}

// WithLogger set logger instance for this http.Client slog.
func WithLogger(logger *slog.Logger) Opt {
	return func(tripper *roundTripper) error {
		if logger == nil {
			tripper.logger = slog.Default()
			return nil
		}

		tripper.logger = logger
		return nil
	}
}

// WithTracer set tracerProvider instance to add span.
func WithTracer(t trace.TracerProvider) Opt {
	return func(tripper *roundTripper) error {
		if t == nil {
			tripper.tracerProvider = noop.NewTracerProvider()
			return nil
		}

		tripper.tracerProvider = t
		return nil
	}
}

// roundTripper hold an implementation of http.RoundTripper
type roundTripper struct {
	base           http.RoundTripper
	msg            string
	logger         *slog.Logger
	tracerProvider trace.TracerProvider
	tracer         trace.Tracer
}

var _ http.RoundTripper = (*roundTripper)(nil)

func newTracer(tp trace.TracerProvider) trace.Tracer {
	return tp.Tracer(instrumentationName)
}

// NewHttpRoundTripper return http.RoundTripper to be used as "middleware" in http.Client
// to log any outgoing http request.
func NewHttpRoundTripper(opts ...Opt) http.RoundTripper {
	noopTracer := noop.NewTracerProvider()

	instance := &roundTripper{
		base:           http.DefaultTransport,
		msg:            "request logger",
		logger:         slog.Default(),
		tracerProvider: noopTracer,
		tracer:         newTracer(noopTracer),
	}

	for _, opt := range opts {
		err := opt(instance)
		if err != nil {
			panic(err)
		}
	}

	instance.applyConfig()

	return instance
}

func (r *roundTripper) applyConfig() {
	r.tracer = newTracer(r.tracerProvider)
}

// RoundTrip do a http.RoundTrip and log the request/response body.
func (r *roundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	if req == nil {
		return nil, fmt.Errorf("http: nil Request")
	}

	t0 := time.Now()
	ctx := req.Context()

	reqURL := req.URL
	if reqURL == nil {
		reqURL = &url.URL{}
	}

	ctx, span := r.tracer.Start(ctx, fmt.Sprintf("HTTP %s %s", req.Method, reqURL.EscapedPath()))
	defer span.End()

	// Capture outgoing request body, then restore it so the base transport can read it.
	var (
		reqBodyBuf = &bytes.Buffer{}
		reqBodyLen int64
		reqErrCum  error
	)

	if req.Body != nil {
		if n, err := io.Copy(reqBodyBuf, req.Body); err != nil {
			reqErrCum = errors.Join(reqErrCum, fmt.Errorf("copy request body: %w", err))
			reqBodyBuf = &bytes.Buffer{}
		} else {
			reqBodyLen = n
		}

		if err := req.Body.Close(); err != nil {
			reqErrCum = errors.Join(reqErrCum, fmt.Errorf("close request body: %w", err))
		}

		req.Body = io.NopCloser(bytes.NewReader(reqBodyBuf.Bytes()))
	}

	var reqBodyCaptured any
	if reqBodyBuf.Len() > 0 {
		if err := json.Unmarshal(reqBodyBuf.Bytes(), &reqBodyCaptured); err != nil {
			reqErrCum = errors.Join(reqErrCum, fmt.Errorf("unmarshal request body: %w", err))
			reqBodyCaptured = reqBodyBuf.String()
		}
	}

	reqLog := OutgoingLog{
		Method:  req.Method,
		Host:    req.Host,
		Path:    reqURL.Path,
		Header:  toSimpleMap(req.Header),
		Body:    reqBodyCaptured,
		BodyLen: reqBodyLen,
	}
	if reqErrCum != nil {
		reqLog.Error = reqErrCum.Error()
	}

	r.logger.InfoContext(ctx, r.msg, slog.Any("request", reqLog))

	resp, roundTripErr := r.base.RoundTrip(req.WithContext(ctx))
	if roundTripErr != nil {
		return resp, roundTripErr
	}

	if resp == nil {
		return nil, nil
	}

	// Capture response body, then restore it so the caller can read it.
	var (
		respBodyBuf = &bytes.Buffer{}
		respBodyLen int64
		respErrCum  error
	)

	if resp.Body != nil {
		if n, err := io.Copy(respBodyBuf, resp.Body); err != nil {
			respErrCum = errors.Join(respErrCum, fmt.Errorf("copy response body: %w", err))
			respBodyBuf = &bytes.Buffer{}
		} else {
			respBodyLen = n
		}

		if err := resp.Body.Close(); err != nil {
			respErrCum = errors.Join(respErrCum, fmt.Errorf("close response body: %w", err))
		}

		resp.Body = io.NopCloser(bytes.NewReader(respBodyBuf.Bytes()))
	}

	var respBodyCaptured any
	if respBodyBuf.Len() > 0 {
		if err := json.Unmarshal(respBodyBuf.Bytes(), &respBodyCaptured); err != nil {
			respErrCum = errors.Join(respErrCum, fmt.Errorf("unmarshal response body: %w", err))
			respBodyCaptured = respBodyBuf.String()
		}
	}

	respLog := OutgoingLog{
		Method:      req.Method,
		Host:        req.Host,
		Path:        reqURL.Path,
		StatusCode:  resp.StatusCode,
		Header:      toSimpleMap(resp.Header),
		Body:        respBodyCaptured,
		BodyLen:     respBodyLen,
		ElapsedTime: time.Since(t0).Milliseconds(),
	}
	if respErrCum != nil {
		respLog.Error = respErrCum.Error()
	}

	r.logger.InfoContext(ctx, r.msg, slog.Any("response", respLog))

	return resp, nil
}

// toSimpleMap converts http.Header which as array of string as value to simple string.
func toSimpleMap(h http.Header) map[string]string {
	out := map[string]string{}
	for k, v := range h {
		out[k] = strings.Join(v, " ")
	}

	return out
}
