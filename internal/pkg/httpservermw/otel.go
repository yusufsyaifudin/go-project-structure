package httpservermw

import (
	"fmt"
	"net/http"

	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

type OtelOpt func(*otelMw) error

// OtelMwWithTracer use explicit OpenTelemetry TracerProvider.
// This to make each middleware has scoped OpenTelemetry provider.
// Useful when we set up multiple server in one binary that has different server.
func OtelMwWithTracer(tracer trace.Tracer) OtelOpt {
	return func(mw *otelMw) error {
		if tracer == nil {
			return nil
		}

		mw.tracer = tracer
		return nil
	}
}

type otelMw struct {
	tracer trace.Tracer
}

// OpenTelemetryMiddleware is a middleware that add OpenTelemetry context into Request and Response header.
func OpenTelemetryMiddleware(next http.Handler, opts ...OtelOpt) http.Handler {
	l := &otelMw{
		tracer: trace.NewNoopTracerProvider().Tracer("noop_tracer"),
	}

	for _, opt := range opts {
		err := opt(l)
		if err != nil {
			panic(err)
		}
	}

	fn := func(w http.ResponseWriter, req *http.Request) {

		reqCtx := req.Context()
		spanName := fmt.Sprintf("%s %s", req.Method, req.URL.EscapedPath())

		propagator := propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{})

		// extract from request header
		reqCtx = propagator.Extract(reqCtx, propagation.HeaderCarrier(req.Header))

		// create new span for this request
		newReqCtx, span := l.tracer.Start(reqCtx, spanName)
		defer span.End()

		// Inject OpenTelemetry to response header
		propagator.Inject(newReqCtx, propagation.HeaderCarrier(w.Header()))

		next.ServeHTTP(w, req.WithContext(newReqCtx))

	}

	return http.HandlerFunc(fn)
}
