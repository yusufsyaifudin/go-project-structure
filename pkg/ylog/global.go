package ylog

import (
	"context"
	"os"
	"strings"
	"sync"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var logMut sync.RWMutex
var globalLogger Logger = &Noop{}

func GetGlobalLogger() Logger {
	logMut.RLock()
	defer logMut.RUnlock()
	return globalLogger
}

func SetGlobalLogger(log Logger) {
	logMut.Lock()
	defer logMut.Unlock()
	globalLogger = log
}

func Debug(ctx context.Context, msg string, fields ...KeyValue) {
	GetGlobalLogger().Debug(ctx, msg, fields...)
}

func Info(ctx context.Context, msg string, fields ...KeyValue) {
	GetGlobalLogger().Info(ctx, msg, fields...)
}

func Warn(ctx context.Context, msg string, fields ...KeyValue) {
	GetGlobalLogger().Warn(ctx, msg, fields...)
}

func Error(ctx context.Context, msg string, fields ...KeyValue) {
	GetGlobalLogger().Error(ctx, msg, fields...)
}

func Panic(ctx context.Context, msg string, fields ...KeyValue) {
	GetGlobalLogger().Panic(ctx, msg, fields...)
}

func Fatal(ctx context.Context, msg string, fields ...KeyValue) {
	GetGlobalLogger().Fatal(ctx, msg, fields...)
}

func Access(ctx context.Context, msg string, data AccessLogData) {
	GetGlobalLogger().Access(ctx, msg, data)
}

func SetupDefaultGlobalLogger(level string) {
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
	SetGlobalLogger(NewZap(log))
}
