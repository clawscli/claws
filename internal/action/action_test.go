package action

import (
	"context"
	"testing"

	"github.com/clawscli/claws/internal/dao"
)

// mockResource implements dao.Resource for testing
type mockResource struct {
	id   string
	name string
	arn  string
	tags map[string]string
}

func (m *mockResource) GetID() string              { return m.id }
func (m *mockResource) GetName() string            { return m.name }
func (m *mockResource) GetARN() string             { return m.arn }
func (m *mockResource) GetTags() map[string]string { return m.tags }
func (m *mockResource) Raw() any                   { return nil }

// mockResourceWithPrivateIP implements dao.Resource with PrivateIP method
type mockResourceWithPrivateIP struct {
	mockResource
	privateIP string
}

func (m *mockResourceWithPrivateIP) PrivateIP() string { return m.privateIP }

func TestExpandVariables(t *testing.T) {
	resource := &mockResource{
		id:   "i-1234567890abcdef0",
		name: "test-instance",
		arn:  "arn:aws:ec2:us-east-1:123456789012:instance/i-1234567890abcdef0",
	}

	tests := []struct {
		name     string
		cmd      string
		expected string
	}{
		{
			name:     "expand ID",
			cmd:      "aws ec2 describe-instances --instance-ids ${ID}",
			expected: "aws ec2 describe-instances --instance-ids i-1234567890abcdef0",
		},
		{
			name:     "expand NAME",
			cmd:      "echo ${NAME}",
			expected: "echo test-instance",
		},
		{
			name:     "expand ARN",
			cmd:      "aws iam get-role --role-arn ${ARN}",
			expected: "aws iam get-role --role-arn arn:aws:ec2:us-east-1:123456789012:instance/i-1234567890abcdef0",
		},
		{
			name:     "expand INSTANCE_ID",
			cmd:      "ssh ec2-user@${INSTANCE_ID}",
			expected: "ssh ec2-user@i-1234567890abcdef0",
		},
		{
			name:     "expand BUCKET",
			cmd:      "aws s3 ls s3://${BUCKET}",
			expected: "aws s3 ls s3://i-1234567890abcdef0",
		},
		{
			name:     "expand multiple variables",
			cmd:      "${ID} - ${NAME}",
			expected: "i-1234567890abcdef0 - test-instance",
		},
		{
			name:     "no variables",
			cmd:      "echo hello",
			expected: "echo hello",
		},
		{
			name:     "unknown variable stays unchanged",
			cmd:      "echo ${UNKNOWN}",
			expected: "echo ${UNKNOWN}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ExpandVariables(tt.cmd, resource)
			if err != nil {
				t.Errorf("ExpandVariables(%q) returned unexpected error: %v", tt.cmd, err)
			}
			if result != tt.expected {
				t.Errorf("ExpandVariables(%q) = %q, want %q", tt.cmd, result, tt.expected)
			}
		})
	}
}

func TestExpandVariables_WithPrivateIP(t *testing.T) {
	resource := &mockResourceWithPrivateIP{
		mockResource: mockResource{
			id:   "i-1234567890abcdef0",
			name: "test-instance",
			arn:  "arn:aws:ec2:us-east-1:123456789012:instance/i-1234567890abcdef0",
		},
		privateIP: "10.0.1.100",
	}

	cmd := "ssh ec2-user@${PRIVATE_IP}"
	expected := "ssh ec2-user@10.0.1.100"

	result, err := ExpandVariables(cmd, resource)
	if err != nil {
		t.Errorf("ExpandVariables(%q) returned unexpected error: %v", cmd, err)
	}
	if result != expected {
		t.Errorf("ExpandVariables(%q) = %q, want %q", cmd, result, expected)
	}
}

func TestExpandVariables_UnsafeCharacters(t *testing.T) {
	tests := []struct {
		name     string
		resource *mockResource
		cmd      string
		wantErr  bool
	}{
		{
			name:     "semicolon in ID",
			resource: &mockResource{id: "test; rm -rf /"},
			cmd:      "echo ${ID}",
			wantErr:  true,
		},
		{
			name:     "pipe in name",
			resource: &mockResource{name: "test | cat /etc/passwd"},
			cmd:      "echo ${NAME}",
			wantErr:  true,
		},
		{
			name:     "ampersand in ID",
			resource: &mockResource{id: "test && whoami"},
			cmd:      "echo ${ID}",
			wantErr:  true,
		},
		{
			name:     "dollar sign in ID",
			resource: &mockResource{id: "test$HOME"},
			cmd:      "echo ${ID}",
			wantErr:  true,
		},
		{
			name:     "backtick in ID",
			resource: &mockResource{id: "test`whoami`"},
			cmd:      "echo ${ID}",
			wantErr:  true,
		},
		{
			name:     "newline in ID",
			resource: &mockResource{id: "test\nrm -rf /"},
			cmd:      "echo ${ID}",
			wantErr:  true,
		},
		{
			name:     "safe characters",
			resource: &mockResource{id: "i-1234567890abcdef0", name: "my-instance_01"},
			cmd:      "echo ${ID} ${NAME}",
			wantErr:  false,
		},
		{
			name:     "unsafe in unused variable",
			resource: &mockResource{id: "safe-id", name: "bad; rm"},
			cmd:      "echo ${ID}",
			wantErr:  false, // NAME is not used in cmd
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ExpandVariables(tt.cmd, tt.resource)
			if tt.wantErr && err == nil {
				t.Error("ExpandVariables() expected error but got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("ExpandVariables() unexpected error: %v", err)
			}
		})
	}
}

