package aws

import (
	"context"
	"iter"
)

// Paginate collects all items from a paginated AWS API using NextToken pattern.
// The fetch function should return items, the next token (nil if done), and any error.
func Paginate[T any](ctx context.Context, fetch func(token *string) (items []T, nextToken *string, err error)) ([]T, error) {
	var all []T
	var token *string

	for {
		items, nextToken, err := fetch(token)
		if err != nil {
			return nil, err
		}

		all = append(all, items...)

		if nextToken == nil || *nextToken == "" {
			break
		}
		token = nextToken

		// Check context cancellation between pages
		if err := ctx.Err(); err != nil {
			return nil, err
		}
	}

	return all, nil
}

// PaginateMarker is like Paginate but uses Marker instead of NextToken.
// Some AWS APIs (like ELBv2, Lambda) use Marker/NextMarker instead of NextToken.
func PaginateMarker[T any](ctx context.Context, fetch func(marker *string) (items []T, nextMarker *string, err error)) ([]T, error) {
	return Paginate(ctx, fetch)
}

// PaginateIter returns an iterator that yields items one at a time from paginated results.
// This is memory-efficient for large result sets and supports early termination.
// Uses Go 1.23+ range over function feature.
func PaginateIter[T any](ctx context.Context, fetch func(token *string) (items []T, nextToken *string, err error)) iter.Seq2[T, error] {
	return func(yield func(T, error) bool) {
		var token *string

		for {
			items, nextToken, err := fetch(token)
			if err != nil {
				var zero T
				yield(zero, err)
				return
			}

			for _, item := range items {
				if !yield(item, nil) {
					return // Early termination requested
				}
			}

			if nextToken == nil || *nextToken == "" {
				return
			}
			token = nextToken

			// Check context cancellation between pages
			if ctx.Err() != nil {
				var zero T
				yield(zero, ctx.Err())
				return
			}
		}
	}
}

// CollectWithLimit collects items from an iterator up to a maximum count.
// Returns the collected items and any error encountered.
func CollectWithLimit[T any](seq iter.Seq2[T, error], limit int) ([]T, error) {
	var items []T
	for item, err := range seq {
		if err != nil {
			return items, err
		}
		items = append(items, item)
		if limit > 0 && len(items) >= limit {
			break
		}
	}
	return items, nil
}
