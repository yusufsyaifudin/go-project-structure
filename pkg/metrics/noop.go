package metrics

import "net/http"

type noopStatCounterVec struct{}

var _ StatCounterVec = (*noopStatCounterVec)(nil)

func (n *noopStatCounterVec) WithValues(labelValues ...string) StatCounter { return &noopStatCounter{} }

type noopStatCounter struct{}

var _ StatCounter = (*noopStatCounter)(nil)

func (n noopStatCounter) Incr(_ int64) {}

type noopStatGaugeVec struct{}

var _ StatGaugeVec = (*noopStatGaugeVec)(nil)

func (n *noopStatGaugeVec) WithValues(labelValues ...string) StatGauge { return &noopStatGauge{} }

type noopStatGauge struct{}

var _ StatGauge = (*noopStatGauge)(nil)

func (n *noopStatGauge) Set(_ int64) {}

func (n *noopStatGauge) Incr(_ int64) {}

func (n *noopStatGauge) Decr(_ int64) {}

type noopStatTimerVec struct{}

var _ StatTimerVec = (*noopStatTimerVec)(nil)

func (n *noopStatTimerVec) WithValues(labelValues ...string) StatTimer { return &noopStatTimer{} }

type noopStatTimer struct{}

var _ StatTimer = (*noopStatTimer)(nil)

func (n *noopStatTimer) Timing(_ int64) {}

type NoopMetric struct{}

func NewNoop() *NoopMetric {
	return &NoopMetric{}
}

var _ Metric = (*NoopMetric)(nil)

func (n *NoopMetric) GetCounterVec(name string, labelNames ...string) StatCounterVec {
	return &noopStatCounterVec{}
}

func (n *NoopMetric) GetGaugeVec(name string, labelNames ...string) StatGaugeVec {
	return &noopStatGaugeVec{}
}

func (n *NoopMetric) GetTimerVec(name string, labelNames ...string) StatTimerVec {
	return &noopStatTimerVec{}
}

func (n *NoopMetric) HandlerFunc() http.HandlerFunc {
	return nil
}

func (n *NoopMetric) Close() error {
	return nil
}
