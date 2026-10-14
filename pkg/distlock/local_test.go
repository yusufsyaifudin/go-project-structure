package distlock

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewLocal(t *testing.T) {
	l := NewLocal()
	assert.NotNil(t, l)
}

func TestLocalLocker_TryLock_Success(t *testing.T) {
	l := NewLocal()
	err := l.TryLock(context.Background(), "/locks/key1", "holder-a")
	require.NoError(t, err)
}

func TestLocalLocker_TryLock_LockHeld(t *testing.T) {
	l := NewLocal()
	require.NoError(t, l.TryLock(context.Background(), "/locks/key1", "holder-a"))

	err := l.TryLock(context.Background(), "/locks/key1", "holder-b")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrLockHeld)
	assert.ErrorContains(t, err, "holder-a")
}

func TestLocalLocker_TryLock_LockHeld_EmptyHolder(t *testing.T) {
	// Storing an empty holderID triggers the "unknown" fallback in the error message.
	l := NewLocal()
	require.NoError(t, l.TryLock(context.Background(), "/locks/empty", ""))

	err := l.TryLock(context.Background(), "/locks/empty", "holder-b")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrLockHeld)
	assert.ErrorContains(t, err, "unknown")
}

func TestLocalLocker_Unlock_ReleasesLock(t *testing.T) {
	l := NewLocal()
	require.NoError(t, l.TryLock(context.Background(), "/locks/key2", "holder-a"))
	require.NoError(t, l.Unlock(context.Background(), "/locks/key2"))

	// Lock should be available again after Unlock.
	err := l.TryLock(context.Background(), "/locks/key2", "holder-b")
	require.NoError(t, err)
}

func TestLocalLocker_Unlock_NonExistentKey(t *testing.T) {
	l := NewLocal()
	// Unlocking a key that was never locked must not error.
	err := l.Unlock(context.Background(), "/locks/nonexistent")
	require.NoError(t, err)
}

func TestLocalLocker_Close(t *testing.T) {
	l := NewLocal()
	assert.NoError(t, l.Close())
}

func TestLocalLocker_IsHeld_True(t *testing.T) {
	l := NewLocal()
	require.NoError(t, l.TryLock(context.Background(), "/locks/key3", "holder-a"))
	holder, ok := l.isHeld("/locks/key3")
	assert.True(t, ok)
	assert.Equal(t, "holder-a", holder)
}

func TestLocalLocker_IsHeld_False(t *testing.T) {
	l := NewLocal()
	holder, ok := l.isHeld("/locks/key3")
	assert.False(t, ok)
	assert.Empty(t, holder)
}
