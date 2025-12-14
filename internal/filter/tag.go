// Package filter provides common filtering utilities for resources.
package filter

import "strings"

// MatchesTagFilter checks if a tags map matches the given filter.
// All matching is case-insensitive for both keys and values.
// Supported syntax:
//   - key=value: match on tag value (case-insensitive)
//   - key: tag key exists (any value, case-insensitive)
//   - key~partial: partial match on tag value (case-insensitive)
//
// Returns false if tags is nil or empty and filter is not empty.
func MatchesTagFilter(tags map[string]string, tagFilter string) bool {
	if tags == nil {
		return false
	}

	if tagFilter == "" {
		// No filter, match if has any tags
		return len(tags) > 0
	}

	// Parse the tag filter
	if strings.Contains(tagFilter, "~") {
		// Partial match: key~partial (case-insensitive)
		parts := strings.SplitN(tagFilter, "~", 2)
		if len(parts) != 2 {
			return false
		}
		key, partial := strings.ToLower(parts[0]), strings.ToLower(parts[1])
		for k, v := range tags {
			if strings.ToLower(k) == key {
				return strings.Contains(strings.ToLower(v), partial)
			}
		}
		return false
	}

	if strings.Contains(tagFilter, "=") {
		// Exact match: key=value (case-insensitive)
		parts := strings.SplitN(tagFilter, "=", 2)
		if len(parts) != 2 {
			return false
		}
		key, expected := strings.ToLower(parts[0]), strings.ToLower(parts[1])
		for k, v := range tags {
			if strings.ToLower(k) == key {
				return strings.ToLower(v) == expected
			}
		}
		return false
	}

	// Key exists: key (case-insensitive)
	keyLower := strings.ToLower(tagFilter)
	for k := range tags {
		if strings.ToLower(k) == keyLower {
			return true
		}
	}
	return false
}

// CycleIndex cycles an index through a range [0, length) in either direction.
// If reverse is true, decrements (wrapping from 0 to length-1).
// If reverse is false, increments (wrapping from length-1 to 0).
// Returns 0 if length <= 0.
func CycleIndex(current, length int, reverse bool) int {
	if length <= 0 {
		return 0
	}
	if reverse {
		current--
		if current < 0 {
			return length - 1
		}
		return current
	}
	current++
	if current >= length {
		return 0
	}
	return current
}
