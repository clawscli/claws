package dao

import (
	"context"
	"testing"
)

// TestWrapWithRegion verifies that region wrapping preserves resource data
func TestWrapWithRegion(t *testing.T) {
	original := &BaseResource{
		ID:   "test-id",
		Name: "test-name",
		ARN:  "arn:aws:service:region:account:resource",
		Tags: map[string]string{"key": "value"},
		Data: map[string]interface{}{"field": "data"},
	}

	region := "us-west-2"
	wrapped := WrapWithRegion(original, region)

	// Verify wrapper properties
	if wrapped.Region != region {
		t.Errorf("Region mismatch: got %q, want %q", wrapped.Region, region)
	}

	// Verify original resource is preserved
	if wrapped.Resource != original {
		t.Error("Original resource not preserved in wrapper")
	}

	// Verify GetRegion works
	if wrapped.GetRegion() != region {
		t.Errorf("GetRegion() mismatch: got %q, want %q", wrapped.GetRegion(), region)
	}

	// Verify ID is region-qualified
	expectedID := "us-west-2:test-id"
	if wrapped.GetID() != expectedID {
		t.Errorf("GetID() mismatch: got %q, want %q", wrapped.GetID(), expectedID)
	}

	// Verify other properties are delegated correctly
	if wrapped.GetName() != "test-name" {
		t.Errorf("GetName() mismatch: got %q, want %q", wrapped.GetName(), "test-name")
	}

	if wrapped.GetARN() != "arn:aws:service:region:account:resource" {
		t.Errorf("GetARN() mismatch")
	}

	tags := wrapped.GetTags()
	if tags["key"] != "value" {
		t.Errorf("GetTags() mismatch")
	}
}

// TestGetResourceRegion extracts region from regional resources
func TestGetResourceRegion(t *testing.T) {
	original := &BaseResource{ID: "test-id", Name: "test"}

	// Test with regional resource
	regional := WrapWithRegion(original, "eu-west-1")
	if GetResourceRegion(regional) != "eu-west-1" {
		t.Errorf("GetResourceRegion() should return region from wrapped resource")
	}

	// Test with non-regional resource
	if GetResourceRegion(original) != "" {
		t.Errorf("GetResourceRegion() should return empty string for non-regional resource")
	}
}

// TestUnwrapResource unwraps regional resources back to original
func TestUnwrapResource(t *testing.T) {
	original := &BaseResource{ID: "test-id", Name: "test"}
	wrapped := WrapWithRegion(original, "ap-southeast-1")

	// Test unwrapping
	unwrapped := UnwrapResource(wrapped)
	if unwrapped != original {
		t.Error("UnwrapResource() did not return original resource")
	}

	// Test unwrapping non-wrapped resource
	if UnwrapResource(original) != original {
		t.Error("UnwrapResource() should return same resource for non-wrapped")
	}
}

// TestMultipleRegions verifies different regions create separate qualified IDs
func TestMultipleRegions(t *testing.T) {
	original := &BaseResource{ID: "same-id"}

	regions := []string{"us-east-1", "us-west-2", "eu-west-1"}
	ids := make([]string, len(regions))

	for i, region := range regions {
		wrapped := WrapWithRegion(original, region)
		ids[i] = wrapped.GetID()
	}

	// Verify all IDs are unique
	for i := 0; i < len(ids); i++ {
		for j := i + 1; j < len(ids); j++ {
			if ids[i] == ids[j] {
				t.Errorf("IDs should be unique: %q == %q", ids[i], ids[j])
			}
		}
	}

	// Verify format
	for i, id := range ids {
		expectedPrefix := regions[i] + ":"
		if id != expectedPrefix+"same-id" {
			t.Errorf("ID format incorrect: got %q, want %q*", id, expectedPrefix)
		}
	}
}

// MockDAO provides a simple DAO for testing
type MockDAO struct {
	BaseDAO
	resources []Resource
}

func (m *MockDAO) List(ctx context.Context) ([]Resource, error) {
	return m.resources, nil
}

func (m *MockDAO) Get(ctx context.Context, id string) (Resource, error) {
	for _, res := range m.resources {
		if res.GetID() == id {
			return res, nil
		}
	}
	return nil, nil
}

func (m *MockDAO) Delete(ctx context.Context, id string) error {
	return nil
}

// CustomResource is a mock concrete resource type for testing type assertions
type CustomResource struct {
	BaseResource
	CustomField string
}

func (c *CustomResource) GetCustomField() string {
	return c.CustomField
}

