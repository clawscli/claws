package config

import "sync"

// withRLock executes fn while holding a read lock on mu.
func withRLock[T any](mu *sync.RWMutex, fn func() T) T {
	mu.RLock()
	defer mu.RUnlock()
	return fn()
}

// doWithLock executes fn while holding a write lock on mu.
func doWithLock(mu *sync.RWMutex, fn func()) {
	mu.Lock()
	defer mu.Unlock()
	fn()
}
