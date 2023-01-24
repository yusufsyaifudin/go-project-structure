package ylog

import (
	"os"
	"strings"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func SetupZapLogger(level string) Logger {
	zapLevel := zapcore.DebugLevel
	switch strings.ToUpper(level) {
	case "DEBUG":
		zapLevel = zapcore.DebugLevel
	case "INFO":
		zapLevel = zapcore.InfoLevel
	case "WARN":
		zapLevel = zapcore.WarnLevel
	case "ERROR":
		zapLevel = zapcore.ErrorLevel
	case "PANIC":
		zapLevel = zapcore.PanicLevel
	case "FATAL":
		zapLevel = zapcore.FatalLevel
	}

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
		zapcore.NewMultiWriteSyncer(zapcore.AddSync(os.Stdout)), // pipe to multiple writer
		zapLevel,
	)

	log := zap.New(core)
	return NewZap(log)
}

type kv struct {
	k string
	v any
}

var _ KeyValue = (*kv)(nil)

// KV implements KeyValue interface that hold additional tag for the logger.
func KV(k string, v interface{}) KeyValue {
	return &kv{
		k: k,
		v: v,
	}
}

func (k *kv) Key() string {
	return k.k
}

func (k *kv) Value() any {
	return k.v
}
