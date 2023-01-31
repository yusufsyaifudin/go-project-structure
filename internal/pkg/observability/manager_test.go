package observability_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel/trace"

	"github.com/yusufsyaifudin/go-project-structure/internal/pkg/observability"
	"github.com/yusufsyaifudin/go-project-structure/pkg/metrics"
	"github.com/yusufsyaifudin/go-project-structure/pkg/ylog"
)

var mgrTest = &observability.Manager{}

func TestWithLogger(t *testing.T) {
	t.Run("nil", func(t *testing.T) {
		opt := observability.WithLogger(nil)
		err := opt(mgrTest)
		assert.Error(t, err)
	})

	t.Run("not-nil", func(t *testing.T) {
		opt := observability.WithLogger(ylog.NewNoop())
		err := opt(mgrTest)
		assert.NoError(t, err)
	})
}

func TestWithTracerProvider(t *testing.T) {
	t.Run("nil", func(t *testing.T) {
		opt := observability.WithTracerProvider(nil)
		err := opt(mgrTest)
		assert.Error(t, err)
	})

	t.Run("not-nil", func(t *testing.T) {
		opt := observability.WithTracerProvider(trace.NewNoopTracerProvider())
		err := opt(mgrTest)
		assert.NoError(t, err)
	})
}

func TestWithMetric(t *testing.T) {
	t.Run("nil", func(t *testing.T) {
		opt := observability.WithMetric(nil)
		err := opt(mgrTest)
		assert.Error(t, err)
	})

	t.Run("not-nil", func(t *testing.T) {
		opt := observability.WithMetric(metrics.NewNoop())
		err := opt(mgrTest)
		assert.NoError(t, err)
	})
}

func TestNewManager(t *testing.T) {
	t.Run("ok", func(t *testing.T) {
		observer, err := observability.NewManager()
		assert.NotNil(t, observer)
		assert.NoError(t, err)

		logger := observer.Logger()
		assert.NotNil(t, logger)

		tracer := observer.Tracer()
		assert.NotNil(t, tracer)
	})

	t.Run("error", func(t *testing.T) {
		observer, err := observability.NewManager(observability.WithLogger(nil))
		assert.Nil(t, observer)
		assert.Error(t, err)
	})
}
