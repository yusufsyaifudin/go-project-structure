package httpservermw

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/yusufsyaifudin/go-project-structure/pkg/ylog"
)

type PrometheusOpt func(*Prometheus) error

// PrometheusOptEnableGoMetric enable Golang runtime metric or not.
// Default false, means disabled.
func PrometheusOptEnableGoMetric(b bool) PrometheusOpt {
	return func(p *Prometheus) error {
		p.enableGoMetric = b
		return nil
	}
}

// PrometheusOptWithRegisterer set prometheus.Registerer.
// We need to pass the non-nil prometheus.Registerer before middleware PrometheusMiddleware can running.
// This because we may register another Collector outside HTTP middleware.
// For example, collectors.NewDBStatsCollector() can be registered outside this PrometheusMiddleware,
// then the same prometheus.Registerer passed into this function.
// This resulting the endpoint /metrics will show all Collector registered in the same Registerer.
func PrometheusOptWithRegisterer(reg prometheus.Registerer) PrometheusOpt {
	return func(p *Prometheus) error {
		if reg == nil {
			return fmt.Errorf("cannot use empty registerer for prometheus metric")
		}

		p.registerer = reg
		return nil
	}
}

// PrometheusOptWithGatherer register the prometheus.Gatherer.
// Gatherer is the interface for the part of a registry in charge of gathering
// the collected metrics into a number of MetricFamilies.
// We need to pass this prometheus.Gatherer so we can serve the /metrics endpoint with all Collected metric.
func PrometheusOptWithGatherer(gatherer prometheus.Gatherer) PrometheusOpt {
	return func(p *Prometheus) error {
		if gatherer == nil {
			return fmt.Errorf("cannot use empty gatherer for prometheus metric")
		}

		p.gatherer = gatherer
		return nil
	}
}

type Prometheus struct {
	baseMux http.Handler // required base multiplexer net/http server

	// options must be passed via PrometheusOpt function
	enableGoMetric bool
	logger         ylog.Logger
	registerer     prometheus.Registerer
	gatherer       prometheus.Gatherer

	// default statistic for HTTP middleware
	httpRequestsCounter  *prometheus.CounterVec
	httpRequestsDuration *prometheus.HistogramVec
}

var _ http.Handler = (*Prometheus)(nil)

// PrometheusMiddleware creates http.Handler and do some counter for HTTP statistic (request counter, etc),
// then will continue the request into next baseMux http.Handler.
// If user request to path /metrics, it will serve the metric instead of doing HTTP statistic.
func PrometheusMiddleware(baseMux http.Handler, opts ...PrometheusOpt) (*Prometheus, error) {
	if baseMux == nil {
		return nil, fmt.Errorf("prometheus middleware: cannot use nil http.Handler")
	}

	// We don't create global variable for any Prometheus stats (counter, gauge, etc) here.
	// Local variable is easier to debug since it scoped in this function.
	// No other function outside this package will call this.
	// Also, if this value changed, some other package must call through this Prometheus method handler,
	// which easier to track!
	httpRequests := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Number of HTTP requests made.",
		},
		[]string{"code", "method", "path"},
	)

	httpReqDur := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_requests_duration",
			Help:    "The HTTP request latencies in nanoseconds.",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"code", "method", "path"},
	)

	prom := &Prometheus{
		baseMux:              baseMux,
		enableGoMetric:       false,
		logger:               ylog.NewNoop(),
		httpRequestsCounter:  httpRequests,
		httpRequestsDuration: httpReqDur,
	}

	for _, opt := range opts {
		err := opt(prom)
		if err != nil {
			return nil, err
		}
	}

	// check registerer and gatherer for Prometheus to work.
	if prom.registerer == nil {
		return nil, fmt.Errorf("cannot setup prometheus middleware due to prometheus.Registerer is nil")
	}

	if prom.gatherer == nil {
		return nil, fmt.Errorf("cannot setup prometheus middleware due to prometheus.Gatherer is nil")
	}

	// register all stats collector
	err := prom.registerer.Register(httpRequests)
	if err != nil {
		return nil, err
	}

	if prom.enableGoMetric {
		err = prom.registerer.Register(collectors.NewGoCollector(
			collectors.WithGoCollectorRuntimeMetrics(),
		))
		if err != nil {
			return nil, err
		}
	}

	return prom, nil
}

// ServeHTTP implements http.Handler and act as net/http server middleware.
func (p *Prometheus) ServeHTTP(w http.ResponseWriter, req *http.Request) {

	t0 := time.Now()

	if req != nil && req.URL != nil && req.URL.Path == "/metrics" {
		// If user request /metrics endpoint,
		// then return the Prometheus metrics.
		prometheusHandler := promhttp.InstrumentMetricHandler(
			p.registerer, promhttp.HandlerFor(p.gatherer, promhttp.HandlerOpts{}),
		)

		prometheusHandler.ServeHTTP(w, req)

		// Don't do any further request processing.
		// In this scope the response should already
		// be written by the Prometheus HTTP handler.
		return
	}

	// capture response for statistic purpose
	respRec := httptest.NewRecorder()
	p.baseMux.ServeHTTP(respRec, req) // continue to the next http.Handler

	// ** Doing some stats counter/gauge/anything here...
	// Always increment request counter
	p.httpRequestsCounter.With(prometheus.Labels{
		"code":   strconv.Itoa(respRec.Code),
		"method": req.Method,
		"path":   req.URL.Path,
	}).Inc()

	p.httpRequestsDuration.With(prometheus.Labels{
		"code":   strconv.Itoa(respRec.Code),
		"method": req.Method,
		"path":   req.URL.Path,
	}).Observe(float64(time.Since(t0).Nanoseconds()))

	// Write headers to actual writer.
	for k, v := range respRec.Header() {
		w.Header().Set(k, strings.Join(v, " "))
	}

	// Write response status code.
	w.WriteHeader(respRec.Code)

	// Write response body.
	if n, err := w.Write(respRec.Body.Bytes()); err != nil {
		ctx := context.Background()
		if req != nil && req.Context() != nil {
			ctx = req.Context()
		}

		p.logger.Error(ctx, "prometheus middleware writing response body error",
			ylog.KV("bytes_written", n),
			ylog.KV("error", err),
		)
	}
}
