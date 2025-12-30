package registry

import (
	"context"
	"testing"

	"github.com/clawscli/claws/internal/aws"
	"github.com/clawscli/claws/internal/dao"
)

// MockDAO for testing wrapper functionality
type MockDAO struct {
	dao.BaseDAO
	resources []dao.Resource
	lastGetID string
	lastDelID string
}

func NewMockDAO() *MockDAO {
	return &MockDAO{
		BaseDAO: dao.NewBaseDAO("test", "resources"),
		resources: []dao.Resource{
			&dao.BaseResource{
				ID:   "res-1",
				Name: "resource-1",
				ARN:  "arn:test",
				Tags: map[string]string{},
			},
			&dao.BaseResource{
				ID:   "res-2",
				Name: "resource-2",
				ARN:  "arn:test",
				Tags: map[string]string{},
			},
		},
	}
}

func (m *MockDAO) List(ctx context.Context) ([]dao.Resource, error) {
	return m.resources, nil
}

func (m *MockDAO) Get(ctx context.Context, id string) (dao.Resource, error) {
	m.lastGetID = id
	for _, res := range m.resources {
		if res.GetID() == id {
			return res, nil
		}
	}
	return nil, nil
}

func (m *MockDAO) Delete(ctx context.Context, id string) error {
	m.lastDelID = id
	return nil
}

// TestRegionalDAOWrapperNoRegion returns unwrapped DAO without region override
func TestRegionalDAOWrapperNoRegion(t *testing.T) {
	ctx := context.Background()
	mockDAO := NewMockDAO()

	wrapper := NewRegionalDAOWrapper(ctx, mockDAO)

	// Without region override, should return the delegate unwrapped
	if wrapper != mockDAO {
		t.Error("Wrapper should return unwrapped DAO without region override")
	}
}

// TestRegionalDAOWrapperWithRegion wraps DAO when region override is present
func TestRegionalDAOWrapperWithRegion(t *testing.T) {
	mockDAO := NewMockDAO()
	region := "us-west-2"
	ctx := aws.WithRegionOverride(context.Background(), region)

	wrapper := NewRegionalDAOWrapper(ctx, mockDAO)

	// Should return a wrapper, not the delegate
	if wrapper == mockDAO {
		t.Error("Wrapper should return wrapped DAO with region override")
	}

	// Verify wrapper is the correct type
	_, ok := wrapper.(*RegionalDAOWrapper)
	if !ok {
		t.Error("Wrapper should be RegionalDAOWrapper type")
	}
}

// TestRegionalDAOWrapperListWrapsResources verifies List wraps all resources
func TestRegionalDAOWrapperListWrapsResources(t *testing.T) {
	mockDAO := NewMockDAO()
	region := "eu-west-1"
	ctx := aws.WithRegionOverride(context.Background(), region)

	wrapper := NewRegionalDAOWrapper(ctx, mockDAO).(*RegionalDAOWrapper)
	resources, err := wrapper.List(context.Background())

	if err != nil {
		t.Fatalf("List() error: %v", err)
	}

	if len(resources) != 2 {
		t.Errorf("List() should return 2 resources, got %d", len(resources))
	}

	// Verify resources are wrapped with region
	for i, res := range resources {
		if dao.GetResourceRegion(res) != region {
			t.Errorf("Resource %d should have region %q, got %q",
				i, region, dao.GetResourceRegion(res))
		}

		// Verify ID is region-qualified
		expectedPrefix := region + ":"
		if res.GetID()[:len(expectedPrefix)] != expectedPrefix {
			t.Errorf("Resource %d ID should start with %q", i, expectedPrefix)
		}
	}
}

