package observability_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yusufsyaifudin/go-project-structure/internal/pkg/observability"
)

func TestNewNoop(t *testing.T) {
	t.Run("ok", func(t *testing.T) {
		observer := observability.NewNoop()
		assert.NotNil(t, observer)

		logger := observer.Logger()
		assert.NotNil(t, logger)

		tracer := observer.Tracer()
		assert.NotNil(t, tracer)
	})
}
