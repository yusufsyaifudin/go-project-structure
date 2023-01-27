package httpservermw_test

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"

	"github.com/yusufsyaifudin/go-project-structure/internal/pkg/httpservermw"
)

var promTest = &httpservermw.Prometheus{}

func TestPrometheusOptEnableGoMetric(t *testing.T) {
	t.Run("true", func(t *testing.T) {
		opt := httpservermw.PrometheusOptEnableGoMetric(true)
		err := opt(promTest)
		assert.NoError(t, err)
	})

	t.Run("false", func(t *testing.T) {
		opt := httpservermw.PrometheusOptEnableGoMetric(false)
		err := opt(promTest)
		assert.NoError(t, err)
	})
}

func TestPrometheusOptWithRegisterer(t *testing.T) {
	t.Run("nil", func(t *testing.T) {
		opt := httpservermw.PrometheusOptWithRegisterer(nil)
		err := opt(promTest)
		assert.NoError(t, err)
	})

	t.Run("non-nil", func(t *testing.T) {
		opt := httpservermw.PrometheusOptWithRegisterer(prometheus.NewRegistry())
		err := opt(promTest)
		assert.NoError(t, err)
	})
}

func TestPrometheusOptWithGatherer(t *testing.T) {
	t.Run("nil", func(t *testing.T) {
		opt := httpservermw.PrometheusOptWithGatherer(nil)
		err := opt(promTest)
		assert.NoError(t, err)
	})

	t.Run("non-nil", func(t *testing.T) {
		opt := httpservermw.PrometheusOptWithGatherer(prometheus.NewRegistry())
		err := opt(promTest)
		assert.NoError(t, err)
	})
}
