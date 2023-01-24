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

		manager.tracerProvider = provider
		return nil
	}
}

type Manager struct {
	logger         ylog.Logger
	tracerName     string
	tracerProvider trace.TracerProvider
	tracer         trace.Tracer
}

var _ Observability = (*Manager)(nil)

func NewManager(opts ...Opt) (*Manager, error) {

	tp := trace.NewNoopTracerProvider()
	mgr := &Manager{
		logger:         &ylog.Noop{},
		tracerName:     "default_tracer",
		tracerProvider: tp,
		tracer:         tp.Tracer(instrumentationName),
	}

	for _, opt := range opts {
		err := opt(mgr)
		if err != nil {
			return nil, err
		}
	}

	// prepare tracer with injected tracer provider
	mgr.tracer = mgr.tracerProvider.Tracer(instrumentationName)

	return mgr, nil
}

func (m *Manager) Logger() ylog.Logger {
	return m.logger
}

func (m *Manager) Tracer() trace.Tracer {
	return m.tracer
}
