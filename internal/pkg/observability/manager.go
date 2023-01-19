package observability

import (
	"go.opentelemetry.io/otel/trace"

	"github.com/yusufsyaifudin/go-project-structure/pkg/ylog"
)

type Opt func(*Manager) error

func WithLogger(log ylog.Logger) Opt {
	return func(manager *Manager) error {
		if log == nil {
			return nil
		}

		manager.logger = log
		return nil
	}
}

func WithTracerName(name string) Opt {
	return func(manager *Manager) error {
		if name == "" {
			return nil
		}

		manager.tracerName = name
		return nil
	}
}

func WithTracerProvider(provider trace.TracerProvider) Opt {
	return func(manager *Manager) error {
		if provider == nil {
			return nil
		}

		manager.tracer = provider
		return nil
	}
}

type Manager struct {
	logger     ylog.Logger
	tracerName string
	tracer     trace.TracerProvider
}

var _ Observability = (*Manager)(nil)

func NewManager(opts ...Opt) (*Manager, error) {
	mgr := &Manager{
		logger:     &ylog.Noop{},
		tracerName: "default_tracer",
		tracer:     trace.NewNoopTracerProvider(),
	}

	for _, opt := range opts {
		err := opt(mgr)
		if err != nil {
			return nil, err
		}
	}

	return mgr, nil
}

func (m *Manager) Logger() ylog.Logger {
	return m.logger
}

func (m *Manager) Tracer() trace.Tracer {
	return m.tracer.Tracer(m.tracerName)
}
