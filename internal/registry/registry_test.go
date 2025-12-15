package registry

import (
	"context"
	"testing"

	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/render"
)

func TestServiceResource_String(t *testing.T) {
	sr := ServiceResource{Service: "ec2", Resource: "instances"}
	want := "ec2/instances"
	if got := sr.String(); got != want {
		t.Errorf("String() = %q, want %q", got, want)
	}
}

func TestRegistry_RegisterAndGet(t *testing.T) {
	reg := New()

	// Create mock factories
	daoFactory := func(ctx context.Context) (dao.DAO, error) {
		return nil, nil
	}
	rendererFactory := func() render.Renderer {
		return nil
	}

	entry := Entry{
		DAOFactory:      daoFactory,
		RendererFactory: rendererFactory,
	}

	// Register custom
	reg.RegisterCustom("ec2", "instances", entry)

	// Get should return the entry
	got, ok := reg.Get("ec2", "instances")
	if !ok {
		t.Fatal("Get() returned false, want true")
	}
	if got.DAOFactory == nil {
		t.Error("DAOFactory should not be nil")
	}
	if got.RendererFactory == nil {
		t.Error("RendererFactory should not be nil")
	}

	// Non-existent should return false
	_, ok = reg.Get("nonexistent", "resource")
	if ok {
		t.Error("Get() for nonexistent should return false")
	}
}

func TestRegistry_Priority(t *testing.T) {
	reg := New()

	customCalled := false
	generatedCalled := false

	customEntry := Entry{
		DAOFactory: func(ctx context.Context) (dao.DAO, error) {
			customCalled = true
			return nil, nil
		},
	}

	generatedEntry := Entry{
		DAOFactory: func(ctx context.Context) (dao.DAO, error) {
			generatedCalled = true
			return nil, nil
		},
	}

	// Register generated first, then custom
	reg.RegisterGenerated("ec2", "instances", generatedEntry)
	reg.RegisterCustom("ec2", "instances", customEntry)

	// Get should return custom (higher priority)
	entry, ok := reg.Get("ec2", "instances")
	if !ok {
		t.Fatal("Get() returned false")
	}

	// Call the factory
	_, _ = entry.DAOFactory(context.Background())

	if !customCalled {
		t.Error("custom factory should have been called")
	}
	if generatedCalled {
		t.Error("generated factory should not have been called")
	}
}

func TestRegistry_GeneratedOnly(t *testing.T) {
	reg := New()

	generatedCalled := false
	generatedEntry := Entry{
		DAOFactory: func(ctx context.Context) (dao.DAO, error) {
			generatedCalled = true
			return nil, nil
		},
	}

	// Register only generated
	reg.RegisterGenerated("ec2", "volumes", generatedEntry)

	// Get should return generated when no custom exists
	entry, ok := reg.Get("ec2", "volumes")
	if !ok {
		t.Fatal("Get() returned false")
	}

	_, _ = entry.DAOFactory(context.Background())

	if !generatedCalled {
		t.Error("generated factory should have been called")
	}
}

func TestRegistry_ListServices(t *testing.T) {
	reg := New()

	reg.RegisterCustom("ec2", "instances", Entry{})
	reg.RegisterCustom("s3", "buckets", Entry{})
	reg.RegisterCustom("iam", "roles", Entry{})

	services := reg.ListServices()

	if len(services) != 3 {
		t.Errorf("ListServices() returned %d services, want 3", len(services))
	}

	// Check all services are present (order not guaranteed)
	serviceMap := make(map[string]bool)
	for _, svc := range services {
		serviceMap[svc] = true
	}

	for _, expected := range []string{"ec2", "s3", "iam"} {
		if !serviceMap[expected] {
			t.Errorf("ListServices() missing %q", expected)
		}
	}
}

func TestRegistry_ListResources(t *testing.T) {
	reg := New()

	reg.RegisterCustom("ec2", "instances", Entry{})
	reg.RegisterCustom("ec2", "volumes", Entry{})
	reg.RegisterCustom("ec2", "security-groups", Entry{})

	resources := reg.ListResources("ec2")

	if len(resources) != 3 {
		t.Errorf("ListResources() returned %d resources, want 3", len(resources))
	}

	// Should be sorted
	expected := []string{"instances", "security-groups", "volumes"}
	for i, want := range expected {
		if resources[i] != want {
			t.Errorf("ListResources()[%d] = %q, want %q", i, resources[i], want)
		}
	}
}

func TestRegistry_ListResources_ExcludesSubResources(t *testing.T) {
	reg := New()

	reg.RegisterCustom("cloudformation", "stacks", Entry{})
	reg.RegisterCustom("cloudformation", "events", Entry{})    // sub-resource
	reg.RegisterCustom("cloudformation", "resources", Entry{}) // sub-resource

	resources := reg.ListResources("cloudformation")

	// Should only return "stacks", excluding sub-resources
	if len(resources) != 1 {
		t.Errorf("ListResources() returned %d resources, want 1", len(resources))
	}
	if resources[0] != "stacks" {
		t.Errorf("ListResources()[0] = %q, want %q", resources[0], "stacks")
	}
}

