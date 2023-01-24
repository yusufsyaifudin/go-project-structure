package ylog

import (
	"context"

	"go.uber.org/zap"
)

type Zap struct {
	writer *zap.Logger
}

var _ Logger = (*Zap)(nil)

func NewZap(zapLogger *zap.Logger) *Zap {
	return &Zap{writer: zapLogger}
}

func (z *Zap) Debug(ctx context.Context, msg string, fields ...KeyValue) {
	z.writer.Debug(msg, localFieldZapFields(ctx, typeSys, fields)...)
}

func (z *Zap) Info(ctx context.Context, msg string, fields ...KeyValue) {
	z.writer.Info(msg, localFieldZapFields(ctx, typeSys, fields)...)
}

func (z *Zap) Warn(ctx context.Context, msg string, fields ...KeyValue) {
	z.writer.Warn(msg, localFieldZapFields(ctx, typeSys, fields)...)
}

func (z *Zap) Error(ctx context.Context, msg string, fields ...KeyValue) {
	z.writer.Error(msg, localFieldZapFields(ctx, typeSys, fields)...)
}

func (z *Zap) Panic(ctx context.Context, msg string, fields ...KeyValue) {
	z.writer.Panic(msg, localFieldZapFields(ctx, typeSys, fields)...)
}

func (z *Zap) Fatal(ctx context.Context, msg string, fields ...KeyValue) {
	z.writer.Fatal(msg, localFieldZapFields(ctx, typeSys, fields)...)
}

func (z *Zap) Access(ctx context.Context, msg string, data AccessLogData) {
	z.writer.Info(msg, localFieldZapFields(ctx, typeAccessLog, []KeyValue{KV("data", data)})...)
}

func localFieldZapFields(ctx context.Context, logType logType, fields []KeyValue) []zap.Field {
	zapFields := make([]zap.Field, 0)
	zapFields = append(zapFields, zap.String("log_type", string(logType)))

	for _, field := range fields {
		zapFields = append(zapFields, zap.Any(field.Key(), field.Value()))
	}

	return zapFields
}
