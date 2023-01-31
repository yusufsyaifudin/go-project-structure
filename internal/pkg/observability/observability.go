package observability

import (
	"go.opentelemetry.io/otel/trace"

	"github.com/yusufsyaifudin/go-project-structure/pkg/metrics"
	"github.com/yusufsyaifudin/go-project-structure/pkg/ylog"
)

const (
	instrumentationName = "github.com/yusufsyaifudin/go-project-structure/internal/pkg/observability"
)

// Observability is an interface that provide a range
// of observability APIs to components. This is primarily done the service-wide managers.
// Using this, we ensure that it easily to replace and manage rather than using global tracerProvider and logger.
type Observability interface {
	Logger() ylog.Logger
	Tracer() trace.Tracer
	Metric() metrics.Metric
}

type Noop struct{}

var _ Observability = (*Noop)(nil)

// NewNoop return no-op Observability
func NewNoop() *Noop {
	return &Noop{}
}

func (n *Noop) Logger() ylog.Logger {
	return ylog.NewNoop()
}

func (n *Noop) Tracer() trace.Tracer {
	return trace.NewNoopTracerProvider().Tracer(instrumentationName)
}

func (n *Noop) Metric() metrics.Metric {
	return metrics.NewNoop()
}
