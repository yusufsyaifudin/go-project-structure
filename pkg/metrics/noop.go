package metrics

import "net/http"

type noopStatCounter struct{}

var _ StatCounter = (*noopStatCounter)(nil)

func (n noopStatCounter) Incr(_ int64) {}

type noopStatGauge struct{}

var _ StatGauge = (*noopStatGauge)(nil)

func (n *noopStatGauge) Set(_ int64) {}

func (n *noopStatGauge) Incr(_ int64) {}

func (n *noopStatGauge) Decr(_ int64) {}

type noopStatTimer struct{}

var _ StatTimer = (*noopStatTimer)(nil)

func (n *noopStatTimer) Timing(_ int64) {}

type NoopMetric struct{}

func NewNoop() *NoopMetric {
	return &NoopMetric{}
}

var _ Metric = (*NoopMetric)(nil)

func (n *NoopMetric) GetCounterVec(name string, labels map[string]string) StatCounter {
	return &noopStatCounter{}
}

func (n *NoopMetric) GetGaugeVec(name string, labels map[string]string) StatGauge {
	return &noopStatGauge{}
}

func (n *NoopMetric) GetTimerVec(name string, labels map[string]string) StatTimer {
	return &noopStatTimer{}
}

func (n *NoopMetric) HandlerFunc() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {}
}

func (n *NoopMetric) Close() error {
	return nil
}
