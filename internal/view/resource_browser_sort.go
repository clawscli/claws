package view

import (
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/clawscli/claws/internal/dao"
)

// applySorting sorts the filtered resources by the selected column
func (r *ResourceBrowser) applySorting() {
	if r.sortColumn < 0 || r.renderer == nil {
		return
	}

	cols := r.renderer.Columns()
	if r.sortColumn >= len(cols) {
		return
	}

	col := cols[r.sortColumn]
	if col.Getter == nil {
		return
	}

	slices.SortStableFunc(r.filtered, func(a, b dao.Resource) int {
		valA := col.Getter(dao.UnwrapResource(a))
		valB := col.Getter(dao.UnwrapResource(b))

		cmp := compareValues(valA, valB)
		if !r.sortAscending {
			cmp = -cmp
		}
		return cmp
	})
}

// compareValues compares two string values, attempting numeric/date comparison first
func compareValues(a, b string) int {
	// Try numeric comparison
	if numA, errA := parseNumeric(a); errA == nil {
		if numB, errB := parseNumeric(b); errB == nil {
			if numA < numB {
				return -1
			} else if numA > numB {
				return 1
			}
			return 0
		}
	}

	// Try age/duration comparison (e.g., "5d", "2h", "30m")
	if durA, okA := parseAge(a); okA {
		if durB, okB := parseAge(b); okB {
			if durA < durB {
				return -1
			} else if durA > durB {
				return 1
			}
			return 0
		}
	}

	// Fall back to string comparison (case-insensitive)
	return strings.Compare(strings.ToLower(a), strings.ToLower(b))
}

// parseNumeric attempts to parse a string as a number (handles sizes like "1.5 GiB")
func parseNumeric(s string) (float64, error) {
	s = strings.TrimSpace(s)
	if s == "" || s == "-" || s == "N/A" {
		return 0, strconv.ErrSyntax
	}

	// Handle size suffixes with multipliers
	multiplier := 1.0
	suffixes := map[string]float64{
		" TiB": 1024 * 1024 * 1024 * 1024,
		" GiB": 1024 * 1024 * 1024,
		" MiB": 1024 * 1024,
		" KiB": 1024,
		" TB":  1000 * 1000 * 1000 * 1000,
		" GB":  1000 * 1000 * 1000,
		" MB":  1000 * 1000,
		" KB":  1000,
		" B":   1,
		"%":    1,
	}

	for suffix, mult := range suffixes {
		if before, ok := strings.CutSuffix(s, suffix); ok {
			s = before
			multiplier = mult
			break
		}
	}

	val, err := strconv.ParseFloat(strings.TrimSpace(s), 64)
	if err != nil {
		return 0, err
	}
	return val * multiplier, nil
}

// parseAge parses age strings like "5d", "2h", "30m", "10s"
func parseAge(s string) (time.Duration, bool) {
	s = strings.TrimSpace(s)
	if s == "" || s == "-" || s == "N/A" {
		return 0, false
	}

	// Handle compound formats like "1y", "30d", "2h", "5m", "10s"
	multipliers := map[byte]time.Duration{
		's': time.Second,
		'm': time.Minute,
		'h': time.Hour,
		'd': 24 * time.Hour,
		'w': 7 * 24 * time.Hour,
		'y': 365 * 24 * time.Hour,
	}

	if len(s) < 2 {
		return 0, false
	}

	suffix := s[len(s)-1]
	mult, ok := multipliers[suffix]
	if !ok {
		// Try "mo" for months
		if strings.HasSuffix(s, "mo") {
			mult = 30 * 24 * time.Hour
			s = s[:len(s)-2]
		} else {
			return 0, false
		}
	} else {
		s = s[:len(s)-1]
	}

	num, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, false
	}

	return time.Duration(num * float64(mult)), true
}

// SetSort sets the sort column and direction
func (r *ResourceBrowser) SetSort(colIndex int, ascending bool) {
	r.sortColumn = colIndex
	r.sortAscending = ascending
}

// ClearSort clears sorting
func (r *ResourceBrowser) ClearSort() {
	r.sortColumn = -1
	r.sortAscending = true
}

// getSortIndicator returns the sort indicator for a column header
func (r *ResourceBrowser) getSortIndicator(colIndex int) string {
	if r.sortColumn != colIndex {
		return ""
	}
	if r.sortAscending {
		return " ▲"
	}
	return " ▼"
}

// FindColumnByName finds a column index by partial name match (case-insensitive)
func (r *ResourceBrowser) FindColumnByName(name string) int {
	if r.renderer == nil {
		return -1
	}

	cols := r.renderer.Columns()
	name = strings.ToLower(strings.TrimSpace(name))

	// First try exact match
	for i, col := range cols {
		if strings.ToLower(col.Name) == name {
			return i
		}
	}

	// Then try prefix match
	for i, col := range cols {
		if strings.HasPrefix(strings.ToLower(col.Name), name) {
			return i
		}
	}

	// Then try contains match
	for i, col := range cols {
		if strings.Contains(strings.ToLower(col.Name), name) {
			return i
		}
	}

	return -1
}
