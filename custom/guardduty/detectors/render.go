package detectors

import (
	"fmt"
	"sort"

	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/render"
)

// DetectorRenderer renders GuardDuty detectors
// Ensure DetectorRenderer implements render.Navigator
var _ render.Navigator = (*DetectorRenderer)(nil)

type DetectorRenderer struct {
	render.BaseRenderer
}

// NewDetectorRenderer creates a new DetectorRenderer
func NewDetectorRenderer() *DetectorRenderer {
	return &DetectorRenderer{
		BaseRenderer: render.BaseRenderer{
			Service:  "guardduty",
			Resource: "detectors",
			Cols: []render.Column{
				{Name: "DETECTOR ID", Width: 36, Getter: getDetectorId},
				{Name: "STATUS", Width: 10, Getter: getStatus},
				{Name: "FREQUENCY", Width: 15, Getter: getFrequency},
				{Name: "AGE", Width: 12, Getter: getAge},
			},
		},
	}
}

func getDetectorId(r dao.Resource) string {
	if d, ok := r.(*DetectorResource); ok {
		return d.DetectorId
	}
	return ""
}

func getStatus(r dao.Resource) string {
	if d, ok := r.(*DetectorResource); ok {
		return d.Status()
	}
	return ""
}

func getFrequency(r dao.Resource) string {
	if d, ok := r.(*DetectorResource); ok {
		return d.FindingPublishingFrequency()
	}
	return ""
}

func getAge(r dao.Resource) string {
	if d, ok := r.(*DetectorResource); ok {
		if t := d.CreatedAtTime(); t != nil {
			return render.FormatAge(*t)
		}
	}
	return "-"
}

// RenderDetail renders detailed detector information
func (r *DetectorRenderer) RenderDetail(resource dao.Resource) string {
	detector, ok := resource.(*DetectorResource)
	if !ok {
		return ""
	}

	d := render.NewDetailBuilder()

	d.Title("GuardDuty Detector", detector.DetectorId)

	// Basic Info
	d.Section("Basic Information")
	d.Field("Detector ID", detector.DetectorId)
	d.Field("Status", detector.Status())
	d.Field("Finding Publishing Frequency", detector.FindingPublishingFrequency())

	if role := detector.ServiceRole(); role != "" {
		d.Field("Service Role", role)
	}

	// Features
	featuresStatus := detector.FeaturesStatus()
	if len(featuresStatus) > 0 {
		d.Section("Features")
		// Sort keys for consistent ordering
		keys := make([]string, 0, len(featuresStatus))
		for k := range featuresStatus {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			d.Field(k, string(featuresStatus[k]))
		}
	}

	// Tags
	if len(detector.Tags) > 0 {
		d.Section("Tags")
		// Sort keys for consistent ordering
		keys := make([]string, 0, len(detector.Tags))
		for k := range detector.Tags {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			d.Field(k, detector.Tags[k])
		}
	}

	// Timestamps
	d.Section("Timestamps")
	if created := detector.CreatedAt(); created != "" {
		d.Field("Created", created)
	}
	if updated := detector.UpdatedAt(); updated != "" {
		d.Field("Last Updated", updated)
	}

	return d.String()
}

// RenderSummary returns summary fields for the header panel
func (r *DetectorRenderer) RenderSummary(resource dao.Resource) []render.SummaryField {
	detector, ok := resource.(*DetectorResource)
	if !ok {
		return r.BaseRenderer.RenderSummary(resource)
	}

	fields := []render.SummaryField{
		{Label: "Detector ID", Value: detector.DetectorId},
		{Label: "Status", Value: detector.Status()},
		{Label: "Frequency", Value: detector.FindingPublishingFrequency()},
	}

	// Count enabled features
	featuresStatus := detector.FeaturesStatus()
	if len(featuresStatus) > 0 {
		enabledCount := 0
		for _, status := range featuresStatus {
			if status == "ENABLED" {
				enabledCount++
			}
		}
		fields = append(fields, render.SummaryField{
			Label: "Features",
			Value: fmt.Sprintf("%d/%d enabled", enabledCount, len(featuresStatus)),
		})
	}

	if created := detector.CreatedAt(); created != "" {
		fields = append(fields, render.SummaryField{Label: "Created", Value: created})
	}

	return fields
}

// Navigations returns navigation shortcuts
func (r *DetectorRenderer) Navigations(resource dao.Resource) []render.Navigation {
	detector, ok := resource.(*DetectorResource)
	if !ok {
		return nil
	}

	return []render.Navigation{
		{
			Key: "f", Label: "Findings", Service: "guardduty", Resource: "findings",
			FilterField: "DetectorId", FilterValue: detector.DetectorId,
		},
	}
}
