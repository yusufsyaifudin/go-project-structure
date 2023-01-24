package oteltracer_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/yusufsyaifudin/go-project-structure/internal/pkg/oteltracer"
	"github.com/yusufsyaifudin/go-project-structure/pkg/ylog"
)

var tracerExporterText = &oteltracer.ExporterOption{}

func TestWithLogger(t *testing.T) {
	t.Run("nil", func(t *testing.T) {
		opt := oteltracer.WithLogger(nil)
		err := opt(tracerExporterText)
		assert.Error(t, err)
	})

	t.Run("not-nil", func(t *testing.T) {
		opt := oteltracer.WithLogger(ylog.NewNoop())
		err := opt(tracerExporterText)
		assert.NoError(t, err)
	})
}

func TestWithJaegerEndpoint(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		opt := oteltracer.WithJaegerEndpoint("")
		err := opt(tracerExporterText)
		assert.NoError(t, err)
	})

	t.Run("not-nil", func(t *testing.T) {
		opt := oteltracer.WithJaegerEndpoint("http://localhost:14268/api/traces")
		err := opt(tracerExporterText)
		assert.NoError(t, err)
	})
}

func TestWithOTLPEndpoint(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		opt := oteltracer.WithOTLPEndpoint("")
		err := opt(tracerExporterText)
		assert.NoError(t, err)
	})

	t.Run("not-nil", func(t *testing.T) {
		opt := oteltracer.WithOTLPEndpoint("localhost:4318")
		err := opt(tracerExporterText)
		assert.NoError(t, err)
	})
}

func TestNewTracerExporter(t *testing.T) {
	t.Run("error exporter", func(t *testing.T) {
		spanExporter, err := oteltracer.NewTracerExporter("stdout", oteltracer.WithLogger(nil))
		assert.Nil(t, spanExporter)
		assert.Error(t, err)
	})

	t.Run("available types", func(t *testing.T) {
		types := []string{
			"JAEGER",
			"OTLP",
			"STDOUT",
			"NOOP",
		}

		for _, ty := range types {
			t.Run(ty, func(t *testing.T) {
				spanExporter, err := oteltracer.NewTracerExporter(ty)
				assert.NotNil(t, spanExporter)
				assert.NoError(t, err)
			})
		}
	})

	t.Run("jaeger without endpoint", func(t *testing.T) {
		spanExporter, err := oteltracer.NewTracerExporter("jaeger", oteltracer.WithJaegerEndpoint(""))
		assert.Nil(t, spanExporter)
		assert.Error(t, err)
	})

	t.Run("otlp without endpoint", func(t *testing.T) {
		spanExporter, err := oteltracer.NewTracerExporter("otlp", oteltracer.WithOTLPEndpoint(""))
		assert.Nil(t, spanExporter)
		assert.Error(t, err)
	})

	t.Run("unknown type", func(t *testing.T) {
		spanExporter, err := oteltracer.NewTracerExporter("unknown")
		assert.Nil(t, spanExporter)
		assert.Error(t, err)
	})
}
