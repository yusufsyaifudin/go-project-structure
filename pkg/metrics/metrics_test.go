package metrics_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yusufsyaifudin/go-project-structure/pkg/metrics"
)

// ─── NoopMetric ───────────────────────────────────────────────────────────────

func TestNewNoop(t *testing.T) {
	m := metrics.NewNoop()
	assert.NotNil(t, m)
}

func TestNoop_GetCounterVec(t *testing.T) {
	m := metrics.NewNoop()
	cv := m.GetCounterVec("requests", "method")
	assert.NotNil(t, cv)
	c := cv.WithValues("GET")
	assert.NotNil(t, c)
	c.Incr(1) // must not panic
}

func TestNoop_GetGaugeVec(t *testing.T) {
	m := metrics.NewNoop()
	gv := m.GetGaugeVec("connections", "host")
	assert.NotNil(t, gv)
	g := gv.WithValues("localhost")
	assert.NotNil(t, g)
	g.Set(10)
	g.Incr(1)
	g.Decr(1)
}

func TestNoop_GetTimerVec(t *testing.T) {
	m := metrics.NewNoop()
	tv := m.GetTimerVec("latency", "route")
	assert.NotNil(t, tv)
	ti := tv.WithValues("/api/v1")
	assert.NotNil(t, ti)
	ti.Timing(1000)
}

func TestNoop_HandlerFunc_IsNil(t *testing.T) {
	m := metrics.NewNoop()
	assert.Nil(t, m.HandlerFunc())
}

func TestNoop_Close(t *testing.T) {
	m := metrics.NewNoop()
	assert.NoError(t, m.Close())
}

// ─── Prometheus ───────────────────────────────────────────────────────────────

func TestNewPrometheus_Default(t *testing.T) {
	p, err := metrics.NewPrometheus()
	require.NoError(t, err)
	assert.NotNil(t, p)
}

func TestNewPrometheus_WithPrefix(t *testing.T) {
	p, err := metrics.NewPrometheus(metrics.PrometheusWithPrefix("myapp_"))
	require.NoError(t, err)
	assert.NotNil(t, p)
}

func TestPrometheus_GetCounterVec(t *testing.T) {
	p, err := metrics.NewPrometheus()
	require.NoError(t, err)

	cv := p.GetCounterVec("test_counter", "method", "status")
	assert.NotNil(t, cv)
	cv.WithValues("GET", "200").Incr(1)
}

func TestPrometheus_GetCounterVec_SameNameSameLabels(t *testing.T) {
	p, err := metrics.NewPrometheus()
	require.NoError(t, err)

	cv1 := p.GetCounterVec("shared_counter", "code")
	cv2 := p.GetCounterVec("shared_counter", "code") // same name + labels → returns cached
	assert.NotNil(t, cv1)
	assert.NotNil(t, cv2)
}

func TestPrometheus_GetCounterVec_MismatchLabels_Panics(t *testing.T) {
	p, err := metrics.NewPrometheus()
	require.NoError(t, err)

	p.GetCounterVec("conflict_counter", "a")
	assert.Panics(t, func() {
		p.GetCounterVec("conflict_counter", "b") // different labels → panic
	})
}

func TestPrometheus_GetGaugeVec(t *testing.T) {
	p, err := metrics.NewPrometheus()
	require.NoError(t, err)

	gv := p.GetGaugeVec("test_gauge", "host")
	assert.NotNil(t, gv)
	g := gv.WithValues("localhost")
	g.Set(5)
	g.Incr(1)
	g.Decr(2)
}

func TestPrometheus_GetGaugeVec_SameName(t *testing.T) {
	p, err := metrics.NewPrometheus()
	require.NoError(t, err)
	p.GetGaugeVec("shared_gauge", "label")
	gv := p.GetGaugeVec("shared_gauge", "label")
	assert.NotNil(t, gv)
}

func TestPrometheus_GetGaugeVec_MismatchLabels_Panics(t *testing.T) {
	p, err := metrics.NewPrometheus()
	require.NoError(t, err)
	p.GetGaugeVec("conflict_gauge", "x")
	assert.Panics(t, func() {
		p.GetGaugeVec("conflict_gauge", "y")
	})
}

func TestPrometheus_GetTimerVec(t *testing.T) {
	p, err := metrics.NewPrometheus()
	require.NoError(t, err)

	tv := p.GetTimerVec("test_timer", "route")
	assert.NotNil(t, tv)
	tv.WithValues("/health").Timing(500)
}

func TestPrometheus_GetTimerVec_SameName(t *testing.T) {
	p, err := metrics.NewPrometheus()
	require.NoError(t, err)
	p.GetTimerVec("shared_timer", "label")
	tv := p.GetTimerVec("shared_timer", "label")
	assert.NotNil(t, tv)
}

func TestPrometheus_GetTimerVec_MismatchLabels_Panics(t *testing.T) {
	p, err := metrics.NewPrometheus()
	require.NoError(t, err)
	p.GetTimerVec("conflict_timer", "p")
	assert.Panics(t, func() {
		p.GetTimerVec("conflict_timer", "q")
	})
}

