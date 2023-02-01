package metrics

import (
	"fmt"
	"net/http"
	"sync"

	"github.com/google/go-cmp/cmp"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type PrometheusOpt func(*Prometheus) error

// PrometheusWithPrefix set prefix for each metric
func PrometheusWithPrefix(prefix string) PrometheusOpt {
	return func(p *Prometheus) error {
		p.prefix = prefix
		return nil
	}
}

type Prometheus struct {
	prefix     string
	registerer prometheus.Registerer
	gatherer   prometheus.Gatherer

	lock    sync.RWMutex
	counter map[string]*promCounterVec
	gauge   map[string]*promGaugeVec
	timer   map[string]*promTimerVec
}

var _ Metric = (*Prometheus)(nil)

func NewPrometheus(opts ...PrometheusOpt) (*Prometheus, error) {
	registry := prometheus.NewRegistry()

	p := &Prometheus{
		prefix:     "",
		registerer: registry,
		gatherer:   registry,
		lock:       sync.RWMutex{},
		counter:    make(map[string]*promCounterVec),
		gauge:      make(map[string]*promGaugeVec),
		timer:      make(map[string]*promTimerVec),
	}

	for _, opt := range opts {
		err := opt(p)
		if err != nil {
			return nil, err
		}
	}

	if p.prefix != "" {
		p.registerer = prometheus.WrapRegistererWithPrefix(p.prefix, p.registerer)
	}

	err := p.registerer.Register(collectors.NewGoCollector(
		collectors.WithGoCollectorRuntimeMetrics(),
	))

	if err != nil {
		return nil, err
	}

	return p, nil
}

func (p *Prometheus) GetCounterVec(name string, labelNames ...string) StatCounterVec {

	p.lock.RLock()
	counter, exist := p.counter[name]
	if exist && counter != nil {
		p.lock.RUnlock()
		if !cmp.Equal(labelNames, counter.registeredLabels) {
			panic(fmt.Errorf("counter vector name '%s' already registered: mismatch labels: %s vs %s", name, labelNames, counter.registeredLabels))
		}

		return counter
	}
	p.lock.RUnlock()

	p.lock.Lock()
	defer p.lock.Unlock()
	promCountVec := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: name,
		Help: fmt.Sprintf("%s counter metric", name),
	}, labelNames)

	p.registerer.MustRegister(promCountVec)

	c := &promCounterVec{
		counter:          promCountVec,
		registeredLabels: labelNames,
	}

	p.counter[name] = c
	return c
}

func (p *Prometheus) GetGaugeVec(name string, labelNames ...string) StatGaugeVec {
	p.lock.RLock()
	gauge, exist := p.gauge[name]
	if exist && gauge != nil {
		p.lock.RUnlock()
		if !cmp.Equal(labelNames, gauge.registeredLabels) {
			panic(fmt.Errorf("gauge vector name '%s' already registered: mismatch labels: %s vs %s", name, labelNames, gauge.registeredLabels))
		}

		return gauge
	}
	p.lock.RUnlock()

	p.lock.Lock()
	defer p.lock.Unlock()
	prom := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: name,
		Help: fmt.Sprintf("%s gauge metric", name),
	}, labelNames)

	p.registerer.MustRegister(prom)

	g := &promGaugeVec{
		gauge:            prom,
		registeredLabels: labelNames,
	}

	p.gauge[name] = g
	return g
}

func (p *Prometheus) GetTimerVec(name string, labelNames ...string) StatTimerVec {
	p.lock.RLock()
	summary, exist := p.timer[name]
	if exist && summary != nil {
		p.lock.RUnlock()
		if !cmp.Equal(labelNames, summary.registeredLabels) {
			panic(fmt.Errorf("timer vector name '%s' already registered: mismatch labels: %s vs %s", name, labelNames, summary.registeredLabels))
		}

		return summary
	}
	p.lock.RUnlock()

	p.lock.Lock()
	defer p.lock.Unlock()
	timerVec := prometheus.NewSummaryVec(prometheus.SummaryOpts{
		Name:       name,
		Help:       fmt.Sprintf("%s timer metric", name),
		Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
	}, labelNames)

	p.registerer.MustRegister(timerVec)

	t := &promTimerVec{
		timing:           timerVec,
		registeredLabels: labelNames,
	}

	p.timer[name] = t
	return t
}

func (p *Prometheus) HandlerFunc() http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		promhttp.HandlerFor(p.gatherer, promhttp.HandlerOpts{}).ServeHTTP(writer, request)
	}
}

// Close TODO implement close
func (p *Prometheus) Close() error {
	return nil
}
