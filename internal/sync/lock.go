// Package sync provides synchronization utilities.
package sync

import "sync"

// WithRLock executes fn while holding a read lock on mu and returns the result.
func WithRLock[T any](mu *sync.RWMutex, fn func() T) T {
	mu.RLock()
	defer mu.RUnlock()
	return fn()
}

// DoWithLock executes fn while holding a write lock on mu.
func DoWithLock(mu *sync.RWMutex, fn func()) {
	mu.Lock()
	defer mu.Unlock()
	fn()
}
