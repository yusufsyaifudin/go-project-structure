package idgen

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNextID_ReturnsPositive(t *testing.T) {
	id, err := NextID()
	require.NoError(t, err)
	assert.Greater(t, id, int64(0))
}

func TestNextID_Unique(t *testing.T) {
	id1, err := NextID()
	require.NoError(t, err)
	id2, err := NextID()
	require.NoError(t, err)
	assert.NotEqual(t, id1, id2)
}

func TestNextID_FitsInt64(t *testing.T) {
	id, err := NextID()
	require.NoError(t, err)
	// Sonyflake produces 63-bit IDs; the high bit must be zero.
	assert.GreaterOrEqual(t, id, int64(0))
}
