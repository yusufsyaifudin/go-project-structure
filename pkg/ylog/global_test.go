package ylog_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/yusufsyaifudin/go-project-structure/pkg/ylog"
)

func TestSetupZapLogger(t *testing.T) {
	levels := []string{
		"DEBUG",
		"INFO",
		"WARN",
		"ERROR",
		"PANIC",
		"FATAL",
	}

	for _, level := range levels {
		t.Run(level, func(t *testing.T) {
			logger := ylog.SetupZapLogger(level)
			assert.NotNil(t, logger)
		})
	}
}

func TestKV(t *testing.T) {
	t.Run("ok", func(t *testing.T) {
		k := "key"
		v := map[string]any{
			"number": 1,
			"float":  1.2,
			"bool":   true,
			"string": "text",
			"slices": []any{1, 1, 2, true, "text", []int{0, 1}},
		}

		kv := ylog.KV(k, v)
		assert.Equal(t, kv.Key(), k)
		assert.Equal(t, kv.Value(), v)
	})
}
