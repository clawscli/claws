package view

import (
	"strings"
	"testing"

	"charm.land/lipgloss/v2"

	"github.com/clawscli/claws/internal/config"
	"github.com/clawscli/claws/internal/render"
)

func TestHeaderPanel_New(t *testing.T) {
	hp := NewHeaderPanel()

	if hp == nil {
		t.Fatal("NewHeaderPanel() returned nil")
	}
}

func TestHeaderPanel_RenderNormalMode(t *testing.T) {
	cfg := config.Global()
	t.Cleanup(func() { cfg.SetCompactHeader(false) })
	cfg.SetCompactHeader(false)

	hp := NewHeaderPanel()
	hp.SetWidth(80)

	output := hp.Render("ec2", "instances", nil)

	lines := strings.Count(output, "\n")
	if lines < 3 {
		t.Errorf("Normal mode should have multiple lines (at least 4), got %d lines", lines+1)
	}

	if !strings.Contains(output, "Profile:") {
		t.Error("Normal mode output should contain 'Profile:' label")
	}
	if !strings.Contains(output, "Region:") {
		t.Error("Normal mode output should contain 'Region:' label")
	}
}

func TestHeaderPanel_RenderCompactMode(t *testing.T) {
	cfg := config.Global()
	t.Cleanup(func() { cfg.SetCompactHeader(false) })
	cfg.SetCompactHeader(true)

	hp := NewHeaderPanel()
	hp.SetWidth(80)

	output := hp.Render("ec2", "instances", nil)

	lines := strings.Count(output, "\n")
	if lines > 3 {
		t.Errorf("Compact mode should have minimal lines (1-2), got %d lines", lines+1)
	}

	if !strings.Contains(output, "│") {
		t.Error("Compact mode output should contain '│' separator")
	}
}

func TestHeaderPanel_RenderModeSwitching(t *testing.T) {
	cfg := config.Global()
	t.Cleanup(func() { cfg.SetCompactHeader(false) })
	hp := NewHeaderPanel()
	hp.SetWidth(80)

	cfg.SetCompactHeader(false)
	normalOutput := hp.Render("ec2", "instances", nil)
	normalLines := strings.Count(normalOutput, "\n")

	cfg.SetCompactHeader(true)
	compactOutput := hp.Render("ec2", "instances", nil)
	compactLines := strings.Count(compactOutput, "\n")

	if normalLines <= compactLines {
		t.Errorf("Normal mode should have more lines than Compact mode. Normal: %d, Compact: %d", normalLines+1, compactLines+1)
	}
}

func TestHeaderPanel_RenderHome(t *testing.T) {
	cfg := config.Global()
	t.Cleanup(func() { cfg.SetCompactHeader(false) })
	hp := NewHeaderPanel()
	hp.SetWidth(80)

	cfg.SetCompactHeader(false)
	output := hp.RenderHome()

	if !strings.Contains(output, "Profile:") {
		t.Error("RenderHome() should contain 'Profile:' label")
	}
	if !strings.Contains(output, "Region:") {
		t.Error("RenderHome() should contain 'Region:' label")
	}
}

func TestHeaderPanel_RenderHomeCompact(t *testing.T) {
	cfg := config.Global()
	t.Cleanup(func() { cfg.SetCompactHeader(false) })
	hp := NewHeaderPanel()
	hp.SetWidth(80)

	cfg.SetCompactHeader(true)
	output := hp.RenderHome()

	if !strings.Contains(output, "│") {
		t.Error("RenderHome() in compact mode should contain '│' separator")
	}
}

func TestHeaderPanel_RenderWithSummaryFields(t *testing.T) {
	cfg := config.Global()
	t.Cleanup(func() { cfg.SetCompactHeader(false) })
	cfg.SetCompactHeader(false)

	hp := NewHeaderPanel()
	hp.SetWidth(80)

	summaryFields := []render.SummaryField{
		{Label: "ID", Value: "i-1234567890abcdef0"},
		{Label: "State", Value: "running"},
		{Label: "Type", Value: "t3.medium"},
	}

	output := hp.Render("ec2", "instances", summaryFields)

	if !strings.Contains(output, "ID:") {
		t.Error("Output should contain 'ID:' label from summary fields")
	}
	if !strings.Contains(output, "State:") {
		t.Error("Output should contain 'State:' label from summary fields")
	}
}

func TestHeaderPanel_Height(t *testing.T) {
	cfg := config.Global()
	t.Cleanup(func() { cfg.SetCompactHeader(false) })

	hp := NewHeaderPanel()
	hp.SetWidth(80)
	cfg.SetCompactHeader(false)

	output := hp.Render("ec2", "instances", nil)
	height := hp.Height(output)

	if height < 1 {
		t.Errorf("Height() should return positive value, got %d", height)
	}

	expectedHeight := strings.Count(output, "\n") + 1
	if height != expectedHeight {
		t.Errorf("Height() = %d, want %d based on newline count", height, expectedHeight)
	}
}

