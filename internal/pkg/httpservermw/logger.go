package httpservermw

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"time"

	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/multierr"

	"github.com/yusufsyaifudin/go-project-structure/pkg/ylog"
)

const (
	instrumentationName = "github.com/yusufsyaifudin/go-project-structure/internal/pkg/httpservermw"
)

type LoggerOpt func(*LogMiddleware) error

// Filter is a predicate used to determine whether a given http.request should
// be logged. A Filter must return true if the request should be traced.
// Don't change any value in request.
// If Filter return false, then it should be skipped.
// Otherwise, if return true, it will be included in log.
type Filter func(*http.Request) bool

// LogMwWithMessage change message on logger.
func LogMwWithMessage(msg string) LoggerOpt {
	return func(tripper *LogMiddleware) error {
		tripper.msg = msg
		return nil
	}
}

// LogMwWithLogger set logger
func LogMwWithLogger(logger ylog.Logger) LoggerOpt {
	return func(tripper *LogMiddleware) error {
		if logger == nil {
			tripper.logger = ylog.NewNoop()
			return nil
		}

		tripper.logger = logger
		return nil
	}
}

// LogMwWithTracer set OpenTelemetry tracer provider instance to add span.
func LogMwWithTracer(t trace.TracerProvider) LoggerOpt {
	return func(tripper *LogMiddleware) error {
		if t == nil {
			tripper.tracerProvider = trace.NewNoopTracerProvider()
			return nil
		}

		tripper.tracerProvider = t
		return nil
	}
}

// LogMwWithFilter filter out what route that don't want to be logged
func LogMwWithFilter(f Filter) LoggerOpt {
	return func(tripper *LogMiddleware) error {
		tripper.filter = f
		return nil
	}
}

type LogMiddleware struct {
	msg            string
	logger         ylog.Logger
	tracerProvider trace.TracerProvider
	filter         Filter
}

// LoggingMiddleware is a middleware that logs incoming requests
func LoggingMiddleware(next http.Handler, opts ...LoggerOpt) http.Handler {
	l := &LogMiddleware{
		msg:            "request logger",
		logger:         ylog.NewNoop(),
		tracerProvider: trace.NewNoopTracerProvider(),
	}

	for _, opt := range opts {
		err := opt(l)
		if err != nil {
			panic(err)
		}
	}

	tracer := l.tracerProvider.Tracer(instrumentationName)

	propagator := propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{})

	fn := func(w http.ResponseWriter, req *http.Request) {
		if l.filter != nil && req != nil {
			if !l.filter(req) {
				next.ServeHTTP(w, req)
				return
			}
		}

		t0 := time.Now()

		parentCtx := context.Background()
		if req != nil && req.Context() != nil {
			parentCtx = req.Context()

			// extract from request header if any span pass through from request context via header
			parentCtx = propagator.Extract(parentCtx, propagation.HeaderCarrier(req.Header))
		}

		// parent span for this logging middleware
		spanName := "[Log MW]"
		if req != nil {
			spanName = fmt.Sprintf("%s %s [Log MW]", req.Method, req.URL.EscapedPath())
		}

		var parentSpan trace.Span
		parentCtx, parentSpan = tracer.Start(parentCtx, spanName)
		defer parentSpan.End()

		// create child span for this request and pass it to the actual handler span.
		// this span will end after this middleware function call done.
		reqCtx, reqSpan := tracer.Start(parentCtx, "Log middleware")
		defer reqSpan.End()

		captReqCtx, captReqSpan := tracer.Start(reqCtx, "Capture request")
		var (
			errCum error // final cumulative error

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

		accessLog := ylog.AccessLogData{}

		// append to map only when the http.Request is not nil
		if req != nil {
			accessLog.Method = req.Method
			accessLog.Request = &ylog.HTTPData{
				Header: HttpHeaderToSimpleMap(req.Header),
				Body:   reqBodyCaptured,
			}
		}

		if req != nil && req.URL != nil {
			accessLog.Host = req.URL.Host
			accessLog.Path = req.URL.String()
		}

		// ending the capture request span right before we do actual ServeHTTP.
		// meaning that *this* span only to capture how many times needed to get the request body
		captReqSpan.End()

		// Pass the request to the next handler
		respRec := httptest.NewRecorder()

		// use the child request span context, so the handler will continue the child span for this request context
		next.ServeHTTP(respRec, req.WithContext(captReqCtx))

		// create new span for this response span.
		// response span MUST continue from parent span, because it's process is scoped in this middleware only.
		var respSpan trace.Span
		_, respSpan = tracer.Start(reqCtx, "Capture response")
		defer respSpan.End() // done response span

		var (
			respBodyCaptured interface{}
			respBodyBuf      = &bytes.Buffer{}
		)

		if respRec.Result() != nil && respRec.Body != nil {
			respBodyBuf = respRec.Body
		}

		// use json.Unmarshal instead of json.NewDecoder to make sure we can re-read the buffer
		if _err := json.Unmarshal(respBodyBuf.Bytes(), &respBodyCaptured); _err != nil && respBodyBuf.Len() > 0 {
			respBodyCaptured = respBodyBuf.String()
		}

		// inject Traceparent to response recorder header,
		// next it will write to actual writer response header
		propagator.Inject(parentCtx, propagation.HeaderCarrier(respRec.Header()))

		// append to map only when the http.Response is not nil
		httpStatusCode := http.StatusInternalServerError
		if respRec.Result() != nil {
			httpStatusCode = respRec.Result().StatusCode
			accessLog.Response = &ylog.HTTPData{
				StatusCode: respRec.Result().StatusCode,
				Header:     HttpHeaderToSimpleMap(respRec.Header()),
				Body:       respBodyCaptured,
			}
		}

		// write to actual response writer
		for k, v := range respRec.Header() {
			w.Header().Set(k, strings.Join(v, " "))
		}

		w.WriteHeader(httpStatusCode)

		// write response body
		if _, _err := w.Write(respBodyBuf.Bytes()); _err != nil {
			errCum = multierr.Append(errCum, fmt.Errorf("failed to write to actual response writer: %w", _err))
		}

		// append error if any
		if errCum != nil {
			accessLog.Error = errCum.Error()

			respSpan.RecordError(errCum)
			respSpan.SetStatus(codes.Error, "some error occurred during capturing log")
		}

		accessLog.ElapsedTime = time.Since(t0).Nanoseconds()

		// write log here
		l.logger.Access(parentCtx, l.msg, accessLog)

	}

	return http.HandlerFunc(fn)
}

// HttpHeaderToSimpleMap converts http.Header which as array of string as value to simple string.
func HttpHeaderToSimpleMap(h http.Header) map[string]string {
	out := map[string]string{}
	for k, v := range h {
		out[k] = strings.Join(v, " ")
	}

	return out
}
