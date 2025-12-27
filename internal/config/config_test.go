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
