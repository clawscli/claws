package action

import (
	"context"
	"errors"
	"os/exec"
	"strings"
	"testing"

	"github.com/clawscli/claws/internal/config"
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

func TestIsAllowedInReadOnly(t *testing.T) {
	tests := []struct {
		name string
		act  Action
		want bool
	}{
		{"exec allowlisted", Action{Type: ActionTypeExec, Name: ActionNameLogin}, true},
		{"exec not allowlisted", Action{Type: ActionTypeExec, Name: "SomeExec"}, false},
		{"api allowlisted", Action{Type: ActionTypeAPI, Operation: "DetectStackDrift"}, true},
		{"api not allowlisted", Action{Type: ActionTypeAPI, Operation: "DeleteStack"}, false},
		{"unknown type", Action{Type: ActionType("unknown")}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsAllowedInReadOnly(tt.act); got != tt.want {
				t.Errorf("IsAllowedInReadOnly() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestReadOnlyAllowlist(t *testing.T) {
	// Verify expected operations are in the allowlist
	expected := []string{
		"DetectStackDrift",     // CloudFormation: read-only drift detection
		"InvokeFunctionDryRun", // Lambda: validation only
	}

	for _, op := range expected {
		if !ReadOnlyAllowlist[op] {
			t.Errorf("ReadOnlyAllowlist should contain %q", op)
		}
	}

	// Verify dangerous operations are NOT in allowlist
	dangerous := []string{
		"DeleteStack",
		"StopInstances",
		"TerminateInstances",
		"InvokeFunction",
	}

	for _, op := range dangerous {
		if ReadOnlyAllowlist[op] {
			t.Errorf("ReadOnlyAllowlist should NOT contain %q", op)
		}
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

func TestSuccessResult(t *testing.T) {
	result := SuccessResult("operation completed")
	if !result.Success {
		t.Error("SuccessResult should have Success=true")
	}
	if result.Message != "operation completed" {
		t.Errorf("Message = %q, want %q", result.Message, "operation completed")
	}
	if result.Error != nil {
		t.Errorf("Error should be nil, got %v", result.Error)
	}
}

func TestSuccessResultWithFollowUp(t *testing.T) {
	followUp := "test-follow-up"
	result := SuccessResultWithFollowUp("done", followUp)
	if !result.Success {
		t.Error("Success should be true")
	}
	if result.FollowUpMsg != followUp {
		t.Errorf("FollowUpMsg = %v, want %v", result.FollowUpMsg, followUp)
	}
}

func TestFailResult(t *testing.T) {
	err := errors.New("test error")
	result := FailResult(err)
	if result.Success {
		t.Error("FailResult should have Success=false")
	}
	if result.Error != err {
		t.Errorf("Error = %v, want %v", result.Error, err)
	}
}

func TestFailResultf(t *testing.T) {
	baseErr := errors.New("connection failed")
	result := FailResultf(baseErr, "start instance %s", "i-123")
	if result.Success {
		t.Error("FailResultf should have Success=false")
	}
	if result.Error == nil {
		t.Fatal("Error should not be nil")
	}
	if !errors.Is(result.Error, baseErr) {
		t.Error("wrapped error should contain original error")
	}
	want := "start instance i-123: connection failed"
	if result.Error.Error() != want {
		t.Errorf("Error.Error() = %q, want %q", result.Error.Error(), want)
	}
}

func TestActionType(t *testing.T) {
	tests := []struct {
		typ  ActionType
		want string
	}{
		{ActionTypeExec, "exec"},
		{ActionTypeAPI, "api"},
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
		Confirm:   ConfirmSimple,
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
	if action.Confirm != ConfirmSimple {
		t.Errorf("Confirm = %v, want ConfirmSimple", action.Confirm)
	}
}

func TestSetAWSEnv(t *testing.T) {
	tests := []struct {
		name          string
		mode          string // "sdk_default", "env_only", "named_profile"
		profileName   string
		region        string
		baseEnv       []string
		wantProfile   string // expected AWS_PROFILE value, "" means not set
		wantNoProfile bool   // true if AWS_PROFILE should be removed
		wantRegion    string // expected AWS_REGION value, "" means not modified
		wantDefRegion string // expected AWS_DEFAULT_REGION value
	}{
		{
			name:          "SDKDefault preserves existing AWS_PROFILE",
			mode:          "sdk_default",
			region:        "",
			baseEnv:       []string{"AWS_PROFILE=existing", "PATH=/usr/bin"},
			wantProfile:   "existing",
			wantNoProfile: false,
		},
		{
			name:          "SDKDefault with region injects both region vars",
			mode:          "sdk_default",
			region:        "us-west-2",
			baseEnv:       []string{"PATH=/usr/bin"},
			wantProfile:   "",
			wantNoProfile: false,
			wantRegion:    "us-west-2",
			wantDefRegion: "us-west-2",
		},
		{
			name:          "NamedProfile sets AWS_PROFILE",
			mode:          "named_profile",
			profileName:   "myprofile",
			region:        "",
			baseEnv:       []string{"PATH=/usr/bin"},
			wantProfile:   "myprofile",
			wantNoProfile: false,
		},
		{
			name:          "NamedProfile replaces existing AWS_PROFILE",
			mode:          "named_profile",
			profileName:   "newprofile",
			region:        "",
			baseEnv:       []string{"AWS_PROFILE=oldprofile", "PATH=/usr/bin"},
			wantProfile:   "newprofile",
			wantNoProfile: false,
		},
		{
			name:          "NamedProfile with region sets both",
			mode:          "named_profile",
			profileName:   "myprofile",
			region:        "ap-northeast-1",
			baseEnv:       []string{"PATH=/usr/bin"},
			wantProfile:   "myprofile",
			wantNoProfile: false,
			wantRegion:    "ap-northeast-1",
			wantDefRegion: "ap-northeast-1",
		},
		{
			name:          "EnvOnly removes AWS_PROFILE and ignores config files",
			mode:          "env_only",
			region:        "",
			baseEnv:       []string{"AWS_PROFILE=toremove", "PATH=/usr/bin"},
			wantProfile:   "",
			wantNoProfile: true,
		},
		{
			name:          "EnvOnly with region removes profile and sets region",
			mode:          "env_only",
			region:        "eu-west-1",
			baseEnv:       []string{"AWS_PROFILE=toremove", "PATH=/usr/bin"},
			wantProfile:   "",
			wantNoProfile: true,
			wantRegion:    "eu-west-1",
			wantDefRegion: "eu-west-1",
		},
		{
			name:          "EnvOnly sets config file env vars to /dev/null",
			mode:          "env_only",
			region:        "",
			baseEnv:       []string{"PATH=/usr/bin"},
			wantProfile:   "",
			wantNoProfile: true,
		},
		{
			name:          "Region replaces existing region vars",
			mode:          "sdk_default",
			region:        "sa-east-1",
			baseEnv:       []string{"AWS_REGION=old", "AWS_DEFAULT_REGION=old", "PATH=/usr/bin"},
			wantRegion:    "sa-east-1",
			wantDefRegion: "sa-east-1",
		},
		{
			name:          "Empty region preserves existing region vars",
			mode:          "sdk_default",
			region:        "",
			baseEnv:       []string{"AWS_REGION=existing", "AWS_DEFAULT_REGION=existing", "PATH=/usr/bin"},
			wantRegion:    "existing",
			wantDefRegion: "existing",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup global config
			cfg := config.Global()
			switch tt.mode {
			case "sdk_default":
				cfg.UseSDKDefault()
			case "env_only":
				cfg.UseEnvOnly()
			case "named_profile":
				cfg.UseProfile(tt.profileName)
			}
			cfg.SetRegion(tt.region)

			// Create command with base env
			cmd := &exec.Cmd{Env: tt.baseEnv}

			// Call setAWSEnv
			setAWSEnv(cmd, "")

			// Parse resulting env into map
			envMap := make(map[string]string)
			for _, e := range cmd.Env {
				parts := strings.SplitN(e, "=", 2)
				if len(parts) == 2 {
					envMap[parts[0]] = parts[1]
				}
			}

			// Check AWS_PROFILE
			if tt.wantNoProfile {
				if _, exists := envMap["AWS_PROFILE"]; exists {
					t.Errorf("AWS_PROFILE should be removed, but found: %s", envMap["AWS_PROFILE"])
				}
			} else if tt.wantProfile != "" {
				if envMap["AWS_PROFILE"] != tt.wantProfile {
					t.Errorf("AWS_PROFILE = %q, want %q", envMap["AWS_PROFILE"], tt.wantProfile)
				}
			}

			// Check AWS_REGION
			if tt.wantRegion != "" {
				if envMap["AWS_REGION"] != tt.wantRegion {
					t.Errorf("AWS_REGION = %q, want %q", envMap["AWS_REGION"], tt.wantRegion)
				}
			}

			// Check AWS_DEFAULT_REGION
			if tt.wantDefRegion != "" {
				if envMap["AWS_DEFAULT_REGION"] != tt.wantDefRegion {
					t.Errorf("AWS_DEFAULT_REGION = %q, want %q", envMap["AWS_DEFAULT_REGION"], tt.wantDefRegion)
				}
			}

			// Check PATH is preserved
			if envMap["PATH"] != "/usr/bin" {
				t.Errorf("PATH should be preserved, got %q", envMap["PATH"])
			}

			// Check EnvOnly sets config file vars to /dev/null
			if tt.mode == "env_only" {
				if envMap["AWS_CONFIG_FILE"] != "/dev/null" {
					t.Errorf("AWS_CONFIG_FILE = %q, want /dev/null", envMap["AWS_CONFIG_FILE"])
				}
				if envMap["AWS_SHARED_CREDENTIALS_FILE"] != "/dev/null" {
					t.Errorf("AWS_SHARED_CREDENTIALS_FILE = %q, want /dev/null", envMap["AWS_SHARED_CREDENTIALS_FILE"])
				}
			}
		})
	}
}

func TestReadOnlyEnforcement_ExecuteWithDAO(t *testing.T) {
	t.Run("read-only blocks non-allowlisted API action", func(t *testing.T) {
		// Enable read-only mode
		config.Global().SetReadOnly(true)
		defer config.Global().SetReadOnly(false)

		action := Action{
			Name:      "Terminate",
			Type:      ActionTypeAPI,
			Operation: "TerminateInstances", // Not in ReadOnlyAllowlist
		}

		result := ExecuteWithDAO(context.Background(), action, &mockResource{id: "i-123"}, "ec2", "instances")

		if result.Success {
			t.Error("read-only should block non-allowlisted API action")
		}
		if result.Error != ErrReadOnlyDenied {
			t.Errorf("Error = %v, want %v", result.Error, ErrReadOnlyDenied)
		}
	})

	t.Run("read-only allows allowlisted API action", func(t *testing.T) {
		// Enable read-only mode
		config.Global().SetReadOnly(true)
		defer config.Global().SetReadOnly(false)

		// Register a test executor
		Global.RegisterExecutor("test", "readonly", func(ctx context.Context, action Action, resource dao.Resource) ActionResult {
			return ActionResult{Success: true, Message: "executed"}
		})

		action := Action{
			Name:      "DetectStackDrift",
			Type:      ActionTypeAPI,
			Operation: "DetectStackDrift", // In ReadOnlyAllowlist
		}

		result := ExecuteWithDAO(context.Background(), action, &mockResource{id: "test"}, "test", "readonly")

		if !result.Success {
			t.Errorf("read-only should allow allowlisted API action, got error: %v", result.Error)
		}
	})

	t.Run("read-only blocks non-allowlisted exec action", func(t *testing.T) {
		// Enable read-only mode
		config.Global().SetReadOnly(true)
		defer config.Global().SetReadOnly(false)

		action := Action{
			Name:    "SSM Session",
			Type:    ActionTypeExec,
			Command: "aws ssm start-session --target ${ID}", // Not in ReadOnlyExecAllowlist
		}

		result := ExecuteWithDAO(context.Background(), action, &mockResource{id: "i-123"}, "ec2", "instances")

		if result.Success {
			t.Error("read-only should block non-allowlisted exec action")
		}
		if result.Error != ErrReadOnlyDenied {
			t.Errorf("Error = %v, want %v", result.Error, ErrReadOnlyDenied)
		}
	})

	t.Run("read-only allows allowlisted exec action", func(t *testing.T) {
		// Enable read-only mode
		config.Global().SetReadOnly(true)
		defer config.Global().SetReadOnly(false)

		action := Action{
			Name:    ActionNameSSOLogin,
			Type:    ActionTypeExec,
			Command: "echo test", // Use harmless command for test
		}

		result := ExecuteWithDAO(context.Background(), action, &mockResource{id: "test"}, "ec2", "instances")

		// Should pass read-only gate (but may fail later for other reasons)
		if result.Error == ErrReadOnlyDenied {
			t.Error("read-only should not block allowlisted exec action")
		}
	})

	t.Run("non-read-only allows all actions", func(t *testing.T) {
		// Ensure read-only mode is disabled
		config.Global().SetReadOnly(false)

		// Register a test executor
		Global.RegisterExecutor("test", "nonreadonly", func(ctx context.Context, action Action, resource dao.Resource) ActionResult {
			return ActionResult{Success: true, Message: "executed"}
		})

		action := Action{
			Name:      "Terminate",
			Type:      ActionTypeAPI,
			Operation: "TerminateInstances", // Not in ReadOnlyAllowlist
		}

		result := ExecuteWithDAO(context.Background(), action, &mockResource{id: "test"}, "test", "nonreadonly")

		if !result.Success {
			t.Errorf("non-read-only should allow all actions, got error: %v", result.Error)
		}
	})
}

func TestReadOnlyEnforcement_SimpleExec(t *testing.T) {
	t.Run("read-only blocks non-allowlisted exec", func(t *testing.T) {
		// Enable read-only mode
		config.Global().SetReadOnly(true)
		defer config.Global().SetReadOnly(false)

		exec := &SimpleExec{
			Command:    "echo test",
			ActionName: "SSM Session", // Not in ReadOnlyExecAllowlist
		}

		err := exec.Run()

		if err != ErrReadOnlyDenied {
			t.Errorf("Error = %v, want %v", err, ErrReadOnlyDenied)
		}
	})

	t.Run("read-only allows allowlisted exec", func(t *testing.T) {
		// Enable read-only mode
		config.Global().SetReadOnly(true)
		defer config.Global().SetReadOnly(false)

		exec := &SimpleExec{
			Command:    "echo test",
			ActionName: ActionNameLogin, // In ReadOnlyExecAllowlist
		}

		err := exec.Run()

		// Should not return ErrReadOnlyDenied (may succeed or fail for other reasons)
		if err == ErrReadOnlyDenied {
			t.Error("read-only should not block allowlisted exec")
		}
	})
}

func TestExecuteWithDAO_EmptyOperation_BeforeReadOnlyCheck(t *testing.T) {
	// P3: Verify that empty Operation error is returned before read-only check
	// This ensures better diagnostics - misconfigured actions show ErrEmptyOperation, not ErrReadOnlyDenied

	t.Run("empty Operation returns ErrEmptyOperation even in read-only mode", func(t *testing.T) {
		// Enable read-only mode
		config.Global().SetReadOnly(true)
		defer config.Global().SetReadOnly(false)

		action := Action{
			Name:      "Misconfigured Action",
			Type:      ActionTypeAPI,
			Operation: "", // Empty - misconfigured
		}

		result := ExecuteWithDAO(context.Background(), action, &mockResource{id: "test"}, "test", "resource")

		if result.Success {
			t.Error("action with empty Operation should fail")
		}
		// Key assertion: should get ErrEmptyOperation, NOT ErrReadOnlyDenied
		if result.Error != ErrEmptyOperation {
			t.Errorf("Error = %v, want %v (not ErrReadOnlyDenied)", result.Error, ErrEmptyOperation)
		}
	})

	t.Run("empty Operation returns ErrEmptyOperation in non-read-only mode", func(t *testing.T) {
		// Disable read-only mode
		config.Global().SetReadOnly(false)

		action := Action{
			Name:      "Misconfigured Action",
			Type:      ActionTypeAPI,
			Operation: "", // Empty - misconfigured
		}

		result := ExecuteWithDAO(context.Background(), action, &mockResource{id: "test"}, "test", "resource")

		if result.Success {
			t.Error("action with empty Operation should fail")
		}
		if result.Error != ErrEmptyOperation {
			t.Errorf("Error = %v, want %v", result.Error, ErrEmptyOperation)
		}
	})
}

func TestUnknownOperationError(t *testing.T) {
	err := UnknownOperationError("TestOp")
	if err == nil {
		t.Fatal("UnknownOperationError should return non-nil error")
	}
	if !strings.Contains(err.Error(), "TestOp") {
		t.Errorf("error should contain operation name, got: %s", err.Error())
	}
	if !strings.Contains(err.Error(), "unknown operation") {
		t.Errorf("error should contain 'unknown operation', got: %s", err.Error())
	}
}

func TestInvalidResourceResult(t *testing.T) {
	result := InvalidResourceResult()
	if result.Success {
		t.Error("InvalidResourceResult should return Success=false")
	}
	if result.Error != ErrInvalidResourceType {
		t.Errorf("Error = %v, want %v", result.Error, ErrInvalidResourceType)
	}
}

func TestUnknownOperationResult(t *testing.T) {
	result := UnknownOperationResult("MyOp")
	if result.Success {
		t.Error("UnknownOperationResult should return Success=false")
	}
	if result.Error == nil {
		t.Fatal("UnknownOperationResult should return non-nil error")
	}
	if !strings.Contains(result.Error.Error(), "MyOp") {
		t.Errorf("error should contain operation name, got: %s", result.Error.Error())
	}
}

func TestConfirmTokenName(t *testing.T) {
	tests := []struct {
		name     string
		resource *mockResource
		want     string
	}{
		{"with name", &mockResource{id: "i-123", name: "my-instance"}, "my-instance"},
		{"empty name", &mockResource{id: "id-only", name: ""}, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ConfirmTokenName(tt.resource); got != tt.want {
				t.Errorf("ConfirmTokenName() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestConfirmSuffix(t *testing.T) {
	tests := []struct {
		token    string
		expected string
	}{
		{"abc", "abc"},
		{"abcdef", "abcdef"},
		{"abcdefg", "bcdefg"},
		{"i-1234567890abcdef0", "bcdef0"},
		{"arn:aws:iam::123456789012:policy/MyPolicy", "Policy"},
		{"", "CONFIRM"},
	}

	for _, tt := range tests {
		t.Run(tt.token, func(t *testing.T) {
			result := ConfirmSuffix(tt.token)
			if result != tt.expected {
				t.Errorf("ConfirmSuffix(%q) = %q, want %q", tt.token, result, tt.expected)
			}
		})
	}
}

func TestConfirmMatches(t *testing.T) {
	tests := []struct {
		name     string
		token    string
		input    string
		expected bool
	}{
		{"exact match short", "abc", "abc", true},
		{"exact match 6 chars", "abcdef", "abcdef", true},
		{"suffix match long token", "i-1234567890abcdef0", "bcdef0", true},
		{"suffix match ARN", "arn:aws:iam::123456789012:policy/MyPolicy", "Policy", true},
		{"wrong suffix", "i-1234567890abcdef0", "wrong", false},
		{"partial suffix", "i-1234567890abcdef0", "def0", false},
		{"empty input", "abcdef", "", false},
		{"empty token requires CONFIRM", "", "CONFIRM", true},
		{"empty token rejects empty input", "", "", false},
		{"full token when suffix expected", "i-1234567890abcdef0", "i-1234567890abcdef0", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ConfirmMatches(tt.token, tt.input)
			if result != tt.expected {
				t.Errorf("ConfirmMatches(%q, %q) = %v, want %v", tt.token, tt.input, result, tt.expected)
			}
		})
	}
}

func TestReadOnlyEnforcement_ExecWithHeader(t *testing.T) {
	t.Run("read-only blocks non-allowlisted exec", func(t *testing.T) {
		// Enable read-only mode
		config.Global().SetReadOnly(true)
		defer config.Global().SetReadOnly(false)

		exec := &ExecWithHeader{
			Command:    "echo test",
			ActionName: "ECS Exec", // Not in ReadOnlyExecAllowlist
			Resource:   &mockResource{id: "test", name: "test"},
			Service:    "ecs",
			ResType:    "tasks",
		}

		err := exec.Run()

		if err != ErrReadOnlyDenied {
			t.Errorf("Error = %v, want %v", err, ErrReadOnlyDenied)
		}
	})

	t.Run("read-only allows allowlisted exec", func(t *testing.T) {
		// Enable read-only mode
		config.Global().SetReadOnly(true)
		defer config.Global().SetReadOnly(false)

		exec := &ExecWithHeader{
			Command:    "echo test",
			ActionName: ActionNameSSOLogin, // In ReadOnlyExecAllowlist
			Resource:   &mockResource{id: "test", name: "test"},
			Service:    "ec2",
			ResType:    "instances",
		}

		err := exec.Run()

		// Should not return ErrReadOnlyDenied (may succeed or fail for other reasons)
		if err == ErrReadOnlyDenied {
			t.Error("read-only should not block allowlisted exec")
		}
	})
}

func TestSimpleExec_SetIO(t *testing.T) {
	e := &SimpleExec{Command: "echo test", ActionName: "test"}

	var stdout, stderr strings.Builder
	stdinReader := strings.NewReader("input")

	e.SetStdin(stdinReader)
	e.SetStdout(&stdout)
	e.SetStderr(&stderr)

	if e.stdin != stdinReader {
		t.Error("SetStdin did not set stdin")
	}
	if e.stdout != &stdout {
		t.Error("SetStdout did not set stdout")
	}
	if e.stderr != &stderr {
		t.Error("SetStderr did not set stderr")
	}
}

func TestExecWithHeader_SetIO(t *testing.T) {
	e := &ExecWithHeader{
		Command:    "echo test",
		ActionName: "test",
		Resource:   &mockResource{id: "test", name: "test"},
		Service:    "test",
		ResType:    "test",
	}

	var stdout, stderr strings.Builder
	stdinReader := strings.NewReader("input")

	e.SetStdin(stdinReader)
	e.SetStdout(&stdout)
	e.SetStderr(&stderr)

	if e.stdin != stdinReader {
		t.Error("SetStdin did not set stdin")
	}
	if e.stdout != &stdout {
		t.Error("SetStdout did not set stdout")
	}
	if e.stderr != &stderr {
		t.Error("SetStderr did not set stderr")
	}
}
