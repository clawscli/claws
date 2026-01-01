package loggroups

import (
	"fmt"
	"time"

	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/render"
)

// LogGroupRenderer renders CloudWatch Log Groups
// Ensure LogGroupRenderer implements render.Navigator
var _ render.Navigator = (*LogGroupRenderer)(nil)

type LogGroupRenderer struct {
	render.BaseRenderer
}

// NewLogGroupRenderer creates a new LogGroupRenderer
func NewLogGroupRenderer() render.Renderer {
	return &LogGroupRenderer{
		BaseRenderer: render.BaseRenderer{
			Service:  "cloudwatch",
			Resource: "log-groups",
			Cols: []render.Column{
				{Name: "LOG GROUP", Width: 50, Getter: func(r dao.Resource) string { return r.GetID() }},
				{Name: "SIZE", Width: 12, Getter: getSize},
				{Name: "RETENTION", Width: 12, Getter: getRetention},
				{Name: "CLASS", Width: 12, Getter: getClass},
				{Name: "AGE", Width: 10, Getter: getAge},
			},
		},
	}
}

func getSize(r dao.Resource) string {
	if lg, ok := r.(*LogGroupResource); ok {
		return render.FormatSize(lg.StoredBytes())
	}
	return "-"
}

func getRetention(r dao.Resource) string {
	if lg, ok := r.(*LogGroupResource); ok {
		days := lg.RetentionDays()
		if days == 0 {
			return "Never"
		}
		if days == 1 {
			return "1 day"
		}
		return fmt.Sprintf("%d days", days)
	}
	return "-"
}

func getClass(r dao.Resource) string {
	if lg, ok := r.(*LogGroupResource); ok {
		class := lg.LogGroupClass()
		if class == "" || class == "STANDARD" {
			return "Standard"
		}
		return class
	}
	return "-"
}

func getAge(r dao.Resource) string {
	if lg, ok := r.(*LogGroupResource); ok {
		creationTime := lg.CreationTime()
		if creationTime > 0 {
			t := time.UnixMilli(creationTime)
			return render.FormatAge(t)
		}
	}
	return "-"
}

// RenderDetail renders detailed log group information
func (r *LogGroupRenderer) RenderDetail(resource dao.Resource) string {
	lg, ok := resource.(*LogGroupResource)
	if !ok {
		return ""
	}

	d := render.NewDetailBuilder()

	d.Title("CloudWatch Log Group", lg.LogGroupName())

	// Basic Info
	d.Section("Basic Information")
	d.Field("Log Group Name", lg.LogGroupName())
	d.Field("ARN", lg.GetARN())

	// Storage
	d.Section("Storage")
	d.Field("Stored Bytes", render.FormatSize(lg.StoredBytes()))

	retention := lg.RetentionDays()
	if retention == 0 {
		d.Field("Retention", "Never expire")
	} else {
		d.Field("Retention", fmt.Sprintf("%d days", retention))
	}

	if class := lg.LogGroupClass(); class != "" {
		d.Field("Log Group Class", class)
	}

	// Encryption
	if kmsKey := lg.KmsKeyId(); kmsKey != "" {
		d.Section("Encryption")
		d.Field("KMS Key ID", kmsKey)
	}

	// Data Protection
	if status := lg.DataProtectionStatus(); status != "" {
		d.Section("Data Protection")
		d.Field("Status", status)
	}

	// Metrics
	if filterCount := lg.MetricFilterCount(); filterCount > 0 {
		d.Section("Metric Filters")
		d.Field("Filter Count", fmt.Sprintf("%d", filterCount))
	}

	// Creation Time
	if creationTime := lg.CreationTime(); creationTime > 0 {
		d.Section("Timestamps")
		t := time.UnixMilli(creationTime)
		d.Field("Created", t.Format("2006-01-02 15:04:05"))
		d.Field("Age", time.Since(t).Truncate(time.Second).String())
	}

	return d.String()
}

// RenderSummary returns summary fields for the header panel
func (r *LogGroupRenderer) RenderSummary(resource dao.Resource) []render.SummaryField {
	lg, ok := resource.(*LogGroupResource)
	if !ok {
		return r.BaseRenderer.RenderSummary(resource)
	}

	fields := []render.SummaryField{
		{Label: "Log Group", Value: lg.LogGroupName()},
		{Label: "ARN", Value: lg.GetARN()},
		{Label: "Size", Value: render.FormatSize(lg.StoredBytes())},
	}

	retention := lg.RetentionDays()
	if retention == 0 {
		fields = append(fields, render.SummaryField{Label: "Retention", Value: "Never expire"})
	} else {
		fields = append(fields, render.SummaryField{Label: "Retention", Value: fmt.Sprintf("%d days", retention)})
	}

	if class := lg.LogGroupClass(); class != "" {
		fields = append(fields, render.SummaryField{Label: "Class", Value: class})
	}

	if kmsKey := lg.KmsKeyId(); kmsKey != "" {
		fields = append(fields, render.SummaryField{Label: "Encrypted", Value: "Yes"})
	}

	if creationTime := lg.CreationTime(); creationTime > 0 {
		t := time.UnixMilli(creationTime)
		fields = append(fields, render.SummaryField{Label: "Created", Value: t.Format("2006-01-02 15:04:05")})
	}

	return fields
}

// Navigations returns navigation shortcuts
func (r *LogGroupRenderer) Navigations(resource dao.Resource) []render.Navigation {
	lg, ok := resource.(*LogGroupResource)
	if !ok {
		return nil
	}

	return []render.Navigation{
		{
			Key:         "s",
			Label:       "Streams",
			Service:     "cloudwatch",
			Resource:    "log-streams",
			FilterField: "LogGroupName",
			FilterValue: lg.LogGroupName(),
		},
	}
}
