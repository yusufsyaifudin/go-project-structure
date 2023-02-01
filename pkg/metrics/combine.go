package metrics

import (
	"fmt"
	"net/http"

	"go.uber.org/multierr"
)

type CombineOpt func(*Combine) error

func CombineMetricAdd(m Metric) CombineOpt {
	return func(c *Combine) error {
		if m == nil {
			return fmt.Errorf("cannot add nil metric to combined metrics")
		}

		c.metrics = append(c.metrics, m)
		return nil
	}
}

type Combine struct {
	metrics []Metric
}

var _ Metric = (*Combine)(nil)

func NewCombinedMetrics(opts ...CombineOpt) (*Combine, error) {
	c := &Combine{
		metrics: make([]Metric, 0),
	}

	for _, opt := range opts {
		err := opt(c)
		if err != nil {
			return nil, err
		}
	}

	return c, nil
}

func (c *Combine) GetCounterVec(name string, labelNames ...string) StatCounterVec {
	counters := make([]StatCounterVec, 0)
	for _, metric := range c.metrics {
		counters = append(counters, metric.GetCounterVec(name, labelNames...))
	}

	if len(counters) <= 0 {
		return &noopStatCounterVec{}
	}

	return &combineStatCounterVec{
		counters: counters,
	}
}

func (c *Combine) GetGaugeVec(name string, labelNames ...string) StatGaugeVec {
	gauges := make([]StatGaugeVec, 0)
	for _, metric := range c.metrics {
		gauges = append(gauges, metric.GetGaugeVec(name, labelNames...))
	}

	if len(gauges) <= 0 {
		return &noopStatGaugeVec{}
	}

	return &combineStatGaugeVec{
		gauges: gauges,
	}
}

func (c *Combine) GetTimerVec(name string, labelNames ...string) StatTimerVec {
	timers := make([]StatTimerVec, 0)
	for _, metric := range c.metrics {
		timers = append(timers, metric.GetTimerVec(name, labelNames...))
	}

	if len(timers) <= 0 {
		return &noopStatTimerVec{}
	}

	return &combineStatTimerVec{
		timers: timers,
	}
}

func (c *Combine) HandlerFunc() http.HandlerFunc {
	for _, metric := range c.metrics {
		if p, ok := metric.(*Prometheus); ok && metric.HandlerFunc() != nil {
			return p.HandlerFunc()
		}
	}

	return nil
}

func (c *Combine) Close() error {
	var err error
	for _, metric := range c.metrics {
		if _err := metric.Close(); _err != nil {
			err = multierr.Append(err, _err)
		}
	}

	return err
}

type combineStatCounterVec struct {
	counters []StatCounterVec
}

func (c *combineStatCounterVec) WithValues(labelValues ...string) StatCounter {
	counters := make([]StatCounter, 0)
	for _, counter := range c.counters {
		if counter == nil {
			continue
		}
		counters = append(counters, counter.WithValues(labelValues...))
	}

	return &combineStatCounter{
		counters: counters,
	}
}

var _ StatCounterVec = (*combineStatCounterVec)(nil)

type combineStatCounter struct {
	counters []StatCounter
}

var _ StatCounter = (*combineStatCounter)(nil)

func (c *combineStatCounter) Incr(count int64) {
	for _, counter := range c.counters {
		if counter == nil {
			continue
		}
		counter.Incr(count)
	}
}

type combineStatGaugeVec struct {
	gauges []StatGaugeVec
}

var _ StatGaugeVec = (*combineStatGaugeVec)(nil)

func (c *combineStatGaugeVec) WithValues(labelValues ...string) StatGauge {
	gauges := make([]StatGauge, 0)
	for _, gauge := range c.gauges {
		if gauge == nil {
			continue
		}
		gauges = append(gauges, gauge.WithValues(labelValues...))
	}

	return &combineStatGauge{
		gauges: gauges,
	}
}

type combineStatGauge struct {
	gauges []StatGauge
}

var _ StatGauge = (*combineStatGauge)(nil)

func (c *combineStatGauge) Set(value int64) {
	for _, gauge := range c.gauges {
		if gauge == nil {
			continue
		}
		gauge.Set(value)
	}
}

func (c *combineStatGauge) Incr(count int64) {
	for _, gauge := range c.gauges {
		if gauge == nil {
			continue
		}
		gauge.Incr(count)
	}
}

func (c *combineStatGauge) Decr(count int64) {
	for _, gauge := range c.gauges {
		if gauge == nil {
			continue
		}
		gauge.Decr(count)
	}
}

type combineStatTimerVec struct {
	timers []StatTimerVec
}

var _ StatTimerVec = (*combineStatTimerVec)(nil)

func (c combineStatTimerVec) WithValues(labelValues ...string) StatTimer {
	timers := make([]StatTimer, 0)
	for _, timer := range c.timers {
		if timer == nil {
			continue
		}
		timers = append(timers, timer.WithValues(labelValues...))
	}

	return &combineStatTimer{
		timers: timers,
	}
}

type combineStatTimer struct {
	timers []StatTimer
}

var _ StatTimer = (*combineStatTimer)(nil)

func (c *combineStatTimer) Timing(delta int64) {
	for _, timer := range c.timers {
		if timer == nil {
			continue
		}
		timer.Timing(delta)
	}
}