func TestRegistry_ResolveAlias(t *testing.T) {
	reg := New()

	tests := []struct {
		input        string
		wantService  string
		wantResource string
		wantFound    bool
	}{
		{"cfn", "cloudformation", "", true},
		{"cf", "cloudformation", "", true},
		{"sg", "ec2", "security-groups", true},
		{"ec2", "ec2", "", false}, // not an alias
		{"unknown", "unknown", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			service, resource, found := reg.ResolveAlias(tt.input)
			if service != tt.wantService {
				t.Errorf("service = %q, want %q", service, tt.wantService)
			}
			if resource != tt.wantResource {
				t.Errorf("resource = %q, want %q", resource, tt.wantResource)
			}
			if found != tt.wantFound {
				t.Errorf("found = %v, want %v", found, tt.wantFound)
			}
		})
	}
}

func TestRegistry_GetAliasesForService(t *testing.T) {
	reg := New()

	aliases := reg.GetAliasesForService("cloudformation")

	// Should include cf and cfn (both are aliases for cloudformation)
	if len(aliases) != 2 {
		t.Errorf("GetAliasesForService() returned %d aliases, want 2", len(aliases))
	}

	// Check both aliases are present (order may vary)
	hasAliases := make(map[string]bool)
	for _, a := range aliases {
		hasAliases[a] = true
	}
	if !hasAliases["cf"] || !hasAliases["cfn"] {
		t.Errorf("GetAliasesForService() = %v, want [cf, cfn]", aliases)
	}
}

func TestRegistry_GetAliasesForService_WithResourceAlias(t *testing.T) {
	reg := New()

	// sg maps to "ec2/security-groups"
	aliases := reg.GetAliasesForService("ec2")

	// Should include sg
	found := false
	for _, alias := range aliases {
		if alias == "sg" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("GetAliasesForService(ec2) should include 'sg', got %v", aliases)
	}
}

func TestRegistry_GetDAO_NotRegistered(t *testing.T) {
	reg := New()

	_, err := reg.GetDAO(context.Background(), "nonexistent", "resource")
	if err == nil {
		t.Error("GetDAO() should return error for unregistered service/resource")
	}
}

func TestRegistry_GetDAO_NilFactory(t *testing.T) {
	reg := New()

	// Register with nil factory
	reg.RegisterCustom("test", "resource", Entry{DAOFactory: nil})

	_, err := reg.GetDAO(context.Background(), "test", "resource")
	if err == nil {
		t.Error("GetDAO() should return error for nil factory")
	}
}

func TestRegistry_GetRenderer_NotRegistered(t *testing.T) {
	reg := New()

	_, err := reg.GetRenderer("nonexistent", "resource")
	if err == nil {
		t.Error("GetRenderer() should return error for unregistered service/resource")
	}
}

func TestRegistry_GetRenderer_NilFactory(t *testing.T) {
	reg := New()

	// Register with nil factory
	reg.RegisterCustom("test", "resource", Entry{RendererFactory: nil})

	_, err := reg.GetRenderer("test", "resource")
	if err == nil {
		t.Error("GetRenderer() should return error for nil factory")
	}
}

func TestGlobalRegistry(t *testing.T) {
	// Global registry should be initialized
	if Global == nil {
		t.Fatal("Global registry should not be nil")
	}
}

func TestIsSubResource(t *testing.T) {
	tests := []struct {
		service  string
		resource string
		want     bool
	}{
		{"cloudformation", "events", true},
		{"cloudformation", "resources", true},
		{"cloudformation", "outputs", true},
		{"cloudwatch", "log-streams", true},
		{"service-quotas", "quotas", true},
		{"guardduty", "findings", true},
		{"inspector2", "findings", false}, // inspector2/findings is NOT a sub-resource
		{"cloudformation", "stacks", false},
		{"ec2", "instances", false},
		{"s3", "buckets", false},
		{"cloudwatch", "log-groups", false},
		{"service-quotas", "services", false},
	}

	for _, tt := range tests {
		t.Run(tt.service+"/"+tt.resource, func(t *testing.T) {
			if got := isSubResource(tt.service, tt.resource); got != tt.want {
				t.Errorf("isSubResource(%q, %q) = %v, want %v", tt.service, tt.resource, got, tt.want)
			}
		})
	}
}

func TestRegistry_AddServiceDeduplication(t *testing.T) {
	reg := New()

	// Register same resource multiple times
	reg.RegisterCustom("ec2", "instances", Entry{})
	reg.RegisterCustom("ec2", "instances", Entry{})
	reg.RegisterGenerated("ec2", "instances", Entry{})

	resources := reg.ListResources("ec2")

	// Should only appear once
	if len(resources) != 1 {
		t.Errorf("ListResources() returned %d resources, want 1 (should deduplicate)", len(resources))
	}
}
