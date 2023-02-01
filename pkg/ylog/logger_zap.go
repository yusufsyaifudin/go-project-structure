package ylog

import (
	"context"

	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

type Zap struct {
	writer *zap.Logger
}

var _ Logger = (*Zap)(nil)

func NewZap(zapLogger *zap.Logger) *Zap {
	return &Zap{
		writer: zapLogger,
	}
}

func (z *Zap) Debug(ctx context.Context, msg string, fields ...KeyValue) {
	z.writer.Debug(msg, localFieldZapFields(ctx, fields)...)
}

func (z *Zap) Info(ctx context.Context, msg string, fields ...KeyValue) {
	z.writer.Info(msg, localFieldZapFields(ctx, fields)...)
}

func (z *Zap) Warn(ctx context.Context, msg string, fields ...KeyValue) {
	z.writer.Warn(msg, localFieldZapFields(ctx, fields)...)
}

func (z *Zap) Error(ctx context.Context, msg string, fields ...KeyValue) {
	z.writer.Error(msg, localFieldZapFields(ctx, fields)...)
}

func (z *Zap) Panic(ctx context.Context, msg string, fields ...KeyValue) {
	z.writer.Panic(msg, localFieldZapFields(ctx, fields)...)
}

func (z *Zap) Fatal(ctx context.Context, msg string, fields ...KeyValue) {
	z.writer.Fatal(msg, localFieldZapFields(ctx, fields)...)
}

func (z *Zap) Access(ctx context.Context, msg string, data AccessLogData) {
	z.writer.Info(msg, localFieldZapFields(ctx, []KeyValue{KV("data", data)})...)
}

func (z *Zap) WithStaticFields(fields ...KeyValue) Logger {
	zapFields := make([]zap.Field, 0)

	for _, field := range fields {
		zapFields = append(zapFields, zap.Any(field.Key(), field.Value()))
	}

	return &Zap{
		writer: z.writer.With(zapFields...),
	}
}

func localFieldZapFields(ctx context.Context, fields []KeyValue) []zap.Field {
	traceInfo := trace.SpanContextFromContext(ctx)

	zapFields := make([]zap.Field, 0)
	if traceInfo.HasTraceID() {
		zapFields = append(zapFields, zap.String("trace_id", traceInfo.TraceID().String()))
	}

	if traceInfo.HasTraceID() {
		zapFields = append(zapFields, zap.String("span_id", traceInfo.SpanID().String()))
	}

	for _, field := range fields {
		zapFields = append(zapFields, zap.Any(field.Key(), field.Value()))
	}

	return zapFields
}
