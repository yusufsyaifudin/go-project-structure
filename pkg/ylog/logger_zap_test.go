package ylog_test

import (
	"context"
	"io"
	"testing"

	"github.com/yusufsyaifudin/go-project-structure/pkg/ylog"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func BenchmarkNewZap(b *testing.B) {
	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(zapcore.EncoderConfig{
			TimeKey:        "ts",
			MessageKey:     "msg",
			EncodeDuration: zapcore.MillisDurationEncoder,
			EncodeTime:     zapcore.RFC3339NanoTimeEncoder,
			LineEnding:     zapcore.DefaultLineEnding,
			LevelKey:       "level",
			EncodeLevel:    zapcore.LowercaseLevelEncoder,
		}),
		zapcore.NewMultiWriteSyncer(zapcore.AddSync(io.Discard)), // pipe to multiple writer
		zapcore.DebugLevel,
	)
	zapLogger := zap.New(core)
	uniLogger := ylog.NewZap(zapLogger)

	ctx := context.Background()
	for i := 0; i < b.N; i++ {
		// zapLogger.Error("message", zap.Any("tracer", logger.Tracer{AppTraceID: "test"}))
		uniLogger.Error(ctx, "message")
	}

}