// TestRegionalDAOWrapperGetWrapsResource verifies Get wraps the resource
func TestRegionalDAOWrapperGetWrapsResource(t *testing.T) {
	mockDAO := NewMockDAO()
	region := "ap-southeast-1"
	ctx := aws.WithRegionOverride(context.Background(), region)

	wrapper := NewRegionalDAOWrapper(ctx, mockDAO).(*RegionalDAOWrapper)
	res, err := wrapper.Get(context.Background(), "res-1")

	if err != nil {
		t.Fatalf("Get() error: %v", err)
	}

	if res == nil {
		t.Fatal("Get() returned nil resource")
	}

	// Verify resource is wrapped with region
	if dao.GetResourceRegion(res) != region {
		t.Errorf("Resource should have region %q, got %q",
			region, dao.GetResourceRegion(res))
	}

	// Verify ID is region-qualified
	if res.GetID() != region+":res-1" {
		t.Errorf("Resource ID should be %q, got %q", region+":res-1", res.GetID())
	}

	// Verify original data is preserved
	unwrapped := dao.UnwrapResource(res)
	if unwrapped.GetName() != "resource-1" {
		t.Errorf("Unwrapped resource name should be 'resource-1', got %q", unwrapped.GetName())
	}
}

// TestRegionalDAOWrapperDelegatesMethods verifies other methods delegate correctly
func TestRegionalDAOWrapperDelegatesMethods(t *testing.T) {
	mockDAO := NewMockDAO()
	ctx := aws.WithRegionOverride(context.Background(), "us-east-1")
	wrapper := NewRegionalDAOWrapper(ctx, mockDAO).(*RegionalDAOWrapper)

	// Test ServiceName delegation
	if wrapper.ServiceName() != "test" {
		t.Errorf("ServiceName() should be 'test', got %q", wrapper.ServiceName())
	}

	// Test ResourceType delegation
	if wrapper.ResourceType() != "resources" {
		t.Errorf("ResourceType() should be 'resources', got %q", wrapper.ResourceType())
	}

	// Test Supports delegation
	if !wrapper.Supports(dao.OpList) {
		t.Error("Supports(OpList) should be true")
	}
}

// TestPaginatedDAOWrapperWrapsPages verifies pagination wrapper wraps resources
func TestPaginatedDAOWrapperWrapsPages(t *testing.T) {
	mockDAO := NewMockPaginatedDAO()
	region := "us-west-2"
	ctx := aws.WithRegionOverride(context.Background(), region)

	wrapper := NewPaginatedDAOWrapper(ctx, mockDAO)

	// Without wrapping, verify it's the right type
	if wrapper == mockDAO {
		t.Error("Wrapper should wrap paginated DAO")
	}

	// Verify it's the correct wrapper type
	_, ok := wrapper.(*PaginatedDAOWrapper)
	if !ok {
		t.Error("Wrapper should be PaginatedDAOWrapper type")
	}
}

// MockPaginatedDAO for testing pagination wrapper
type MockPaginatedDAO struct {
	*MockDAO
}

func NewMockPaginatedDAO() *MockPaginatedDAO {
	return &MockPaginatedDAO{MockDAO: NewMockDAO()}
}

func (m *MockPaginatedDAO) ListPage(ctx context.Context, pageSize int, pageToken string) ([]dao.Resource, string, error) {
	// Return first page only for simplicity
	return m.resources, "", nil
}

// TestPaginatedDAOWrapperListPageWrapsResources verifies ListPage wraps resources
func TestPaginatedDAOWrapperListPageWrapsResources(t *testing.T) {
	mockDAO := NewMockPaginatedDAO()
	region := "eu-central-1"
	ctx := aws.WithRegionOverride(context.Background(), region)

	wrapper := NewPaginatedDAOWrapper(ctx, mockDAO).(*PaginatedDAOWrapper)
	resources, nextToken, err := wrapper.ListPage(context.Background(), 10, "")

	if err != nil {
		t.Fatalf("ListPage() error: %v", err)
	}

	if len(resources) != 2 {
		t.Errorf("ListPage() should return 2 resources, got %d", len(resources))
	}

	// Verify resources are wrapped with region
	for i, res := range resources {
		if dao.GetResourceRegion(res) != region {
			t.Errorf("Resource %d should have region %q, got %q",
				i, region, dao.GetResourceRegion(res))
		}
	}

	// Verify page token passed through
	if nextToken != "" {
		t.Errorf("NextToken should be empty, got %q", nextToken)
	}
}

type CustomTestResource struct {
	dao.BaseResource
	CustomField string
}

func TestRegionalDAOWrapperGetStripsRegionPrefix(t *testing.T) {
	mockDAO := NewMockDAO()
	ctx := aws.WithRegionOverride(context.Background(), "us-west-2")
	wrapper := NewRegionalDAOWrapper(ctx, mockDAO).(*RegionalDAOWrapper)

	_, _ = wrapper.Get(ctx, "us-west-2:res-1")

	if mockDAO.lastGetID != "res-1" {
		t.Errorf("Get should strip region prefix: got %q, want %q", mockDAO.lastGetID, "res-1")
	}
}

