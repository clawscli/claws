package config

import (
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
	cfg.addWarning("warning 1")
	cfg.addWarning("warning 2")

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

func TestCommonRegions(t *testing.T) {
	if len(CommonRegions) == 0 {
		t.Error("CommonRegions should not be empty")
	}

	// Check some expected regions are present
	expectedRegions := []string{"us-east-1", "us-west-2", "eu-west-1", "ap-northeast-1"}
	for _, expected := range expectedRegions {
		found := false
		for _, region := range CommonRegions {
			if region == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("CommonRegions should contain %q", expected)
		}
	}
}

func TestSelectionLoadOptions(t *testing.T) {
	tests := []struct {
		name    string
		sel     ProfileSelection
		wantLen int
	}{
		{
			name:    "SDK default",
			sel:     SDKDefault(),
			wantLen: 1, // just IMDS region
		},
		{
			name:    "env only",
			sel:     EnvOnly(),
			wantLen: 3, // IMDS region + 2 empty file options
		},
		{
			name:    "named profile",
			sel:     NamedProfile("production"),
			wantLen: 2, // IMDS region + profile option
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := SelectionLoadOptions(tt.sel)
			if len(opts) != tt.wantLen {
				t.Errorf("SelectionLoadOptions(%v) returned %d options, want %d", tt.sel, len(opts), tt.wantLen)
			}
		})
	}
}

func TestConfig_DemoMode(t *testing.T) {
	cfg := &Config{accountID: "111122223333"}

	// Demo mode disabled - should return real account ID
	if cfg.AccountID() != "111122223333" {
		t.Errorf("AccountID() = %q, want %q", cfg.AccountID(), "111122223333")
	}

	// Enable demo mode
	cfg.SetDemoMode(true)
	if !cfg.DemoMode() {
		t.Error("DemoMode() = false, want true")
	}

	// Should return masked account ID
	if cfg.AccountID() != DemoAccountID {
		t.Errorf("AccountID() = %q, want %q (demo mode)", cfg.AccountID(), DemoAccountID)
	}

	// MaskAccountID should also mask
	if cfg.MaskAccountID("999988887777") != DemoAccountID {
		t.Errorf("MaskAccountID() = %q, want %q", cfg.MaskAccountID("999988887777"), DemoAccountID)
	}

	// Disable demo mode
	cfg.SetDemoMode(false)
	if cfg.AccountID() != "111122223333" {
		t.Errorf("AccountID() = %q, want %q after disabling demo mode", cfg.AccountID(), "111122223333")
	}
}
