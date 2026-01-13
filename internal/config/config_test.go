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
	cfg := &Config{
		selections: []ProfileSelection{SDKDefault()},
		accountIDs: map[string]string{ProfileIDSDKDefault: "123456789012"},
	}

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

func TestConfig_CompactHeaderGetSet(t *testing.T) {
	cfg := &Config{}

	// Initial value should be false
	if cfg.CompactHeader() {
		t.Error("CompactHeader() = true, want false")
	}

	// Set to true
	cfg.SetCompactHeader(true)
	if !cfg.CompactHeader() {
		t.Error("CompactHeader() = false, want true")
	}

	// Set back to false
	cfg.SetCompactHeader(false)
	if cfg.CompactHeader() {
		t.Error("CompactHeader() = true, want false")
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
	cfg.SetSelection(SDKDefault())

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

func TestConfig_MultiProfile(t *testing.T) {
	cfg := &Config{}

	if cfg.IsMultiProfile() {
		t.Error("IsMultiProfile() should be false initially")
	}

	cfg.SetSelection(NamedProfile("dev"))
	if cfg.IsMultiProfile() {
		t.Error("IsMultiProfile() should be false with single selection")
	}

	cfg.SetSelections([]ProfileSelection{NamedProfile("dev"), NamedProfile("prod")})
	if !cfg.IsMultiProfile() {
		t.Error("IsMultiProfile() should be true with multiple selections")
	}

	sels := cfg.Selections()
	if len(sels) != 2 {
		t.Errorf("Selections() = %d items, want 2", len(sels))
	}
}

func TestConfig_AccountIDs(t *testing.T) {
	cfg := &Config{}
	cfg.SetSelections([]ProfileSelection{NamedProfile("dev"), NamedProfile("prod")})

	ids := map[string]string{"dev": "111111111111", "prod": "222222222222"}
	cfg.SetAccountIDs(ids)

	got := cfg.AccountIDs()
	if got["dev"] != "111111111111" {
		t.Errorf("AccountIDs()[dev] = %q, want %q", got["dev"], "111111111111")
	}
	if got["prod"] != "222222222222" {
		t.Errorf("AccountIDs()[prod] = %q, want %q", got["prod"], "222222222222")
	}
}

func TestIsValidRegion(t *testing.T) {
	tests := []struct {
		region string
		want   bool
	}{
		{"us-east-1", true},
		{"us-west-2", true},
		{"eu-west-1", true},
		{"eu-central-1", true},
		{"ap-northeast-1", true},
		{"ap-southeast-2", true},
		{"sa-east-1", true},
		{"ca-central-1", true},
		{"me-south-1", true},
		{"af-south-1", true},
		{"us-gov-west-1", true},
		{"us-gov-east-1", true},
		{"us-iso-east-1", true},
		{"us-isob-east-1", true},
		{"", false},
		{"invalid", false},
		{"us-east", false},
		{"us-east-", false},
		{"US-EAST-1", false},
		{"us_east_1", false},
		{"us-east-1-extra", false},
		{"a]b[c", false},
	}

	for _, tt := range tests {
		t.Run(tt.region, func(t *testing.T) {
			if got := IsValidRegion(tt.region); got != tt.want {
				t.Errorf("IsValidRegion(%q) = %v, want %v", tt.region, got, tt.want)
			}
		})
	}
}

func TestIsValidAccountID(t *testing.T) {
	tests := []struct {
		id   string
		want bool
	}{
		{"123456789012", true},
		{"000000000000", true},
		{"999999999999", true},
		{"", true},
		{"12345678901", false},
		{"1234567890123", false},
		{"12345678901a", false},
		{"abcdefghijkl", false},
		{"123-456-789", false},
	}

	for _, tt := range tests {
		t.Run(tt.id, func(t *testing.T) {
			if got := IsValidAccountID(tt.id); got != tt.want {
				t.Errorf("IsValidAccountID(%q) = %v, want %v", tt.id, got, tt.want)
			}
		})
	}
}

func TestIsValidProfileName(t *testing.T) {
	tests := []struct {
		name string
		want bool
	}{
		{"default", true},
		{"production", true},
		{"my-profile", true},
		{"my_profile", true},
		{"my.profile", true},
		{"profile123", true},
		{"Profile-Name_123.test", true},
		{"", false},
		{"profile with space", false},
		{"profile@name", false},
		{"profile/name", false},
		{"profile:name", false},
		{"日本語", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsValidProfileName(tt.name); got != tt.want {
				t.Errorf("IsValidProfileName(%q) = %v, want %v", tt.name, got, tt.want)
			}
		})
	}
}

func TestValidationError(t *testing.T) {
	err := &ValidationError{Field: "region", Value: "invalid", Message: "invalid region format"}
	if err.Error() != "invalid region format" {
		t.Errorf("Error() = %q, want %q", err.Error(), "invalid region format")
	}
}
