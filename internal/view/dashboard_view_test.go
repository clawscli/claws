package view

import (
	"context"
	"testing"

	"github.com/clawscli/claws/internal/registry"
	"github.com/clawscli/claws/internal/ui"
)

func TestRenderBar(t *testing.T) {
	theme := ui.Current()

	tests := []struct {
		name     string
		value    float64
		max      float64
		width    int
		wantLen  int
		wantFull bool
	}{
		{"zero width", 50, 100, 0, 0, false},
		{"zero max", 50, 0, 10, 0, false},
		{"negative max", 50, -10, 10, 0, false},
		{"full bar", 100, 100, 10, 10, true},
		{"half bar", 50, 100, 10, 10, false},
		{"empty bar", 0, 100, 10, 10, false},
		{"overflow value", 150, 100, 10, 10, true},
		{"negative value", -10, 100, 10, 10, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := renderBar(tt.value, tt.max, tt.width, theme)

			if tt.width <= 0 || tt.max <= 0 {
				if result != "" {
					t.Errorf("expected empty string for invalid input, got %q", result)
				}
				return
			}

			// Result contains ANSI codes, so we check it's not empty for valid inputs
			if result == "" && tt.width > 0 && tt.max > 0 {
				t.Error("expected non-empty result for valid input")
			}
		})
	}
}

func TestDashboardView_HitTest(t *testing.T) {
	ctx := context.Background()
	reg := registry.New()

	dv := NewDashboardView(ctx, reg)
	dv.SetSize(100, 40)

	// Build hit areas with known dimensions
	panelWidth := 49
	panelHeight := 15
	headerHeight := 5
	dv.buildHitAreas(panelWidth, panelHeight, headerHeight)

	topRowY1 := headerHeight + 1
	bottomRowY1 := topRowY1 + panelHeight
	bottomRowY2 := bottomRowY1 + panelHeight - 1
	leftX2 := panelWidth
	rightX1 := panelWidth + panelGap

	// Panel indices: 0=cost, 1=operations, 2=security, 3=optimization
	tests := []struct {
		name string
		x, y int
		want int
	}{
		{"top-left panel (cost)", 10, topRowY1 + 2, 0},
		{"top-right panel (operations)", rightX1 + 5, topRowY1 + 2, 1},
		{"bottom-left panel (security)", 10, bottomRowY1 + 2, 2},
		{"bottom-right panel (optimization)", rightX1 + 5, bottomRowY1 + 2, 3},
		{"header area (no hit)", 50, headerHeight - 1, -1},
		{"below all panels (no hit)", 50, bottomRowY2 + 5, -1},
		{"left edge of cost panel", 0, topRowY1, 0},
		{"right edge of cost panel", leftX2, topRowY1, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := dv.hitTestIdx(tt.x, tt.y)
			if got != tt.want {
				t.Errorf("hitTestIdx(%d, %d) = %d, want %d", tt.x, tt.y, got, tt.want)
			}
		})
	}
}

func TestDashboardView_HitTestIdx(t *testing.T) {
	ctx := context.Background()
	reg := registry.New()

	dv := NewDashboardView(ctx, reg)
	dv.SetSize(100, 40)

	panelWidth := 49
	panelHeight := 15
	headerHeight := 5
	dv.buildHitAreas(panelWidth, panelHeight, headerHeight)

	topRowY := headerHeight + 1 + 2
	bottomRowY := headerHeight + 1 + panelHeight + 2
	rightX := panelWidth + panelGap + 5

	tests := []struct {
		name string
		x, y int
		want int
	}{
		{"cost panel (idx 0)", 10, topRowY, 0},
		{"operations panel (idx 1)", rightX, topRowY, 1},
		{"security panel (idx 2)", 10, bottomRowY, 2},
		{"optimization panel (idx 3)", rightX, bottomRowY, 3},
		{"no hit", 50, 2, -1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := dv.hitTestIdx(tt.x, tt.y)
			if got != tt.want {
				t.Errorf("hitTestIdx(%d, %d) = %d, want %d", tt.x, tt.y, got, tt.want)
			}
		})
	}
}

