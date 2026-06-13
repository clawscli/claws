package view

import (
	"fmt"
	"reflect"
	"strings"

	appaws "github.com/clawscli/claws/internal/aws"
	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/filter"
	"github.com/clawscli/claws/internal/render"
)

// applyFilter filters resources based on current filter settings
func (r *ResourceBrowser) applyFilter() {
	// Start with all resources
	working := r.resources

	// Apply field-based filter first (from navigation)
	if r.fieldFilter != "" && r.fieldFilterValue != "" {
		var fieldFiltered []dao.Resource
		for _, res := range working {
			if r.matchesFieldFilter(res) {
				fieldFiltered = append(fieldFiltered, res)
			}
		}
		working = fieldFiltered
	}

	// Apply tag filters (from startup flags and :tag command)
	if len(r.tagFilters) > 0 {
		var tagFiltered []dao.Resource
		for _, res := range working {
			if r.matchesTagFilters(res) {
				tagFiltered = append(tagFiltered, res)
			}
		}
		working = tagFiltered
	}

	// Then apply text filter
	if r.filterText == "" {
		r.filtered = working
		r.applySorting()
		return
	}

	r.filtered = nil

	// Regular text filter (fuzzy match across all columns)
	filterLower := strings.ToLower(r.filterText)

	// Get columns from renderer
	var cols []render.Column
	if r.renderer != nil {
		cols = r.renderer.Columns()
	}

	for _, res := range working {
		// Match against all visible columns
		if r.matchesFilter(res, cols, filterLower) {
			r.filtered = append(r.filtered, res)
		}
	}

	r.applySorting()

	// Clear mark if marked resource is no longer in filtered list
	if r.markedResource != nil {
		found := false
		for _, res := range r.filtered {
			if res.GetID() == r.markedResource.GetID() {
				found = true
				break
			}
		}
		if !found {
			r.markedResource = nil
		}
	}
}

// matchesTagFilters checks if a resource matches all active tag filters.
func (r *ResourceBrowser) matchesTagFilters(res dao.Resource) bool {
	return filter.MatchesTagFilters(res.GetTags(), r.tagFilters)
}

// matchesFieldFilter checks if a resource matches the field-based filter
func (r *ResourceBrowser) matchesFieldFilter(res dao.Resource) bool {
	filterValue := r.fieldFilterValue

	// First, try matching by ID or Name with the original filter value
	// This handles cases where ID is the full ARN (e.g., LoadBalancer, StateMachine)
	if res.GetID() == filterValue || res.GetName() == filterValue {
		return true
	}

	// Extract resource name from ARN if the filter value is an ARN
	// e.g., "arn:aws:iam::123456789012:role/MyRole" -> "MyRole"
	// This handles cases where ID is the resource name (e.g., IAM Role)
	if strings.HasPrefix(filterValue, "arn:aws:") {
		extractedName := appaws.ExtractResourceName(filterValue)
		if res.GetID() == extractedName || res.GetName() == extractedName {
			return true
		}
	}

	// Then try field-based matching using reflection
	data := res.Raw()
	if data == nil {
		// Some DAOs apply context filters server-side and don't expose the filter field in Raw.
		return true
	}

	fieldValue, found := getFieldValue(data, r.fieldFilter)

	if !found {
		// Preserve server-side filtered resources when the SDK shape uses a different field name.
		return true
	}

	return fieldValue == filterValue
}

// matchesFilter checks if a resource matches the text filter
func (r *ResourceBrowser) matchesFilter(res dao.Resource, cols []render.Column, filter string) bool {
	// Always check ID and Name as fallback (fuzzy match)
	if fuzzyMatch(res.GetID(), filter) || fuzzyMatch(res.GetName(), filter) {
		return true
	}

	unwrapped := dao.UnwrapResource(res)

	// Check all column values (fuzzy match)
	for _, col := range cols {
		if col.Getter != nil {
			if fuzzyMatch(col.Getter(unwrapped), filter) {
				return true
			}
		}
	}

	return false
}

// getFieldValue extracts a field value from an AWS resource using reflection
func getFieldValue(data any, fieldName string) (string, bool) {
	if data == nil {
		return "", false
	}

	v := reflect.ValueOf(data)

	// Handle pointer
	if v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return "", false
		}
		v = v.Elem()
	}

	// Must be a struct
	if v.Kind() != reflect.Struct {
		return "", false
	}

	// Get the field
	field := v.FieldByName(fieldName)
	if !field.IsValid() {
		return "", false
	}

	// Handle pointer fields (common in AWS SDK)
	if field.Kind() == reflect.Ptr {
		if field.IsNil() {
			return "", true
		}
		field = field.Elem()
	}

	// Return string representation
	switch field.Kind() {
	case reflect.String:
		return field.String(), true
	case reflect.Int, reflect.Int32, reflect.Int64:
		return fmt.Sprintf("%d", field.Int()), true
	case reflect.Bool:
		return fmt.Sprintf("%v", field.Bool()), true
	default:
		return fmt.Sprintf("%v", field.Interface()), true
	}
}
