package httpservermw

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"time"

	"github.com/yusufsyaifudin/go-project-structure/pkg/metrics"
	"github.com/yusufsyaifudin/go-project-structure/pkg/ylog"
)

type PrometheusOpt func(*Prometheus) error

// PrometheusWithMetric set metrics.Metric
func PrometheusWithMetric(m *metrics.Prometheus) PrometheusOpt {
	return func(p *Prometheus) error {
		p.metric = m
		return nil
	}
}

type Prometheus struct {
	baseMux http.Handler // required base multiplexer net/http server

	// options must be passed via PrometheusOpt function
	logger ylog.Logger
	metric *metrics.Prometheus
}

var _ http.Handler = (*Prometheus)(nil)

// PrometheusMiddleware creates http.Handler and do some counter for HTTP statistic (request counter, etc),
// then will continue the request into next baseMux http.Handler.
// If user request to path /metrics, it will serve the metric instead of doing HTTP statistic.
func PrometheusMiddleware(baseMux http.Handler, opts ...PrometheusOpt) (*Prometheus, error) {
	if baseMux == nil {
		return nil, fmt.Errorf("prometheus middleware: cannot use nil http.Handler")
	}

	prom := &Prometheus{
		baseMux: baseMux,
		logger:  ylog.NewNoop(),
	}

	for _, opt := range opts {
		err := opt(prom)
		if err != nil {
			return nil, err
		}
	}

	return prom, nil
}

// ServeHTTP implements http.Handler and act as net/http server middleware.
func (p *Prometheus) ServeHTTP(w http.ResponseWriter, req *http.Request) {

	t0 := time.Now()
	if req != nil && req.URL != nil && req.URL.Path == "/metrics" && p.metric.HandlerFunc() != nil {
		// If user request /metrics endpoint,
		// then return the Prometheus metrics.

		p.metric.HandlerFunc().ServeHTTP(w, req)
		// Don't do any further request processing.
		// In this scope the response should already
		// be written by the Prometheus HTTP handler.
		return
	}

	// capture response for statistic purpose
	respRec := httptest.NewRecorder()
	p.baseMux.ServeHTTP(respRec, req) // continue to the next http.Handler

	// We don't create global variable for any Prometheus stats (counter, gauge, etc) here.
	// Local variable is easier to debug since it scoped in this function.
	// No other function outside this package will call this.
	// Also, if this value changed, some other package must call through this Prometheus method handler,
	// which easier to track!

	// ** Doing some stats counter/gauge/anything here...
	// Always increment request counter
	p.metric.
		GetCounterVec("http_requests_total", "code", "method", "path").
		WithValues(strconv.Itoa(respRec.Code), req.Method, req.URL.Path).
		Incr(1)

	p.metric.GetTimerVec("http_requests_duration", "code", "method", "path").
		WithValues(strconv.Itoa(respRec.Code), req.Method, req.URL.Path).
		Timing(time.Since(t0).Nanoseconds())

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
