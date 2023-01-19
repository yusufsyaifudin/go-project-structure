package observability

import (
	"github.com/yusufsyaifudin/go-project-structure/pkg/ylog"
	"go.opentelemetry.io/otel/trace"
)

// Observability is an interface that provide a range
// of observability APIs to components. This is primarily done the service-wide managers.
// Using this, we ensure that it easily to replace and manage rather than using global tracer and logger.
type Observability interface {
	Logger() ylog.Logger
	Tracer() trace.Tracer
}

type Noop struct{}

var _ Observability = (*Noop)(nil)

func NewNoop() *Noop {
	return &Noop{}
}

func (n *Noop) Logger() ylog.Logger {
	return &ylog.Noop{}
}

func (n *Noop) Tracer() trace.Tracer {
	return trace.NewNoopTracerProvider().Tracer("noop_tracer")
}