func TestPrometheus_HandlerFunc(t *testing.T) {
	p, err := metrics.NewPrometheus()
	require.NoError(t, err)

	hf := p.HandlerFunc()
	assert.NotNil(t, hf)

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rec := httptest.NewRecorder()
	hf(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestPrometheus_Close(t *testing.T) {
	p, err := metrics.NewPrometheus()
	require.NoError(t, err)
	assert.NoError(t, p.Close())
}

// ─── Combine ─────────────────────────────────────────────────────────────────

func TestNewCombinedMetrics_Empty(t *testing.T) {
	c, err := metrics.NewCombinedMetrics()
	require.NoError(t, err)
	assert.NotNil(t, c)
}

func TestNewCombinedMetrics_WithNoop(t *testing.T) {
	c, err := metrics.NewCombinedMetrics(metrics.CombineMetricAdd(metrics.NewNoop()))
	require.NoError(t, err)
	assert.NotNil(t, c)
}

func TestNewCombinedMetrics_NilMetricError(t *testing.T) {
	_, err := metrics.NewCombinedMetrics(metrics.CombineMetricAdd(nil))
	assert.Error(t, err)
}

func TestCombine_GetCounterVec_Empty(t *testing.T) {
	c, _ := metrics.NewCombinedMetrics()
	cv := c.GetCounterVec("c", "l")
	assert.NotNil(t, cv)
	cv.WithValues("v").Incr(1) // uses noop path
}

func TestCombine_GetCounterVec_WithBackend(t *testing.T) {
	p, _ := metrics.NewPrometheus()
	c, _ := metrics.NewCombinedMetrics(metrics.CombineMetricAdd(p))
	cv := c.GetCounterVec("combine_counter", "label")
	assert.NotNil(t, cv)
	cv.WithValues("x").Incr(1)
}

func TestCombine_GetGaugeVec_Empty(t *testing.T) {
	c, _ := metrics.NewCombinedMetrics()
	gv := c.GetGaugeVec("g", "l")
	g := gv.WithValues("v")
	g.Set(1)
	g.Incr(1)
	g.Decr(1)
}

func TestCombine_GetGaugeVec_WithBackend(t *testing.T) {
	p, _ := metrics.NewPrometheus()
	c, _ := metrics.NewCombinedMetrics(metrics.CombineMetricAdd(p))
	gv := c.GetGaugeVec("combine_gauge", "label")
	g := gv.WithValues("host")
	g.Set(5)
	g.Incr(2)
	g.Decr(1)
}

func TestCombine_GetTimerVec_Empty(t *testing.T) {
	c, _ := metrics.NewCombinedMetrics()
	tv := c.GetTimerVec("t", "l")
	tv.WithValues("v").Timing(100)
}

func TestCombine_GetTimerVec_WithBackend(t *testing.T) {
	p, _ := metrics.NewPrometheus()
	c, _ := metrics.NewCombinedMetrics(metrics.CombineMetricAdd(p))
	tv := c.GetTimerVec("combine_timer", "label")
	tv.WithValues("route").Timing(100)
}

func TestCombine_HandlerFunc_WithPrometheus(t *testing.T) {
	p, _ := metrics.NewPrometheus()
	c, _ := metrics.NewCombinedMetrics(metrics.CombineMetricAdd(p))
	hf := c.HandlerFunc()
	assert.NotNil(t, hf)
}

func TestCombine_HandlerFunc_NoPrometheus(t *testing.T) {
	c, _ := metrics.NewCombinedMetrics(metrics.CombineMetricAdd(metrics.NewNoop()))
	hf := c.HandlerFunc()
	assert.Nil(t, hf)
}

func TestCombine_Close_AllSucceed(t *testing.T) {
	c, _ := metrics.NewCombinedMetrics(
		metrics.CombineMetricAdd(metrics.NewNoop()),
		metrics.CombineMetricAdd(metrics.NewNoop()),
	)
	assert.NoError(t, c.Close())
}

func TestCombine_Close_WithError(t *testing.T) {
	wantErr := errors.New("close failed")
	c, _ := metrics.NewCombinedMetrics(metrics.CombineMetricAdd(&errMetric{closeErr: wantErr}))
	err := c.Close()
	assert.ErrorIs(t, err, wantErr)
}

// errMetric is a Metric that returns errors from Close.
type errMetric struct {
	closeErr error
}

func (e *errMetric) GetCounterVec(_ string, _ ...string) metrics.StatCounterVec {
	return metrics.NewNoop().GetCounterVec("")
}
func (e *errMetric) GetGaugeVec(_ string, _ ...string) metrics.StatGaugeVec {
	return metrics.NewNoop().GetGaugeVec("")
}
func (e *errMetric) GetTimerVec(_ string, _ ...string) metrics.StatTimerVec {
	return metrics.NewNoop().GetTimerVec("")
}
func (e *errMetric) HandlerFunc() http.HandlerFunc { return nil }
func (e *errMetric) Close() error                  { return e.closeErr }