func TestFormatRegions(t *testing.T) {
	valueStyle := lipgloss.NewStyle()

	tests := []struct {
		name       string
		regions    []string
		maxWidth   int
		wantSuffix string
		notWant    string
	}{
		{
			name:       "empty regions",
			regions:    nil,
			maxWidth:   100,
			wantSuffix: "-",
		},
		{
			name:       "single region",
			regions:    []string{"us-east-1"},
			maxWidth:   100,
			wantSuffix: "us-east-1",
			notWant:    "(+",
		},
		{
			name:       "two regions fit",
			regions:    []string{"us-east-1", "us-west-2"},
			maxWidth:   100,
			wantSuffix: "us-west-2",
			notWant:    "(+",
		},
		{
			name:       "narrow width truncates",
			regions:    []string{"us-east-1", "us-west-2", "eu-west-1"},
			maxWidth:   25,
			wantSuffix: "(+2)",
		},
		{
			name:       "non-positive width truncates",
			regions:    []string{"us-east-1", "us-west-2", "eu-west-1"},
			maxWidth:   0,
			wantSuffix: "(+2)",
			notWant:    "us-west-2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatRegions(tt.regions, valueStyle, tt.maxWidth)

			if !strings.Contains(result, tt.wantSuffix) {
				t.Errorf("formatRegions() = %q, want to contain %q", result, tt.wantSuffix)
			}

			if tt.notWant != "" && strings.Contains(result, tt.notWant) {
				t.Errorf("formatRegions() = %q, should not contain %q", result, tt.notWant)
			}
		})
	}
}

func TestFormatProfilesWithAccounts(t *testing.T) {
	valueStyle := lipgloss.NewStyle()
	dangerStyle := lipgloss.NewStyle()

	tests := []struct {
		name       string
		selections []config.ProfileSelection
		accountIDs map[string]string
		maxWidth   int
		wantSuffix string
		notWant    string
	}{
		{
			name:       "empty selections",
			selections: nil,
			accountIDs: nil,
			maxWidth:   100,
			wantSuffix: "-",
		},
		{
			name: "single profile fits",
			selections: []config.ProfileSelection{
				config.NamedProfile("dev"),
			},
			accountIDs: map[string]string{"dev": "111111111111"},
			maxWidth:   100,
			wantSuffix: "dev",
			notWant:    "(+",
		},
		{
			name: "two profiles fit without suffix",
			selections: []config.ProfileSelection{
				config.NamedProfile("dev"),
				config.NamedProfile("prod"),
			},
			accountIDs: map[string]string{
				"dev":  "111111111111",
				"prod": "222222222222",
			},
			maxWidth:   200,
			wantSuffix: "prod",
			notWant:    "(+",
		},
		{
			name: "narrow width truncates second profile",
			selections: []config.ProfileSelection{
				config.NamedProfile("dev"),
				config.NamedProfile("prod"),
			},
			accountIDs: map[string]string{
				"dev":  "111111111111",
				"prod": "222222222222",
			},
			maxWidth:   25,
			wantSuffix: "(+1)",
		},
		{
			name: "very narrow width truncates all but first",
			selections: []config.ProfileSelection{
				config.NamedProfile("dev"),
				config.NamedProfile("staging"),
				config.NamedProfile("prod"),
			},
			accountIDs: map[string]string{
				"dev":     "111111111111",
				"staging": "222222222222",
				"prod":    "333333333333",
			},
			maxWidth:   30,
			wantSuffix: "(+2)",
		},
		{
			name: "non-positive width truncates",
			selections: []config.ProfileSelection{
				config.NamedProfile("dev"),
				config.NamedProfile("staging"),
				config.NamedProfile("prod"),
			},
			accountIDs: map[string]string{
				"dev":     "111111111111",
				"staging": "222222222222",
				"prod":    "333333333333",
			},
			maxWidth:   0,
			wantSuffix: "(+2)",
			notWant:    "staging",
		},
		{
			name: "missing account shows danger style",
			selections: []config.ProfileSelection{
				config.NamedProfile("dev"),
			},
			accountIDs: map[string]string{},
			maxWidth:   100,
			wantSuffix: "(-)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatProfilesWithAccounts(tt.selections, tt.accountIDs, valueStyle, dangerStyle, tt.maxWidth)

			if !strings.Contains(result, tt.wantSuffix) {
				t.Errorf("formatProfilesWithAccounts() = %q, want to contain %q", result, tt.wantSuffix)
			}

			if tt.notWant != "" && strings.Contains(result, tt.notWant) {
				t.Errorf("formatProfilesWithAccounts() = %q, should not contain %q", result, tt.notWant)
			}
		})
	}
}
