package view

import (
	"slices"
	"strings"
)

// fuzzyMatch checks if pattern characters appear in order in str (case insensitive)
func fuzzyMatch(str, pattern string) bool {
	str = strings.ToLower(str)
	pattern = strings.ToLower(pattern)
	pi := 0
	for i := 0; i < len(str) && pi < len(pattern); i++ {
		if str[i] == pattern[pi] {
			pi++
		}
	}
	return pi == len(pattern)
}

// matchNamesWithFallback returns names matching the pattern.
// It first tries prefix matching, then falls back to fuzzy matching if no prefix matches.
func matchNamesWithFallback(names []string, pattern string) []string {
	if pattern == "" {
		result := slices.Clone(names)
		slices.Sort(result)
		return result
	}

	pattern = strings.ToLower(pattern)

	var prefixMatches []string
	for _, name := range names {
		if strings.HasPrefix(strings.ToLower(name), pattern) {
			prefixMatches = append(prefixMatches, name)
		}
	}
	if len(prefixMatches) > 0 {
		slices.Sort(prefixMatches)
		return prefixMatches
	}

	var fuzzyMatches []string
	for _, name := range names {
		if fuzzyMatch(name, pattern) {
			fuzzyMatches = append(fuzzyMatches, name)
		}
	}
	slices.Sort(fuzzyMatches)
	return fuzzyMatches
}