// TestUnwrapResourceTypeAssertion verifies type assertion works after unwrapping
func TestUnwrapResourceTypeAssertion(t *testing.T) {
	original := &CustomResource{
		BaseResource: BaseResource{ID: "test-id", Name: "test-name"},
		CustomField:  "custom-value",
	}

	wrapped := WrapWithRegion(original, "us-east-1")
	unwrapped := UnwrapResource(wrapped)

	// Type assertion to concrete type should work
	custom, ok := unwrapped.(*CustomResource)
	if !ok {
		t.Fatalf("Type assertion to *CustomResource failed after unwrap. Got type: %T", unwrapped)
	}

	if custom.GetCustomField() != "custom-value" {
		t.Errorf("CustomField mismatch: got %q, want %q", custom.GetCustomField(), "custom-value")
	}

	if custom.GetID() != "test-id" {
		t.Errorf("ID mismatch: got %q, want %q", custom.GetID(), "test-id")
	}
}

// TestColumnGetterWithWrappedResource simulates renderer column getter pattern
func TestColumnGetterWithWrappedResource(t *testing.T) {
	original := &CustomResource{
		BaseResource: BaseResource{ID: "test-id", Name: "test-name"},
		CustomField:  "expected-value",
	}

	getter := func(r Resource) string {
		if cr, ok := r.(*CustomResource); ok {
			return cr.GetCustomField()
		}
		return ""
	}

	// Single region: direct resource
	result := getter(original)
	if result != "expected-value" {
		t.Errorf("Direct resource getter failed: got %q, want %q", result, "expected-value")
	}

	// Multi-region: wrapped then unwrapped
	wrapped := WrapWithRegion(original, "us-east-1")
	unwrapped := UnwrapResource(wrapped)
	result = getter(unwrapped)
	if result != "expected-value" {
		t.Errorf("Unwrapped resource getter failed: got %q, want %q. Type: %T", result, "expected-value", unwrapped)
	}

	// Verify wrapped resource WITHOUT unwrap fails (this is expected)
	result = getter(wrapped)
	if result != "" {
		t.Errorf("Wrapped resource should NOT match type assertion, got %q", result)
	}
}

// TestDoubleWrappingBreaksTypeAssertion verifies double wrapping causes type assertion failure
// This test documents the bug that occurred and prevents regression
func TestDoubleWrappingBreaksTypeAssertion(t *testing.T) {
	original := &CustomResource{
		BaseResource: BaseResource{ID: "test-id", Name: "test-name"},
		CustomField:  "custom-value",
	}

	getter := func(r Resource) string {
		if cr, ok := r.(*CustomResource); ok {
			return cr.GetCustomField()
		}
		return ""
	}

	singleWrapped := WrapWithRegion(original, "us-east-1")
	singleUnwrapped := UnwrapResource(singleWrapped)
	if result := getter(singleUnwrapped); result != "custom-value" {
		t.Errorf("Single wrap/unwrap should work: got %q, want %q", result, "custom-value")
	}

	doubleWrapped := WrapWithRegion(singleWrapped, "us-east-1")
	doubleUnwrapped := UnwrapResource(doubleWrapped)
	if result := getter(doubleUnwrapped); result != "" {
		t.Errorf("Double wrapped resources break after single unwrap - this test documents the bug")
	}

	fullyUnwrapped := UnwrapResource(UnwrapResource(doubleWrapped))
	if result := getter(fullyUnwrapped); result != "custom-value" {
		t.Errorf("Double unwrap should recover original: got %q, want %q", result, "custom-value")
	}
}

// TestRegionalResourcePreservesData verifies region wrapping doesn't lose data
func TestRegionalResourcePreservesData(t *testing.T) {
	resources := []Resource{
		&BaseResource{
			ID:   "res-1",
			Name: "resource-1",
			ARN:  "arn:aws:service:region:123456789:resource/res-1",
			Tags: map[string]string{"Env": "prod"},
		},
		&BaseResource{
			ID:   "res-2",
			Name: "resource-2",
			ARN:  "arn:aws:service:region:123456789:resource/res-2",
			Tags: map[string]string{"Env": "dev"},
		},
	}

	region := "us-west-2"
	wrapped := make([]Resource, len(resources))
	for i, res := range resources {
		wrapped[i] = WrapWithRegion(res, region)
	}

	// Verify all data is preserved through wrapping/unwrapping
	for i := range resources {
		unwrapped := UnwrapResource(wrapped[i])
		if unwrapped.GetName() != resources[i].GetName() {
			t.Errorf("Name not preserved: got %q, want %q",
				unwrapped.GetName(), resources[i].GetName())
		}
		if unwrapped.GetARN() != resources[i].GetARN() {
			t.Errorf("ARN not preserved")
		}
		tags := unwrapped.GetTags()
		origTags := resources[i].GetTags()
		for k := range origTags {
			if tags[k] != origTags[k] {
				t.Errorf("Tag not preserved: %s", k)
			}
		}
	}
}
