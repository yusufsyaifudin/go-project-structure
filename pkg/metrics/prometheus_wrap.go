package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

type promCounterVec struct {
	counter          *prometheus.CounterVec
	registeredLabels []string
}

var _ StatCounterVec = (*promCounterVec)(nil)

func (p *promCounterVec) WithValues(labelValues ...string) StatCounter {
	return &promCounter{
		counter: p.counter.WithLabelValues(labelValues...),
	}
}

type promCounter struct {
	counter prometheus.Counter
}

var _ StatCounter = (*promCounter)(nil)

func (p *promCounter) Incr(count int64) {
	p.counter.Add(float64(count))
}

type promGaugeVec struct {
	gauge            *prometheus.GaugeVec
	registeredLabels []string
}

func (p *promGaugeVec) WithValues(labelValues ...string) StatGauge {
	return &promGauge{
		gauge: p.gauge.WithLabelValues(labelValues...),
	}
}

var _ StatGaugeVec = (*promGaugeVec)(nil)

type promGauge struct {
	gauge prometheus.Gauge
}

var _ StatGauge = (*promGauge)(nil)

func (p *promGauge) Set(value int64) {
	p.gauge.Set(float64(value))
}

func (p *promGauge) Incr(count int64) {
	p.gauge.Add(float64(count))
}

func (p *promGauge) Decr(count int64) {
	// Decrement should be negative number
	p.gauge.Add(float64(-count))
}

type promTimerVec struct {
	timing           *prometheus.SummaryVec
	registeredLabels []string
}

func (p *promTimerVec) WithValues(labelValues ...string) StatTimer {
	return &promTimer{
		timing: p.timing.WithLabelValues(labelValues...),
	}
}

var _ StatTimerVec = (*promTimerVec)(nil)

type promTimer struct {
	timing prometheus.Observer
}

var _ StatTimer = (*promTimer)(nil)

func (p *promTimer) Timing(delta int64) {
	p.timing.Observe(float64(delta))
}
