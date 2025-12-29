package metrics

import "testing"

func TestMetricData_Get_Nil(t *testing.T) {
	var data *MetricData
	result := data.Get("test-id")
	if result != nil {
		t.Errorf("Get on nil MetricData = %v, want nil", result)
	}
}

func TestMetricData_Get_NilResults(t *testing.T) {
	data := &MetricData{Results: nil}
	result := data.Get("test-id")
	if result != nil {
		t.Errorf("Get on nil Results = %v, want nil", result)
	}
}

func TestMetricData_Get_NotFound(t *testing.T) {
	data := NewMetricData(nil)
	result := data.Get("nonexistent")
	if result != nil {
		t.Errorf("Get(nonexistent) = %v, want nil", result)
	}
}

func TestMetricData_Get_Found(t *testing.T) {
	data := NewMetricData(nil)
	expected := &MetricResult{ResourceID: "test-id", HasData: true}
	data.Results["test-id"] = expected

	result := data.Get("test-id")
	if result != expected {
		t.Errorf("Get(test-id) = %v, want %v", result, expected)
	}
}
