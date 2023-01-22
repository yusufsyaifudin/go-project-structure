package httpclientmw

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewHttpRoundTripper(t *testing.T) {
	t.Run("ok", func(t *testing.T) {
		mw := NewHttpRoundTripper()
		assert.NotNil(t, mw)
	})
}

func TestWithBaseRoundTripper(t *testing.T) {

}
