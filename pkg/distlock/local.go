package distlock

import (
	"context"
	"fmt"
	"sync"
)

// LocalLocker is an in-process Locker backed by sync.Map.
// All operations are goroutine-safe. Close is a no-op.
// Suitable for single-process deployments and unit tests.
type LocalLocker struct {
	mu sync.Map
}

var _ Locker = (*LocalLocker)(nil)

// NewLocal creates a LocalLocker with an empty lock table.
func NewLocal() *LocalLocker {
	return &LocalLocker{}
}

// TryLock acquires the lock for key using a single atomic LoadOrStore.
// If the key is already held it returns ErrLockHeld (wrapping the holder ID).
func (l *LocalLocker) TryLock(_ context.Context, key, holderID string) error {
	actual, loaded := l.mu.LoadOrStore(key, holderID)
	if !loaded {
		return nil
	}
	holder, _ := actual.(string)
	if holder == "" {
		holder = "unknown"
	}
	return fmt.Errorf("%w (held by %s)", ErrLockHeld, holder)
}

// Unlock releases the lock for key. It is a no-op if the key is not held.
func (l *LocalLocker) Unlock(_ context.Context, key string) error {
	l.mu.Delete(key)
	return nil
}

// Close is a no-op; LocalLocker holds no external resources.
func (l *LocalLocker) Close() error {
	return nil
}

// isHeld reports whether key is currently held and returns the stored holder ID.
// Intended as a fast, read-only in-process pre-check for callers that want to
// short-circuit before doing more expensive lock acquisition work.
func (l *LocalLocker) isHeld(key string) (string, bool) {
	v, ok := l.mu.Load(key)
	if !ok {
		return "", false
	}
	holder, _ := v.(string)
	return holder, true
}
