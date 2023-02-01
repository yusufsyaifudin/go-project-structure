package ylog

import (
	"context"
	"encoding/json"
	"io"
	"strings"
)

type LoggerIOWriterOpt func(*LoggerIOWriter)

// LoggerIOWriterWithContext set context for logger.
func LoggerIOWriterWithContext(ctx context.Context) LoggerIOWriterOpt {
	return func(w *LoggerIOWriter) {
		if ctx == nil {
			return
		}

		w.ctx = ctx
	}
}

// LoggerIOWriterWithMsg set message for logger.
func LoggerIOWriterWithMsg(msg string) LoggerIOWriterOpt {
	return func(w *LoggerIOWriter) {
		msg = strings.TrimSpace(msg)
		if msg == "" {
			return
		}

		w.msg = msg
	}
}

// LoggerIOWriter wrap ylog.Logger as io.Writer
type LoggerIOWriter struct {
	ctx    context.Context
	msg    string
	logger Logger
}

var _ io.Writer = (*LoggerIOWriter)(nil)

// Write writes p as debug log using ylog.Logger.
// Since p may contain valid JSON object, we try to convert it as native Go object.
// Because if we write p directly to logger, it will print as Base64 encoded string.
// As a penalty, it may require some computation that not actually needed only to print the formatted JSON.
func (l *LoggerIOWriter) Write(p []byte) (n int, err error) {
	var jsonObj interface{}
	if json.Unmarshal(p, &jsonObj) != nil {
		jsonObj = string(p)
	}

	l.logger.Debug(l.ctx, l.msg, KV("data", jsonObj))
	return len(p), nil
}

// WrapIOWriter wrap ylog.Logger as io.Writer without context.
func WrapIOWriter(logger Logger, opts ...LoggerIOWriterOpt) io.Writer {
	if logger == nil {
		logger = NewNoop()
	}

	w := &LoggerIOWriter{
		ctx:    context.Background(),
		msg:    "ylog.Logger wrap to io.Writer",
		logger: logger,
	}

	for _, opt := range opts {
		opt(w)
	}

	return w
}
