package metrics

import (
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"

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

func TestProcessResults_WithData(t *testing.T) {
	f := &Fetcher{}
	resourceIDs := []string{"i-abc", "i-def", "i-ghi"}
	data := NewMetricData(nil)

	results := []types.MetricDataResult{
		{Id: aws.String("m0"), Values: []float64{10.0, 20.0, 30.0}},
		{Id: aws.String("m1"), Values: []float64{5.0, 15.0}},
		{Id: aws.String("m2"), Values: []float64{}},
	}

	f.processResults(results, resourceIDs, data)

	if len(data.Results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(data.Results))
	}

	r0 := data.Results["i-abc"]
	if r0 == nil {
		t.Fatal("i-abc not found")
	}
	if !r0.HasData || r0.Latest != 30.0 || len(r0.Values) != 3 {
		t.Errorf("i-abc: HasData=%v, Latest=%v, len=%d", r0.HasData, r0.Latest, len(r0.Values))
	}

	r1 := data.Results["i-def"]
	if r1 == nil {
		t.Fatal("i-def not found")
	}
	if !r1.HasData || r1.Latest != 15.0 {
		t.Errorf("i-def: HasData=%v, Latest=%v", r1.HasData, r1.Latest)
	}

	r2 := data.Results["i-ghi"]
	if r2 == nil {
		t.Fatal("i-ghi not found")
	}
	if r2.HasData {
		t.Errorf("i-ghi should have no data")
	}
}

func TestProcessResults_UnknownQueryID(t *testing.T) {
	f := &Fetcher{}
	resourceIDs := []string{"i-abc"}
	data := NewMetricData(nil)

	results := []types.MetricDataResult{
		{Id: aws.String("m99"), Values: []float64{100.0}},
	}

	f.processResults(results, resourceIDs, data)

	if len(data.Results) != 0 {
		t.Errorf("expected 0 results for unknown query ID, got %d", len(data.Results))
	}
}
