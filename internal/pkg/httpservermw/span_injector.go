package httpservermw

import (
	"fmt"
	"net/http"
	"net/url"

	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

// SpanInjectorMiddleware ensures every request carries a real span context (non-zero trace_id
// and span_id) even for routes where the main OpenTelemetry middleware (otelhttp) is filtered out.
//
// For routes where filter returns false, this middleware starts a non-exported span using a
// NeverSample tracer provider. The span is never sent to the tracing backend, but its TraceID
// and SpanID are real — so structured logs emitted inside the handler (e.g. slog.DebugContext)
// still carry meaningful trace context instead of all-zero IDs.
//
// This middleware must sit between otelhttp and the logging/handler layer:
//
//	... → otelhttp → SpanInjectorMiddleware → LoggingMiddleware → handler
func SpanInjectorMiddleware(next http.Handler, filter Filter) http.Handler {
	// NeverSample: the SDK generates real TraceID/SpanID but DROP decision means
	// the span is never recorded or exported to any backend.
	nsp := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.NeverSample()),
	)
	tracer := nsp.Tracer(instrumentationName)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Only act when the route is excluded from the main OTel middleware.
		// For included routes, otelhttp has already put a real exported span in the context.
		if r != nil && filter != nil && !filter(r) {
			reqURL := r.URL
			if reqURL == nil {
				reqURL = &url.URL{}
			}

			ctx, span := tracer.Start(r.Context(), fmt.Sprintf("%s %s", r.Method, reqURL.EscapedPath()))
			defer span.End()

			r = r.WithContext(ctx)
		}

		next.ServeHTTP(w, r)
	})
}
