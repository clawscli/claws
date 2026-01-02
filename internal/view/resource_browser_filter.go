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

	// Apply tag filter (from :tag command)
	if r.tagFilterText != "" {
		var tagFiltered []dao.Resource
		for _, res := range working {
			if r.matchesTagFilter(res, r.tagFilterText) {
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

// matchesTagFilter checks if a resource matches the tag filter.
func (r *ResourceBrowser) matchesTagFilter(res dao.Resource, tagFilter string) bool {
	return filter.MatchesTagFilter(res.GetTags(), tagFilter)
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
		// No raw data - assume DAO already filtered
		return true
	}

	// Try to get the field value using the getter interface
	fieldValue := getFieldValue(data, r.fieldFilter)

	// If field not found (empty string), assume DAO already filtered correctly
	// This handles cases like ECS where DAO uses "ClusterName" context filter
	// but the actual struct has "ClusterArn" field
	if fieldValue == "" {
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
func getFieldValue(data any, fieldName string) string {
	if data == nil {
		return ""
	}

	v := reflect.ValueOf(data)

	// Handle pointer
	if v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return ""
		}
		v = v.Elem()
	}

	// Must be a struct
	if v.Kind() != reflect.Struct {
		return ""
	}

	// Get the field
	field := v.FieldByName(fieldName)
	if !field.IsValid() {
		return ""
	}

	// Handle pointer fields (common in AWS SDK)
	if field.Kind() == reflect.Ptr {
		if field.IsNil() {
			return ""
		}
		field = field.Elem()
	}

	// Return string representation
	switch field.Kind() {
	case reflect.String:
		return field.String()
	case reflect.Int, reflect.Int32, reflect.Int64:
		return fmt.Sprintf("%d", field.Int())
	case reflect.Bool:
		return fmt.Sprintf("%v", field.Bool())
	default:
		return fmt.Sprintf("%v", field.Interface())
	}
}
