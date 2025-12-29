package metrics

import (
	"strings"
	"testing"
)

func TestRenderSparkline_Nil(t *testing.T) {
	result := RenderSparkline(nil)
	if result != "·······  -" {
		t.Errorf("RenderSparkline(nil) = %q, want %q", result, "·······  -")
	}
}

func TestRenderSparkline_NoData(t *testing.T) {
	result := RenderSparkline(&MetricResult{HasData: false})
	if result != "·······  -" {
		t.Errorf("RenderSparkline(no data) = %q, want %q", result, "·······  -")
	}
}

func TestRenderSparkline_EmptyValues(t *testing.T) {
	result := RenderSparkline(&MetricResult{HasData: true, Values: []float64{}})
	if result != "·······  -" {
		t.Errorf("RenderSparkline(empty) = %q, want %q", result, "·······  -")
	}
}

func TestRenderSparkline_SingleValue(t *testing.T) {
	result := RenderSparkline(&MetricResult{
		HasData: true,
		Values:  []float64{50.0},
		Latest:  50.0,
	})
	if !strings.HasSuffix(result, " 50%") {
		t.Errorf("RenderSparkline(single) = %q, want suffix ' 50%%'", result)
	}
}

func TestRenderSparkline_MultipleValues(t *testing.T) {
	result := RenderSparkline(&MetricResult{
		HasData: true,
		Values:  []float64{0, 25, 50, 75, 100},
		Latest:  100.0,
	})
	if !strings.HasSuffix(result, "100%") {
		t.Errorf("RenderSparkline(multi) = %q, want suffix '100%%'", result)
	}
	if !strings.ContainsAny(result, "▁▂▃▄▅▆▇█") {
		t.Errorf("RenderSparkline(multi) = %q, want sparkline chars", result)
	}
}

func TestRenderSparkline_ConstantValues(t *testing.T) {
	result := RenderSparkline(&MetricResult{
		HasData: true,
		Values:  []float64{50, 50, 50, 50, 50},
		Latest:  50.0,
	})
	if !strings.HasSuffix(result, " 50%") {
		t.Errorf("RenderSparkline(constant) = %q, want suffix ' 50%%'", result)
	}
}

func TestRenderSparkline_TruncatesToWidth(t *testing.T) {
	values := make([]float64, 20)
	for i := range values {
		values[i] = float64(i * 5)
	}
	result := RenderSparkline(&MetricResult{
		HasData: true,
		Values:  values,
		Latest:  95.0,
	})
	parts := strings.Split(result, " ")
	sparkline := parts[0]
	if len([]rune(sparkline)) != SparklineWidth {
		t.Errorf("sparkline width = %d, want %d", len([]rune(sparkline)), SparklineWidth)
	}
}
