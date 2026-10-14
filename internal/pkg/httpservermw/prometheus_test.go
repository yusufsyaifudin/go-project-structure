package httpservermw_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yusufsyaifudin/go-project-structure/internal/pkg/httpservermw"
	"github.com/yusufsyaifudin/go-project-structure/pkg/metrics"
)

func okHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
}

func TestPrometheusMiddleware_NilBaseMux(t *testing.T) {
	_, err := httpservermw.PrometheusMiddleware(nil, httpservermw.PrometheusWithMetric(metrics.NewNoop()))
	assert.Error(t, err)
}

func TestPrometheusMiddleware_Success(t *testing.T) {
	p, err := httpservermw.PrometheusMiddleware(okHandler(), httpservermw.PrometheusWithMetric(metrics.NewNoop()))
	require.NoError(t, err)
	assert.NotNil(t, p)
}

func TestPrometheusMiddleware_ServeHTTP_NormalRequest(t *testing.T) {
	m := metrics.NewNoop()
	p, err := httpservermw.PrometheusMiddleware(okHandler(), httpservermw.PrometheusWithMetric(m))
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/test", nil)
	rec := httptest.NewRecorder()
	p.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "ok", rec.Body.String())
}

func TestPrometheusMiddleware_ServeHTTP_MetricsEndpoint(t *testing.T) {
	pm, err := metrics.NewPrometheus()
	require.NoError(t, err)

	p, err := httpservermw.PrometheusMiddleware(okHandler(), httpservermw.PrometheusWithMetric(pm))
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rec := httptest.NewRecorder()
	p.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "go_")
}
