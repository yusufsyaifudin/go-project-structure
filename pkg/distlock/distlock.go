package distlock

import (
	"context"
	"errors"
)

// ErrLockHeld is returned by TryLock when the key is already locked by another holder.
var ErrLockHeld = errors.New("lock already held")

// Locker is a distributed admission-control lock keyed by an arbitrary string.
// Implementations must be safe for concurrent use from multiple goroutines.
type Locker interface {
	// TryLock attempts to acquire an exclusive lock on key.
	// holderID is stored as the lock's value for observability (e.g. a trace ID).
	// Returns a wrapped ErrLockHeld if the key is already locked by another holder.
	TryLock(ctx context.Context, key, holderID string) error

	// Unlock releases the lock on key.
	// Callers should pass a context that is not tied to the request lifetime
	// (e.g. context.Background()) since the request context may already be
	// cancelled by the time the lock is released.
	Unlock(ctx context.Context, key string) error

	// Close releases any underlying resources held by the locker (e.g. leases or sessions).
	Close() error
}
