package ylog_test

import (
	"bytes"
	"context"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yusufsyaifudin/go-project-structure/pkg/ylog"
)

func newTextHandler(buf *bytes.Buffer) slog.Handler {
	return slog.NewTextHandler(buf, &slog.HandlerOptions{Level: slog.LevelDebug})
}

func TestNewOTEL_NotNil(t *testing.T) {
	h := ylog.NewOTEL(newTextHandler(&bytes.Buffer{}), nil)
	assert.NotNil(t, h)
}

func TestOpenTelemetry_Enabled(t *testing.T) {
	h := ylog.NewOTEL(newTextHandler(&bytes.Buffer{}), nil)
	assert.True(t, h.Enabled(context.Background(), slog.LevelDebug))
	assert.True(t, h.Enabled(context.Background(), slog.LevelInfo))
}

func TestOpenTelemetry_Handle_NoSpan(t *testing.T) {
	var buf bytes.Buffer
	h := ylog.NewOTEL(newTextHandler(&buf), nil)
	logger := slog.New(h)

	logger.InfoContext(context.Background(), "test message", slog.String("key", "value"))
	output := buf.String()
	assert.Contains(t, output, "test message")
	assert.Contains(t, output, "trace_id")
	assert.Contains(t, output, "span_id")
}

func TestOpenTelemetry_Handle_WithContextExtractor(t *testing.T) {
	var buf bytes.Buffer
	opts := &ylog.OpenTelemetryOption{
		ContextExtractor: func(ctx context.Context) []slog.Attr {
			return []slog.Attr{
				slog.String("request_id", "req-abc"),
				// trace_id and span_id keys are skipped by the handler
				slog.String("trace_id", "should-be-skipped"),
				slog.String("span_id", "also-skipped"),
			}
		},
	}
	h := ylog.NewOTEL(newTextHandler(&buf), opts)
	logger := slog.New(h)

	logger.InfoContext(context.Background(), "with extractor")
	output := buf.String()
	assert.Contains(t, output, "request_id=req-abc")
	assert.Contains(t, output, "trace_id")
}

func TestOpenTelemetry_WithAttrs(t *testing.T) {
	var buf bytes.Buffer
	h := ylog.NewOTEL(newTextHandler(&buf), nil)
	h2 := h.WithAttrs([]slog.Attr{slog.String("persistent", "yes")})
	require.NotNil(t, h2)

	logger := slog.New(h2)
	logger.InfoContext(context.Background(), "attrs test")
	assert.Contains(t, buf.String(), "persistent=yes")
}

func TestOpenTelemetry_WithGroup_NonEmpty(t *testing.T) {
	var buf bytes.Buffer
	h := ylog.NewOTEL(newTextHandler(&buf), nil)
	hg := h.WithGroup("grp")
	require.NotNil(t, hg)
	assert.NotEqual(t, h, hg)
}

func TestOpenTelemetry_WithGroup_Empty(t *testing.T) {
	var buf bytes.Buffer
	h := ylog.NewOTEL(newTextHandler(&buf), nil)
	// Empty group name must return the same handler (noop per slog convention).
	hg := h.WithGroup("")
	assert.Equal(t, h, hg)
}
