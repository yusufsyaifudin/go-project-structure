package httpclientmw

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/multierr"

	"github.com/yusufsyaifudin/go-project-structure/pkg/ylog"
)

const (
	instrumentationName = "github.com/yusufsyaifudin/go-project-structure/internal/pkg/httpclientmw"
)

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

// WithLogger set logger instance for this http.Client logger.
func WithLogger(logger ylog.Logger) Opt {
	return func(tripper *roundTripper) error {
		if logger == nil {
			tripper.logger = ylog.NewNoop()
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
			tripper.tracerProvider = trace.NewNoopTracerProvider()
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
	logger         ylog.Logger
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
	noopTracer := trace.NewNoopTracerProvider()

	instance := &roundTripper{
		base:           http.DefaultTransport,
		msg:            "request logger",
		logger:         ylog.NewNoop(),
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
	t0 := time.Now()

	propagator := propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{})

	// create new span for this request
	var parentCtx = context.Background()
	if req != nil && req.Context() != nil {
		parentCtx = req.Context()
	}

	var parentSpan trace.Span
	parentCtx, parentSpan = r.tracer.Start(parentCtx, "Capture http.RoundTrip request")
	defer parentSpan.End() // ending the request span

	if req != nil {
		// extract from parent request header
		parentCtx = propagator.Extract(parentCtx, propagation.HeaderCarrier(req.Header))

		// Inject header needed to pass through the OpenTelemetry span context.
		propagator.Inject(parentCtx, propagation.HeaderCarrier(req.Header))
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

	var (
		respOriginal *http.Response // round trip response
		roundTripErr error          // round trip error
	)

	if req != nil {
		respOriginal, roundTripErr = r.base.RoundTrip(req.WithContext(parentCtx))
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
	if _err := json.Unmarshal(respBodyBuf.Bytes(), &respBodyCaptured); _err != nil && respBodyBuf.Len() > 0 {
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

		parentSpan.RecordError(errCum)
		parentSpan.SetStatus(codes.Error, "some error occurred during capturing log")
	}

	r.logger.Access(parentCtx, r.msg, accessLog)

	return respOriginal, roundTripErr
}

// toSimpleMap converts http.Header which as array of string as value to simple string.
func toSimpleMap(h http.Header) map[string]string {
	out := map[string]string{}
	for k, v := range h {
		out[k] = strings.Join(v, " ")
	}

	return out
}
