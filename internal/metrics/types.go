package metrics

import "github.com/clawscli/claws/internal/render"

// MetricResult holds metric data for a single resource.
type MetricResult struct {
	ResourceID string
	Values     []float64
	Latest     float64
	HasData    bool
}

// MetricData holds metric results for multiple resources.
type MetricData struct {
	Results map[string]*MetricResult
	Spec    *render.MetricSpec
}

func NewMetricData(spec *render.MetricSpec) *MetricData {
	return &MetricData{
		Results: make(map[string]*MetricResult),
		Spec:    spec,
	}
}

func (m *MetricData) Get(resourceID string) *MetricResult {
	if m == nil || m.Results == nil {
		return nil
	}
	return m.Results[resourceID]
}
