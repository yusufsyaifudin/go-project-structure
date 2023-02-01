package ylog

import (
	"bytes"
	"fmt"
	"os"
	"strings"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// SetupZapLogger setup the Logger with Uber Zap logger with specific level.
// Default using DEBUG level if unknown level specified.
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

	log := zap.New(core,
		zap.ErrorOutput(zapcore.AddSync(&errOut{})),
	)
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

type errOut struct{}

// Write format Zap error as JSON.
func (e *errOut) Write(b []byte) (n int, err error) {
	buf := &bytes.Buffer{}
	defer buf.Reset()

	buf.WriteString(`{"level":"error","ts":"`)
	buf.WriteString(time.Now().Format(time.RFC3339Nano))
	buf.WriteString(`",`)
	buf.WriteString(`"msg":"zap error and cannot do some process",`)
	buf.WriteString(`"log_type":"sys",`)
	buf.WriteString(`"error":"`)
	buf.WriteString(strings.TrimSpace(string(b)))
	buf.WriteString(`"}`)
	buf.WriteString("\n")

	return fmt.Fprint(os.Stdout, buf.String())
}
