package metrics

import (
	"testing"

	"github.com/clawscli/claws/internal/render"
)

func TestFetcher_buildQueries(t *testing.T) {
	f := &Fetcher{}
	spec := &render.MetricSpec{
		Namespace:     "AWS/EC2",
		MetricName:    "CPUUtilization",
		DimensionName: "InstanceId",
		Stat:          "Average",
	}

	tests := []struct {
		name        string
		resourceIDs []string
		wantLen     int
	}{
		{"empty", []string{}, 0},
		{"single", []string{"i-123"}, 1},
		{"multiple", []string{"i-1", "i-2", "i-3"}, 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			queries := f.buildQueries(tt.resourceIDs, spec)
			if len(queries) != tt.wantLen {
				t.Errorf("buildQueries() len = %d, want %d", len(queries), tt.wantLen)
			}
		})
	}
}

func TestFetcher_buildQueries_correctStructure(t *testing.T) {
	f := &Fetcher{}
	spec := &render.MetricSpec{
		Namespace:     "AWS/EC2",
		MetricName:    "CPUUtilization",
		DimensionName: "InstanceId",
		Stat:          "Average",
	}

	queries := f.buildQueries([]string{"i-abc123"}, spec)
	if len(queries) != 1 {
		t.Fatalf("expected 1 query, got %d", len(queries))
	}

	q := queries[0]
	if *q.Id != "m0" {
		t.Errorf("Id = %s, want m0", *q.Id)
	}
	if q.MetricStat == nil {
		t.Fatal("MetricStat is nil")
	}
	if *q.MetricStat.Metric.Namespace != "AWS/EC2" {
		t.Errorf("Namespace = %s, want AWS/EC2", *q.MetricStat.Metric.Namespace)
	}
	if *q.MetricStat.Metric.MetricName != "CPUUtilization" {
		t.Errorf("MetricName = %s, want CPUUtilization", *q.MetricStat.Metric.MetricName)
	}
	if len(q.MetricStat.Metric.Dimensions) != 1 {
		t.Fatalf("Dimensions len = %d, want 1", len(q.MetricStat.Metric.Dimensions))
	}
	if *q.MetricStat.Metric.Dimensions[0].Name != "InstanceId" {
		t.Errorf("Dimension name = %s, want InstanceId", *q.MetricStat.Metric.Dimensions[0].Name)
	}
	if *q.MetricStat.Metric.Dimensions[0].Value != "i-abc123" {
		t.Errorf("Dimension value = %s, want i-abc123", *q.MetricStat.Metric.Dimensions[0].Value)
	}
}

func TestBatchSplitting(t *testing.T) {
	tests := []struct {
		name        string
		total       int
		batchSize   int
		wantBatches int
	}{
		{"under limit", 100, 500, 1},
		{"at limit", 500, 500, 1},
		{"over limit", 501, 500, 2},
		{"double", 1000, 500, 2},
		{"triple", 1200, 500, 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			batches := 0
			for i := 0; i < tt.total; i += tt.batchSize {
				batches++
			}
			if batches != tt.wantBatches {
				t.Errorf("batches = %d, want %d", batches, tt.wantBatches)
			}
		})
	}
}

func TestProcessResults(t *testing.T) {
	f := &Fetcher{}
	resourceIDs := []string{"i-1", "i-2", "i-3"}
	data := NewMetricData(nil)

	f.processResults(nil, resourceIDs, data)
	if len(data.Results) != 0 {
		t.Errorf("expected 0 results, got %d", len(data.Results))
	}
}
