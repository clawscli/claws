package aws

import (
	"context"
	"time"
)

const (
	// DefaultAPITimeout is the default timeout for AWS API calls.
	DefaultAPITimeout = 30 * time.Second

	// LongAPITimeout is for operations that may take longer (e.g., large listings).
	LongAPITimeout = 60 * time.Second
)

// WithAPITimeout returns a context with the default API timeout.
// The caller must call the returned cancel function to release resources.
func WithAPITimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(ctx, DefaultAPITimeout)
}

// WithLongAPITimeout returns a context with a longer timeout for slow operations.
// The caller must call the returned cancel function to release resources.
func WithLongAPITimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(ctx, LongAPITimeout)
}

// WithCustomTimeout returns a context with a custom timeout duration.
// The caller must call the returned cancel function to release resources.
func WithCustomTimeout(ctx context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(ctx, timeout)
}
