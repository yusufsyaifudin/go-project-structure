package observability

import (
	"fmt"

	"go.opentelemetry.io/otel/trace"

	"github.com/yusufsyaifudin/go-project-structure/pkg/ylog"
)

type Opt func(*Manager) error

// WithLogger set logger instance for the system observability.
func WithLogger(log ylog.Logger) Opt {
	return func(manager *Manager) error {
		if log == nil {
			return fmt.Errorf("cannot use nil logger")
		}

		manager.logger = log
		return nil
	}
}

// WithTracerProvider set OpenTelemetry tracer provider.
func WithTracerProvider(provider trace.TracerProvider) Opt {
	return func(manager *Manager) error {
		if provider == nil {
			return fmt.Errorf("cannot use nil tracer provider")
		}

		manager.tracerProvider = provider
		return nil
	}
}

type Manager struct {
	logger         ylog.Logger
	tracerProvider trace.TracerProvider
	tracer         trace.Tracer
}

var _ Observability = (*Manager)(nil)

// NewManager return Observability
func NewManager(opts ...Opt) (*Manager, error) {

	tp := trace.NewNoopTracerProvider()
	mgr := &Manager{
		logger:         ylog.NewNoop(),
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
