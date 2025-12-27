package config

import (
	"os"
	"testing"
)

func TestConfig_RegionGetSet(t *testing.T) {
	cfg := &Config{}

	// Initial value should be empty
	if cfg.Region() != "" {
		t.Errorf("Region() = %q, want empty string", cfg.Region())
	}

	// Set and get
	cfg.SetRegion("us-east-1")
	if cfg.Region() != "us-east-1" {
		t.Errorf("Region() = %q, want %q", cfg.Region(), "us-east-1")
	}

	// Update
	cfg.SetRegion("eu-west-1")
	if cfg.Region() != "eu-west-1" {
		t.Errorf("Region() = %q, want %q", cfg.Region(), "eu-west-1")
	}
}

func TestConfig_SelectionGetSet(t *testing.T) {
	cfg := &Config{}

	// Initial value should be SDK default (zero value)
	sel := cfg.Selection()
	if !sel.IsSDKDefault() {
		t.Errorf("Selection() = %v, want SDKDefault", sel)
	}

	// Set named profile
	cfg.UseProfile("production")
	sel = cfg.Selection()
	if !sel.IsNamedProfile() || sel.ProfileName != "production" {
		t.Errorf("Selection() = %v, want NamedProfile(production)", sel)
	}

	// Set env-only mode
	cfg.UseEnvOnly()
	sel = cfg.Selection()
	if !sel.IsEnvOnly() {
		t.Errorf("Selection() = %v, want EnvOnly", sel)
	}

	// Set SDK default
	cfg.UseSDKDefault()
	sel = cfg.Selection()
	if !sel.IsSDKDefault() {
		t.Errorf("Selection() = %v, want SDKDefault", sel)
	}
}

func TestConfig_AccountID(t *testing.T) {
	cfg := &Config{accountID: "123456789012"}

	if cfg.AccountID() != "123456789012" {
		t.Errorf("AccountID() = %q, want %q", cfg.AccountID(), "123456789012")
	}
}

func TestConfig_ReadOnlyGetSet(t *testing.T) {
	cfg := &Config{}

	// Initial value should be false
	if cfg.ReadOnly() {
		t.Error("ReadOnly() = true, want false")
	}

	// Set to true
	cfg.SetReadOnly(true)
	if !cfg.ReadOnly() {
		t.Error("ReadOnly() = false, want true")
	}

	// Set back to false
	cfg.SetReadOnly(false)
	if cfg.ReadOnly() {
		t.Error("ReadOnly() = true, want false")
	}
}

func TestConfig_Warnings(t *testing.T) {
	cfg := &Config{}

	// Initial should be empty
	if len(cfg.Warnings()) != 0 {
		t.Errorf("Warnings() = %v, want empty slice", cfg.Warnings())
	}

	// Add warnings
	cfg.AddWarning("warning 1")
	cfg.AddWarning("warning 2")

	warnings := cfg.Warnings()
	if len(warnings) != 2 {
		t.Errorf("Warnings() has %d items, want 2", len(warnings))
	}
	if warnings[0] != "warning 1" {
		t.Errorf("Warnings()[0] = %q, want %q", warnings[0], "warning 1")
	}
	if warnings[1] != "warning 2" {
		t.Errorf("Warnings()[1] = %q, want %q", warnings[1], "warning 2")
	}
}

func TestGlobal(t *testing.T) {
	// Should return non-nil config
	cfg := Global()
	if cfg == nil {
		t.Fatal("Global() returned nil")
	}

	// Should return same instance on subsequent calls
	cfg2 := Global()
	if cfg != cfg2 {
		t.Error("Global() should return same instance")
	}
}

