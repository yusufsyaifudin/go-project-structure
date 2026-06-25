package httpservermw

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"go.opentelemetry.io/otel/propagation"
	semconv "go.opentelemetry.io/otel/semconv/v1.27.0"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
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

// LogMwWithLogger set logger
func LogMwWithLogger(logger *slog.Logger) LoggerOpt {
	return func(tripper *LogMiddleware) error {
		if logger == nil {
			tripper.logger = slog.Default()
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
			tripper.tracerProvider = noop.NewTracerProvider()
			return nil
		}

		tripper.tracerProvider = t
		return nil
	}
}

// LogMwWithTracerSpanStartOption add span start option to the span created by this middleware.
func LogMwWithTracerSpanStartOption(opts ...trace.SpanStartOption) LoggerOpt {
	return func(tripper *LogMiddleware) error {
		if opts == nil {
			tripper.tracerProvider = noop.NewTracerProvider()
			return nil
		}

		for _, opt := range opts {
			if opt == nil {
				continue
			}

			tripper.spanStartOptions = append(tripper.spanStartOptions, opt)
		}

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
	logger           *slog.Logger
	tracerProvider   trace.TracerProvider
	spanStartOptions []trace.SpanStartOption
	filter           Filter
}

// LoggingMiddleware is a middleware that logs incoming requests
func LoggingMiddleware(next http.Handler, opts ...LoggerOpt) http.Handler {
	l := &LogMiddleware{
		logger:           slog.Default(),
		tracerProvider:   noop.NewTracerProvider(),
		spanStartOptions: make([]trace.SpanStartOption, 0),
	}

	for _, opt := range opts {
		err := opt(l)
		if err != nil {
			panic(err)
		}
	}

	tracer := l.tracerProvider.Tracer(instrumentationName)
	l.spanStartOptions = append(l.spanStartOptions, trace.WithSpanKind(trace.SpanKindServer))

	propagator := propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{})

	fn := func(w http.ResponseWriter, req *http.Request) {
		t0 := time.Now()

		if l.filter != nil && req != nil {
			if !l.filter(req) {
				next.ServeHTTP(w, req)
				return
			}
		}

		parentCtx := context.Background()
		if req == nil {
			next.ServeHTTP(w, req)
			return
		}

		if req.Context() != nil {
			parentCtx = req.Context()

			// extract from request header if any span pass through from request context via header
			parentCtx = propagator.Extract(parentCtx, propagation.HeaderCarrier(req.Header))
		}

		// Always clone to mitigate issues.
		req = req.Clone(parentCtx)

		reqURL := req.URL
		if reqURL == nil {
			reqURL = &url.URL{}
		}

		// parent span for this logging middleware
		spanName := fmt.Sprintf("%s %s [Log MW]", req.Method, reqURL.EscapedPath())

		var parentSpan trace.Span
		parentCtx, parentSpan = tracer.Start(parentCtx, spanName, l.spanStartOptions...)
		parentSpan.SetAttributes(semconv.URLFull(reqURL.String()))
		defer parentSpan.End()

		// create child span for this request and pass it to the actual handler span.
		// this span will end after this middleware function call done.
		reqCtx, reqSpan := tracer.Start(parentCtx, "Log middleware")
		defer reqSpan.End()

		// append to map only when the http.Request is not nil
		reqCtx, reqSpan = tracer.Start(reqCtx, "Capture request")
		captReqCtx := captureRequest(reqCtx, &captureRequestOpt{
			T0:               t0,
			Request:          req,
			Tracer:           tracer,
			SpanStartOptions: l.spanStartOptions,
			Logger:           l.logger,
		})

		// ending the capture request span right before we do actual ServeHTTP.
		// meaning that *this* span only to capture how many times needed to get the request body
		reqCtx.Done()
		reqSpan.End()

		// create new span for this response span.
		// response span MUST continue from parent span, because it's process is scoped in this middleware only.
		var respCtx context.Context
		var respSpan trace.Span
		respCtx, respSpan = tracer.Start(reqCtx, "Capture response")
		defer func() {
			respCtx.Done()
			respSpan.End() // done response span
		}()

		// Pass the request to the next handler
		respRec := newResponseWriter(w)

		// inject Traceparent to response recorder header,
		// next it will write to actual writer response header
		propagator.Inject(captReqCtx, propagation.HeaderCarrier(respRec.Header()))

		// use the child request span context, so the handler will continue the child span for this request context
		next.ServeHTTP(respRec, req.WithContext(reqCtx))

		respBodyLen := int64(len(respRec.body))

		// Log or process the captured status code, headers, and body
		respLog := AccessLog{
			Method:      req.Method,
			Host:        req.Host,
			Path:        reqURL.Path,
			StatusCode:  respRec.statusCode,
			Header:      HttpHeaderToSimpleMap(respRec.headers),
			Body:        nil,
			BodyLen:     respBodyLen,
			QueryParams: nil,
			Error:       "",
			ElapsedTime: time.Since(t0).Milliseconds(),
		}

		var respBodyCaptured any
		var respBodyDecoderErr error
		if respBodyLen > 0 {
			respBodyDecoderErr = json.Unmarshal(respRec.body, &respBodyCaptured)
		}

		if respBodyDecoderErr != nil {
			respBodyCaptured = string(respRec.body) // Fallback with the real request body string
			respLog.Error = fmt.Sprintf("error unmarshal response body: %s", respBodyDecoderErr.Error())
		}

		respLog.Body = respBodyCaptured

		l.logger.InfoContext(respCtx, "capture incoming response payload", slog.Any("response", respLog))
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
