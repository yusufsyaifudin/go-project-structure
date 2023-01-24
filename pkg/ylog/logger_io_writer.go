package ylog

import (
	"context"
	"encoding/json"
	"io"
)

type LoggerIOWriterOpt func(*loggerIOWriter)

// LoggerIOWriterWithContext set context for logger.
func LoggerIOWriterWithContext(ctx context.Context) LoggerIOWriterOpt {
	return func(w *loggerIOWriter) {
		if ctx == nil {
			return
		}

		w.ctx = ctx
	}
}

// loggerIOWriter wrap ylog.Logger as io.Writer
type loggerIOWriter struct {
	ctx    context.Context
	logger Logger
}

var _ io.Writer = (*loggerIOWriter)(nil)

// Write writes p as debug log using ylog.Logger.
// Since p may contain valid JSON object, we try to convert it as native Go object.
// Because if we write p directly to logger, it will print as Base64 encoded string.
// As a penalty, it may require some computation that not actually needed only to print the formatted JSON.
func (l *loggerIOWriter) Write(p []byte) (n int, err error) {
	var jsonObj interface{}
	if json.Unmarshal(p, &jsonObj) != nil {
		jsonObj = string(p)
	}

	l.logger.Debug(l.ctx, "tracer log", KV("data", jsonObj))
	return len(p), nil
}

// WrapIOWriter wrap ylog.Logger as io.Writer without context.
func WrapIOWriter(logger Logger, opts ...LoggerIOWriterOpt) io.Writer {
	if logger == nil {
		logger = NewNoop()
	}

	w := &loggerIOWriter{
		ctx:    context.Background(),
		logger: logger,
	}

	for _, opt := range opts {
		opt(w)
	}

	return w
}