func TestRegionalDAOWrapperDeleteStripsRegionPrefix(t *testing.T) {
	mockDAO := NewMockDAO()
	ctx := aws.WithRegionOverride(context.Background(), "eu-west-1")
	wrapper := NewRegionalDAOWrapper(ctx, mockDAO).(*RegionalDAOWrapper)

	_ = wrapper.Delete(ctx, "eu-west-1:res-2")

	if mockDAO.lastDelID != "res-2" {
		t.Errorf("Delete should strip region prefix: got %q, want %q", mockDAO.lastDelID, "res-2")
	}
}

func TestRegionalDAOWrapperGetPassesRawID(t *testing.T) {
	mockDAO := NewMockDAO()
	ctx := aws.WithRegionOverride(context.Background(), "us-east-1")
	wrapper := NewRegionalDAOWrapper(ctx, mockDAO).(*RegionalDAOWrapper)

	_, _ = wrapper.Get(ctx, "res-1")

	if mockDAO.lastGetID != "res-1" {
		t.Errorf("Get should pass raw ID unchanged: got %q, want %q", mockDAO.lastGetID, "res-1")
	}
}

func TestRegionalDAOWrapperPreservesARN(t *testing.T) {
	mockDAO := NewMockDAO()
	ctx := aws.WithRegionOverride(context.Background(), "us-east-1")
	wrapper := NewRegionalDAOWrapper(ctx, mockDAO).(*RegionalDAOWrapper)

	tests := []struct {
		name     string
		inputID  string
		expected string
	}{
		{
			name:     "ARN without region prefix passes through unchanged",
			inputID:  "arn:aws:states:us-east-1:123456789012:stateMachine:my-state-machine",
			expected: "arn:aws:states:us-east-1:123456789012:stateMachine:my-state-machine",
		},
		{
			name:     "region-prefixed ARN gets stripped correctly",
			inputID:  "us-east-1:arn:aws:states:us-east-1:123456789012:stateMachine:my-state-machine",
			expected: "arn:aws:states:us-east-1:123456789012:stateMachine:my-state-machine",
		},
		{
			name:     "simple ID without region prefix passes through",
			inputID:  "i-1234567890abcdef0",
			expected: "i-1234567890abcdef0",
		},
		{
			name:     "region-prefixed simple ID gets stripped",
			inputID:  "us-east-1:i-1234567890abcdef0",
			expected: "i-1234567890abcdef0",
		},
		{
			name:     "wrong region prefix not stripped",
			inputID:  "eu-west-1:i-1234567890abcdef0",
			expected: "eu-west-1:i-1234567890abcdef0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _ = wrapper.Get(ctx, tt.inputID)
			if mockDAO.lastGetID != tt.expected {
				t.Errorf("Get(%q): got %q, want %q", tt.inputID, mockDAO.lastGetID, tt.expected)
			}
		})
	}
}

func TestRegionalDAOWrapperSingleWrapTypeAssertion(t *testing.T) {
	customRes := &CustomTestResource{
		BaseResource: dao.BaseResource{ID: "custom-1", Name: "custom"},
		CustomField:  "test-value",
	}
	mockDAO := &MockDAO{
		BaseDAO:   dao.NewBaseDAO("test", "resources"),
		resources: []dao.Resource{customRes},
	}

	ctx := aws.WithRegionOverride(context.Background(), "us-east-1")
	wrapper := NewRegionalDAOWrapper(ctx, mockDAO).(*RegionalDAOWrapper)
	resources, _ := wrapper.List(ctx)

	if len(resources) != 1 {
		t.Fatalf("Expected 1 resource, got %d", len(resources))
	}

	unwrapped := dao.UnwrapResource(resources[0])
	custom, ok := unwrapped.(*CustomTestResource)
	if !ok {
		t.Fatalf("Type assertion failed after single unwrap. Got: %T", unwrapped)
	}
	if custom.CustomField != "test-value" {
		t.Errorf("CustomField = %q, want %q", custom.CustomField, "test-value")
	}
}
