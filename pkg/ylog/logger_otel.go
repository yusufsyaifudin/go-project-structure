package ylog

import (
	"context"
	"go.opentelemetry.io/otel/trace"
	"log/slog"
)

type OpenTelemetryOption struct {
	ContextExtractor func(context.Context) []slog.Attr
}

type OpenTelemetry struct {
	next        slog.Handler
	opts        *OpenTelemetryOption
	enabledFunc func(ctx context.Context, level slog.Level, next func(context.Context, slog.Level) bool) bool
}

var _ slog.Handler = (*OpenTelemetry)(nil)

func NewOTEL(parent slog.Handler, opts *OpenTelemetryOption) slog.Handler {
	return &OpenTelemetry{
		next: parent,
		opts: opts,
		enabledFunc: func(ctx context.Context, level slog.Level, next func(context.Context, slog.Level) bool) bool {
			return next(ctx, level)
		},
	}
}

func (z *OpenTelemetry) Enabled(ctx context.Context, level slog.Level) bool {
	return z.enabledFunc(ctx, level, z.next.Enabled)
}

func (z *OpenTelemetry) Handle(ctx context.Context, record slog.Record) error {
	// From [OTEP0114](https://github.com/open-telemetry/oteps/pull/114)
	// https://github.com/open-telemetry/opentelemetry-specification/blob/v1.18.0/specification/logs/README.md?plain=1#L474-L526
	spanCtx := trace.SpanContextFromContext(ctx)
	record.AddAttrs(
		slog.String("trace_id", spanCtx.TraceID().String()),
		slog.String("span_id", spanCtx.SpanID().String()),
	)

	if z.opts != nil && z.opts.ContextExtractor != nil {
		for _, ctxAttr := range z.opts.ContextExtractor(ctx) {
			switch ctxAttr.Key {
			case "trace_id", "span_id":
				continue
			}
			
			record.AddAttrs(ctxAttr)
		}
	}

	return z.next.Handle(ctx, record)
}

// WithAttrs is called when user call slog.With(attrs...)
// or when slog.With(attrs..).WithGroup(name).With(attrs...)
//
// If user already call WithGroup, the appended valued will append to this group name child's.
// We need to call AttributeFormatterFunc here before appended to the OpenTelemetry logger buffer.
func (z *OpenTelemetry) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &OpenTelemetry{
		next:        z.next.WithAttrs(attrs),
		enabledFunc: z.enabledFunc,
	}
}

// WithGroup is called when user call slog.WithGroup(name).
// This open a new namespae on OpenTelemetry
func (z *OpenTelemetry) WithGroup(name string) slog.Handler {
	// https://cs.opensource.google/go/x/exp/+/46b07846:slog/handler.go;l=247
	if name == "" {
		return z
	}

	return &OpenTelemetry{
		next:        z.next.WithGroup(name),
		opts:        z.opts,
		enabledFunc: z.enabledFunc,
	}
}
