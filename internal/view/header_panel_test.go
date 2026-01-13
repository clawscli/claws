package view

import (
	"strings"
	"testing"

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
	hp := NewHeaderPanel()
	hp.SetWidth(80)

	cfg := config.Global()
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