func TestContainsShellMetachar(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"hello", false},
		{"hello-world_123", false},
		{"arn:aws:s3:::bucket", false},
		{"test;rm", true},
		{"test|cat", true},
		{"test&bg", true},
		{"test$var", true},
		{"test`cmd`", true},
		{"test(group)", true},
		{"test{brace}", true},
		{"test<in", true},
		{"test>out", true},
		{"test\ncmd", true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := containsShellMetachar(tt.input)
			if result != tt.expected {
				t.Errorf("containsShellMetachar(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestRegistry(t *testing.T) {
	registry := NewRegistry()

	// Test Register and Get
	actions := []Action{
		{Name: "Stop", Shortcut: "s", Type: ActionTypeAPI, Operation: "StopInstances"},
		{Name: "Start", Shortcut: "S", Type: ActionTypeAPI, Operation: "StartInstances"},
	}

	registry.Register("ec2", "instances", actions)

	got := registry.Get("ec2", "instances")
	if len(got) != 2 {
		t.Errorf("Get() returned %d actions, want 2", len(got))
	}
	if got[0].Name != "Stop" {
		t.Errorf("Get()[0].Name = %q, want %q", got[0].Name, "Stop")
	}

	// Test non-existent key
	got = registry.Get("ec2", "nonexistent")
	if got != nil {
		t.Errorf("Get() for nonexistent key should return nil, got %v", got)
	}
}

func TestRegistry_Executor(t *testing.T) {
	registry := NewRegistry()

	called := false
	executor := func(ctx context.Context, action Action, resource dao.Resource) ActionResult {
		called = true
		return ActionResult{Success: true, Message: "executed"}
	}

	registry.RegisterExecutor("ec2", "instances", executor)

	got := registry.GetExecutor("ec2", "instances")
	if got == nil {
		t.Fatal("GetExecutor() returned nil")
	}

	// Call the executor
	result := got(context.Background(), Action{}, nil)
	if !called {
		t.Error("executor was not called")
	}
	if !result.Success {
		t.Error("executor result should be success")
	}

	// Test non-existent executor
	got = registry.GetExecutor("ec2", "nonexistent")
	if got != nil {
		t.Error("GetExecutor() for nonexistent key should return nil")
	}
}

func TestActionResult(t *testing.T) {
	// Success result
	success := ActionResult{Success: true, Message: "done"}
	if !success.Success {
		t.Error("Success should be true")
	}
	if success.Message != "done" {
		t.Errorf("Message = %q, want %q", success.Message, "done")
	}

	// Error result
	failure := ActionResult{Success: false, Error: ErrEmptyCommand}
	if failure.Success {
		t.Error("Success should be false")
	}
	if failure.Error != ErrEmptyCommand {
		t.Errorf("Error = %v, want %v", failure.Error, ErrEmptyCommand)
	}
}

func TestActionType(t *testing.T) {
	tests := []struct {
		typ  ActionType
		want string
	}{
		{ActionTypeExec, "exec"},
		{ActionTypeAPI, "api"},
		{ActionTypeView, "view"},
	}

	for _, tt := range tests {
		if string(tt.typ) != tt.want {
			t.Errorf("ActionType %v = %q, want %q", tt.typ, string(tt.typ), tt.want)
		}
	}
}

func TestExecuteWithDAO_UnknownType(t *testing.T) {
	action := Action{
		Type: ActionType("unknown"),
	}

	result := ExecuteWithDAO(context.Background(), action, &mockResource{}, "test", "resource")

	if result.Success {
		t.Error("ExecuteWithDAO with unknown type should fail")
	}
	if result.Error == nil {
		t.Error("ExecuteWithDAO with unknown type should return error")
	}
}

func TestGlobalRegistry(t *testing.T) {
	// Global registry should be initialized
	if Global == nil {
		t.Fatal("Global registry should not be nil")
	}

	// Test RegisterExecutor convenience function
	called := false
	RegisterExecutor("test", "resource", func(ctx context.Context, action Action, resource dao.Resource) ActionResult {
		called = true
		return ActionResult{Success: true}
	})

	executor := Global.GetExecutor("test", "resource")
	if executor == nil {
		t.Fatal("Global executor should be registered")
	}

	executor(context.Background(), Action{}, nil)
	if !called {
		t.Error("Global executor was not called")
	}
}

func TestExecuteWithDAO_ExecType(t *testing.T) {
	t.Run("valid command", func(t *testing.T) {
		action := Action{
			Type:    ActionTypeExec,
			Command: "echo hello",
		}

		result := ExecuteWithDAO(context.Background(), action, &mockResource{id: "test"}, "test", "resource")

		if !result.Success {
			t.Errorf("ExecuteWithDAO with valid command should succeed, got error: %v", result.Error)
		}
	})

	t.Run("empty command", func(t *testing.T) {
		action := Action{
			Type:    ActionTypeExec,
			Command: "",
		}

		result := ExecuteWithDAO(context.Background(), action, &mockResource{}, "test", "resource")

		if result.Success {
			t.Error("ExecuteWithDAO with empty command should fail")
		}
		if result.Error != ErrEmptyCommand {
			t.Errorf("Error = %v, want %v", result.Error, ErrEmptyCommand)
		}
	})

	t.Run("command with variable expansion", func(t *testing.T) {
		action := Action{
			Type:    ActionTypeExec,
			Command: "echo ${ID}",
		}

		result := ExecuteWithDAO(context.Background(), action, &mockResource{id: "test-id"}, "test", "resource")

		if !result.Success {
			t.Errorf("ExecuteWithDAO should succeed, got error: %v", result.Error)
		}
	})

	t.Run("failing command", func(t *testing.T) {
		action := Action{
			Type:    ActionTypeExec,
			Command: "exit 1",
		}

		result := ExecuteWithDAO(context.Background(), action, &mockResource{}, "test", "resource")

		if result.Success {
			t.Error("ExecuteWithDAO with failing command should fail")
		}
		if result.Error == nil {
			t.Error("ExecuteWithDAO with failing command should return error")
		}
	})
}

func TestExecuteWithDAO_APIType_NoExecutor(t *testing.T) {
	action := Action{
		Type:      ActionTypeAPI,
		Operation: "UnknownOperation",
	}

	result := ExecuteWithDAO(context.Background(), action, &mockResource{id: "test"}, "nonexistent", "resource")

	if result.Success {
		t.Error("ExecuteWithDAO with no executor should fail")
	}
	if result.Error == nil {
		t.Error("ExecuteWithDAO with no executor should return error")
	}
}

func TestExecuteWithDAO(t *testing.T) {
	t.Run("exec type uses executeExec", func(t *testing.T) {
		action := Action{
			Type:    ActionTypeExec,
			Command: "echo hello",
		}

		result := ExecuteWithDAO(context.Background(), action, &mockResource{}, "ec2", "instances")

		if !result.Success {
			t.Errorf("ExecuteWithDAO should succeed, got error: %v", result.Error)
		}
	})

	t.Run("api type with registered executor", func(t *testing.T) {
		// Register a custom executor
		called := false
		Global.RegisterExecutor("custom", "resource", func(ctx context.Context, action Action, resource dao.Resource) ActionResult {
			called = true
			return ActionResult{Success: true, Message: "custom executed"}
		})

		action := Action{
			Type:      ActionTypeAPI,
			Operation: "CustomOperation",
		}

		result := ExecuteWithDAO(context.Background(), action, &mockResource{}, "custom", "resource")

		if !called {
			t.Error("Custom executor should have been called")
		}
		if !result.Success {
			t.Error("ExecuteWithDAO should succeed")
		}
	})

	t.Run("unknown type", func(t *testing.T) {
		action := Action{
			Type: ActionType("invalid"),
		}

		result := ExecuteWithDAO(context.Background(), action, &mockResource{}, "ec2", "instances")

		if result.Success {
			t.Error("ExecuteWithDAO with unknown type should fail")
		}
	})
}

func TestAction_Struct(t *testing.T) {
	action := Action{
		Name:      "Test",
		Shortcut:  "t",
		Type:      ActionTypeAPI,
		Command:   "test cmd",
		Operation: "TestOp",
		Target:    "ec2/instances",
		Confirm:   true,
		Dangerous: true,
		Requires:  []string{"dep1", "dep2"},
		Vars:      map[string]string{"key": "value"},
	}

	if action.Name != "Test" {
		t.Errorf("Name = %q, want %q", action.Name, "Test")
	}
	if action.Shortcut != "t" {
		t.Errorf("Shortcut = %q, want %q", action.Shortcut, "t")
	}
	if action.Type != ActionTypeAPI {
		t.Errorf("Type = %q, want %q", action.Type, ActionTypeAPI)
	}
	if !action.Confirm {
		t.Error("Confirm should be true")
	}
	if !action.Dangerous {
		t.Error("Dangerous should be true")
	}
	if len(action.Requires) != 2 {
		t.Errorf("Requires length = %d, want 2", len(action.Requires))
	}
	if action.Vars["key"] != "value" {
		t.Errorf("Vars[key] = %q, want %q", action.Vars["key"], "value")
	}
}
