package ylog_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/yusufsyaifudin/go-project-structure/pkg/ylog"
)

var ioLogTest = &ylog.LoggerIOWriter{}

func TestLoggerIOWriterWithContext(t *testing.T) {
	t.Run("nil context", func(t *testing.T) {
		opt := ylog.LoggerIOWriterWithContext(nil) //nolint:staticcheck
		opt(ioLogTest)
	})

	t.Run("with context", func(t *testing.T) {
		opt := ylog.LoggerIOWriterWithContext(context.TODO())
		opt(ioLogTest)
	})
}

func TestWrapIOWriter(t *testing.T) {
	t.Run("with noop logger", func(t *testing.T) {
		payload := `hi`

		writer := ylog.WrapIOWriter(nil)
		n, err := writer.Write([]byte(payload))
		assert.Equal(t, len(payload), n)
		assert.NoError(t, err)
	})

	t.Run("with context", func(t *testing.T) {
		payload := `hi`

		writer := ylog.WrapIOWriter(ylog.NewNoop(), ylog.LoggerIOWriterWithContext(context.Background()))
		n, err := writer.Write([]byte(payload))
		assert.Equal(t, len(payload), n)
		assert.NoError(t, err)
	})

	t.Run("invalid json", func(t *testing.T) {
		payload := `{"foo":"bar"`

		writer := ylog.WrapIOWriter(ylog.NewNoop())
		n, err := writer.Write([]byte(payload))
		assert.Equal(t, len(payload), n)
		assert.NoError(t, err)
	})

	t.Run("valid json", func(t *testing.T) {
		payload := `{"foo":"bar"}`

		writer := ylog.WrapIOWriter(ylog.NewNoop())
		n, err := writer.Write([]byte(payload))
		assert.Equal(t, len(payload), n)
		assert.NoError(t, err)
	})
}
