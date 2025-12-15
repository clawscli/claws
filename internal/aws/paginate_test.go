package aws

import (
	"context"
	"errors"
	"testing"
)

func TestPaginate(t *testing.T) {
	t.Run("single page", func(t *testing.T) {
		items, err := Paginate(context.Background(), func(token *string) ([]int, *string, error) {
			return []int{1, 2, 3}, nil, nil
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(items) != 3 {
			t.Errorf("expected 3 items, got %d", len(items))
		}
	})

	t.Run("multiple pages", func(t *testing.T) {
		page := 0
		items, err := Paginate(context.Background(), func(token *string) ([]int, *string, error) {
			page++
			switch page {
			case 1:
				next := "page2"
				return []int{1, 2}, &next, nil
			case 2:
				next := "page3"
				return []int{3, 4}, &next, nil
			case 3:
				return []int{5}, nil, nil
			default:
				t.Fatal("unexpected page")
				return nil, nil, nil
			}
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(items) != 5 {
			t.Errorf("expected 5 items, got %d", len(items))
		}
	})

	t.Run("error on fetch", func(t *testing.T) {
		expectedErr := errors.New("fetch error")
		_, err := Paginate(context.Background(), func(token *string) ([]int, *string, error) {
			return nil, nil, expectedErr
		})
		if !errors.Is(err, expectedErr) {
			t.Errorf("expected error %v, got %v", expectedErr, err)
		}
	})

	t.Run("context cancellation", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		page := 0
		_, err := Paginate(ctx, func(token *string) ([]int, *string, error) {
			page++
			if page == 2 {
				cancel()
			}
			next := "next"
			return []int{page}, &next, nil
		})
		if !errors.Is(err, context.Canceled) {
			t.Errorf("expected context.Canceled, got %v", err)
		}
	})

	t.Run("empty token string treated as done", func(t *testing.T) {
		items, err := Paginate(context.Background(), func(token *string) ([]int, *string, error) {
			empty := ""
			return []int{1}, &empty, nil
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(items) != 1 {
			t.Errorf("expected 1 item, got %d", len(items))
		}
	})
}

func TestPaginateIter(t *testing.T) {
	t.Run("iterate all items", func(t *testing.T) {
		page := 0
		seq := PaginateIter(context.Background(), func(token *string) ([]int, *string, error) {
			page++
			switch page {
			case 1:
				next := "page2"
				return []int{1, 2}, &next, nil
			case 2:
				return []int{3}, nil, nil
			default:
				return nil, nil, nil
			}
		})

		var items []int
		for item, err := range seq {
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			items = append(items, item)
		}
		if len(items) != 3 {
			t.Errorf("expected 3 items, got %d", len(items))
		}
	})

	t.Run("early termination", func(t *testing.T) {
		fetchCount := 0
		seq := PaginateIter(context.Background(), func(token *string) ([]int, *string, error) {
			fetchCount++
			next := "more"
			return []int{1, 2, 3}, &next, nil
		})

		count := 0
		for _, err := range seq {
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			count++
			if count >= 2 {
				break // Early termination
			}
		}
		if count != 2 {
			t.Errorf("expected 2 items, got %d", count)
		}
		if fetchCount != 1 {
			t.Errorf("expected 1 fetch, got %d", fetchCount)
		}
	})
}

func TestCollectWithLimit(t *testing.T) {
	t.Run("collect with limit", func(t *testing.T) {
		seq := PaginateIter(context.Background(), func(token *string) ([]int, *string, error) {
			next := "more"
			return []int{1, 2, 3, 4, 5}, &next, nil
		})

		items, err := CollectWithLimit(seq, 3)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(items) != 3 {
			t.Errorf("expected 3 items, got %d", len(items))
		}
	})

	t.Run("collect all when limit is 0", func(t *testing.T) {
		page := 0
		seq := PaginateIter(context.Background(), func(token *string) ([]int, *string, error) {
			page++
			if page > 2 {
				return nil, nil, nil
			}
			next := "more"
			return []int{1, 2}, &next, nil
		})

		items, err := CollectWithLimit(seq, 0)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(items) != 4 {
			t.Errorf("expected 4 items, got %d", len(items))
		}
	})
}
