package metrics

import "github.com/prometheus/client_golang/prometheus"

type promCounter struct {
	counter          prometheus.Counter
	registeredLabels []string
}

var _ StatCounter = (*promCounter)(nil)

func (p *promCounter) Incr(count int64) {
	p.counter.Add(float64(count))
}

type promGauge struct {
	gauge            prometheus.Gauge
	registeredLabels []string
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
	if count > 0 {
		count = -count
	}

	p.gauge.Add(float64(count))
}

type promTimer struct {
	timing           prometheus.Observer
	registeredLabels []string
}

var _ StatTimer = (*promTimer)(nil)

func (p *promTimer) Timing(delta int64) {
	p.timing.Observe(float64(delta))
}
