package config

import (
	"sync"

	intsync "github.com/clawscli/claws/internal/sync"
)

// withRLock executes fn while holding a read lock on mu.
func withRLock[T any](mu *sync.RWMutex, fn func() T) T {
	return intsync.WithRLock(mu, fn)
}

// doWithLock executes fn while holding a write lock on mu.
func doWithLock(mu *sync.RWMutex, fn func()) {
	intsync.DoWithLock(mu, fn)
}
