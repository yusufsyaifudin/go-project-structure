package metrics

import (
	"fmt"
	"net/http"
	"sort"
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
	counter map[string]*promCounter
	gauge   map[string]*promGauge
	timer   map[string]*promTimer
}

var _ Metric = (*Prometheus)(nil)

func NewPrometheus(opts ...PrometheusOpt) (*Prometheus, error) {
	registry := prometheus.NewRegistry()

	p := &Prometheus{
		prefix:     "",
		registerer: registry,
		gatherer:   registry,
		lock:       sync.RWMutex{},
		counter:    make(map[string]*promCounter),
		gauge:      make(map[string]*promGauge),
		timer:      make(map[string]*promTimer),
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

func (p *Prometheus) GetCounterVec(name string, labels map[string]string) StatCounter {
	labelNames := make([]string, 0)
	for label := range labels {
		labelNames = append(labelNames, label)
	}

	p.lock.RLock()
	counter, exist := p.counter[name]
	if exist && counter != nil {
		defer p.lock.RUnlock()
		sort.Strings(labelNames)
		sort.Strings(counter.registeredLabels)
		if diff := cmp.Diff(labelNames, counter.registeredLabels); diff != "" {
			panic(fmt.Errorf("counter vector name '%s' already registered: mismatch labels: %s", name, diff))
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

	counterLabelled := promCountVec.With(labels)
	p.registerer.MustRegister(counterLabelled) // register the labelled version to ensure non-duplicate collector

	c := &promCounter{
		counter:          counterLabelled,
		registeredLabels: labelNames,
	}

	p.counter[name] = c // register with prefixed name
	return counter
}

func (p *Prometheus) GetGaugeVec(name string, labels map[string]string) StatGauge {
	labelNames := make([]string, 0)
	for label := range labels {
		labelNames = append(labelNames, label)
	}

	p.lock.RLock()
	gauge, exist := p.gauge[name]
	if exist && gauge != nil {
		defer p.lock.RUnlock()
		sort.Strings(labelNames)
		sort.Strings(gauge.registeredLabels)
		if diff := cmp.Diff(labelNames, gauge.registeredLabels); diff != "" {
			panic(fmt.Errorf("gauge vector name '%s' already registered: mismatch labels: %s", name, diff))
		}

		return gauge
	}
	p.lock.RUnlock()

	p.lock.Lock()
	defer p.lock.Unlock()
	promGaugeVec := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: name,
		Help: fmt.Sprintf("%s gauge metric", name),
	}, labelNames)

	gaugeLabelled := promGaugeVec.With(labels)
	p.registerer.MustRegister(gaugeLabelled) // register the labelled version to ensure non-duplicate collector

	g := &promGauge{
		gauge:            gaugeLabelled,
		registeredLabels: labelNames,
	}

	p.gauge[name] = g // register with prefixed name
	return g
}

func (p *Prometheus) GetTimerVec(name string, labels map[string]string) StatTimer {
	labelNames := make([]string, 0)
	for label := range labels {
		labelNames = append(labelNames, label)
	}

	p.lock.RLock()
	summary, exist := p.timer[name]
	if exist && summary != nil {
		defer p.lock.RUnlock()
		sort.Strings(labelNames)
		sort.Strings(summary.registeredLabels)
		if diff := cmp.Diff(labelNames, summary.registeredLabels); diff != "" {
			panic(fmt.Errorf("timer vector name '%s' already registered: mismatch labels: %s", name, diff))
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

	timerLabelled := timerVec.With(labels)
	p.registerer.MustRegister(timerVec) // register the labelled version to ensure non-duplicate collector

	t := &promTimer{
		timing:           timerLabelled,
		registeredLabels: labelNames,
	}

	p.timer[name] = t // register with prefixed name
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