func TestDashboardView_CalcPanelWidth(t *testing.T) {
	ctx := context.Background()
	reg := registry.New()

	tests := []struct {
		name      string
		width     int
		wantMin   int
		wantEqual int
	}{
		{"normal width", 100, minPanelWidth, 49},
		{"small width", 50, minPanelWidth, minPanelWidth},
		{"very small width", 20, minPanelWidth, minPanelWidth},
		{"wide terminal", 200, minPanelWidth, 99},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dv := NewDashboardView(ctx, reg)
			dv.SetSize(tt.width, 40)

			got := dv.calcPanelWidth()
			if got < tt.wantMin {
				t.Errorf("calcPanelWidth() = %d, want >= %d", got, tt.wantMin)
			}
			if tt.wantEqual > 0 && got != tt.wantEqual {
				t.Errorf("calcPanelWidth() = %d, want %d", got, tt.wantEqual)
			}
		})
	}
}

func TestDashboardView_CalcPanelHeight(t *testing.T) {
	ctx := context.Background()
	reg := registry.New()

	tests := []struct {
		name         string
		height       int
		headerHeight int
		wantMin      int
	}{
		{"normal height", 40, 5, minPanelHeight},
		{"small height", 20, 5, minPanelHeight},
		{"very small height", 10, 5, minPanelHeight},
		{"tall terminal", 80, 5, minPanelHeight},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dv := NewDashboardView(ctx, reg)
			dv.SetSize(100, tt.height)

			got := dv.calcPanelHeight(tt.headerHeight)
			if got < tt.wantMin {
				t.Errorf("calcPanelHeight(%d) = %d, want >= %d", tt.headerHeight, got, tt.wantMin)
			}
		})
	}
}

func TestDashboardView_NavigateTo(t *testing.T) {
	ctx := context.Background()
	reg := registry.New()

	dv := NewDashboardView(ctx, reg)

	tests := []struct {
		name    string
		target  string
		wantCmd bool
	}{
		{"valid target", "ce/costs", true},
		{"valid security target", "securityhub/findings", true},
		{"invalid target no slash", "invalid", false},
		{"empty target", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model, cmd := dv.navigateTo(tt.target)

			if model != dv {
				t.Error("expected same model to be returned")
			}

			if tt.wantCmd && cmd == nil {
				t.Error("expected non-nil cmd for valid target")
			}
			if !tt.wantCmd && cmd != nil {
				t.Error("expected nil cmd for invalid target")
			}
		})
	}
}

func TestDashboardView_IsLoading(t *testing.T) {
	ctx := context.Background()
	reg := registry.New()

	dv := NewDashboardView(ctx, reg)

	// Initially all loading
	if !dv.isLoading() {
		t.Error("expected isLoading() to be true initially")
	}

	// Set all loading to false
	dv.alarmLoading = false
	dv.costLoading = false
	dv.anomalyLoading = false
	dv.healthLoading = false
	dv.secLoading = false
	dv.taLoading = false

	if dv.isLoading() {
		t.Error("expected isLoading() to be false when all loading complete")
	}

	// Set one back to loading
	dv.costLoading = true
	if !dv.isLoading() {
		t.Error("expected isLoading() to be true when any loading")
	}
}

func TestDashboardView_CanRefresh(t *testing.T) {
	ctx := context.Background()
	reg := registry.New()

	dv := NewDashboardView(ctx, reg)

	if !dv.CanRefresh() {
		t.Error("expected CanRefresh() to be true")
	}
}

func TestDashboardView_BuildHitAreas(t *testing.T) {
	ctx := context.Background()
	reg := registry.New()

	dv := NewDashboardView(ctx, reg)

	// First call - hitAreas is nil
	dv.buildHitAreas(50, 15, 5)

	if len(dv.hitAreas) != 4 {
		t.Errorf("expected 4 hit areas, got %d", len(dv.hitAreas))
	}

	// Verify targets
	targets := make(map[string]bool)
	for _, h := range dv.hitAreas {
		targets[h.target] = true
	}

	expectedTargets := []string{targetCost, targetOperations, targetSecurity, targetOptimization}
	for _, target := range expectedTargets {
		if !targets[target] {
			t.Errorf("missing hit area for target %q", target)
		}
	}

	// Second call - should reset and rebuild
	dv.buildHitAreas(60, 20, 6)

	if len(dv.hitAreas) != 4 {
		t.Errorf("expected 4 hit areas after rebuild, got %d", len(dv.hitAreas))
	}
}