func TestProfileSelectionFromID(t *testing.T) {
	tests := []struct {
		id       string
		wantMode CredentialMode
		wantName string
	}{
		{ProfileIDSDKDefault, ModeSDKDefault, ""},
		{ProfileIDEnvOnly, ModeEnvOnly, ""},
		{"my-profile", ModeNamedProfile, "my-profile"},
		{"production", ModeNamedProfile, "production"},
	}

	for _, tt := range tests {
		t.Run(tt.id, func(t *testing.T) {
			sel := ProfileSelectionFromID(tt.id)
			if sel.Mode != tt.wantMode {
				t.Errorf("Mode = %v, want %v", sel.Mode, tt.wantMode)
			}
			if sel.ProfileName != tt.wantName {
				t.Errorf("ProfileName = %q, want %q", sel.ProfileName, tt.wantName)
			}
		})
	}
}

func TestCredentialMode_String(t *testing.T) {
	tests := []struct {
		mode CredentialMode
		want string
	}{
		{ModeSDKDefault, "SDK Default"},
		{ModeNamedProfile, ""},
		{ModeEnvOnly, "Env/IMDS Only"},
		{CredentialMode(99), "Unknown"},
	}

	for _, tt := range tests {
		got := tt.mode.String()
		if got != tt.want {
			t.Errorf("CredentialMode(%d).String() = %q, want %q", tt.mode, got, tt.want)
		}
	}
}

func TestProfileSelection_DisplayName(t *testing.T) {
	// Test without AWS_PROFILE env var
	sel := SDKDefault()
	if got := sel.DisplayName(); got != "SDK Default" {
		t.Errorf("SDKDefault().DisplayName() = %q, want %q", got, "SDK Default")
	}

	sel = EnvOnly()
	if got := sel.DisplayName(); got != "Env/IMDS Only" {
		t.Errorf("EnvOnly().DisplayName() = %q, want %q", got, "Env/IMDS Only")
	}

	sel = NamedProfile("production")
	if got := sel.DisplayName(); got != "production" {
		t.Errorf("NamedProfile(production).DisplayName() = %q, want %q", got, "production")
	}

	// Unknown mode
	sel = ProfileSelection{Mode: CredentialMode(99)}
	if got := sel.DisplayName(); got != "Unknown" {
		t.Errorf("Unknown mode DisplayName() = %q, want %q", got, "Unknown")
	}
}

func TestProfileSelection_DisplayName_WithAWSProfile(t *testing.T) {
	// Save and restore AWS_PROFILE
	orig := os.Getenv("AWS_PROFILE")
	defer os.Setenv("AWS_PROFILE", orig)

	os.Setenv("AWS_PROFILE", "test-profile")
	sel := SDKDefault()
	want := "SDK Default (AWS_PROFILE=test-profile)"
	if got := sel.DisplayName(); got != want {
		t.Errorf("SDKDefault().DisplayName() with AWS_PROFILE = %q, want %q", got, want)
	}
}

func TestProfileSelection_ID(t *testing.T) {
	tests := []struct {
		sel  ProfileSelection
		want string
	}{
		{SDKDefault(), ProfileIDSDKDefault},
		{EnvOnly(), ProfileIDEnvOnly},
		{NamedProfile("production"), "production"},
		{ProfileSelection{Mode: CredentialMode(99)}, ""},
	}

	for _, tt := range tests {
		got := tt.sel.ID()
		if got != tt.want {
			t.Errorf("ProfileSelection.ID() = %q, want %q", got, tt.want)
		}
	}
}

func TestConfig_SetAccountID(t *testing.T) {
	cfg := &Config{}

	// Initial should be empty
	if cfg.AccountID() != "" {
		t.Errorf("AccountID() = %q, want empty", cfg.AccountID())
	}

	// Set and verify
	cfg.SetAccountID("123456789012")
	if cfg.AccountID() != "123456789012" {
		t.Errorf("AccountID() = %q, want %q", cfg.AccountID(), "123456789012")
	}

	// Update
	cfg.SetAccountID("987654321098")
	if cfg.AccountID() != "987654321098" {
		t.Errorf("AccountID() = %q, want %q", cfg.AccountID(), "987654321098")
	}
}
