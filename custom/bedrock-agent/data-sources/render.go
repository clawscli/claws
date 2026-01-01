package datasources

import (
	"fmt"

	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/render"
)

// DataSourceRenderer renders Bedrock Data Source resources
// Ensure DataSourceRenderer implements render.Navigator
var _ render.Navigator = (*DataSourceRenderer)(nil)

type DataSourceRenderer struct {
	render.BaseRenderer
}

// NewDataSourceRenderer creates a new DataSourceRenderer
func NewDataSourceRenderer() render.Renderer {
	return &DataSourceRenderer{
		BaseRenderer: render.BaseRenderer{
			Service:  "bedrock-agent",
			Resource: "data-sources",
			Cols: []render.Column{
				{Name: "NAME", Width: 30, Getter: func(r dao.Resource) string { return r.GetName() }},
				{Name: "STATUS", Width: 12, Getter: getDSStatus},
				{Name: "DESCRIPTION", Width: 35, Getter: getDSDescription},
				{Name: "UPDATED", Width: 12, Getter: getDSAge},
			},
		},
	}
}

func getDSStatus(r dao.Resource) string {
	if ds, ok := r.(*DataSourceResource); ok {
		return ds.Status()
	}
	return ""
}

func getDSDescription(r dao.Resource) string {
	if ds, ok := r.(*DataSourceResource); ok {
		desc := ds.Description()
		if len(desc) > 35 {
			return desc[:32] + "..."
		}
		return desc
	}
	return ""
}

func getDSAge(r dao.Resource) string {
	if ds, ok := r.(*DataSourceResource); ok {
		if updated := ds.UpdatedAt(); updated != nil {
			return render.FormatAge(*updated)
		}
	}
	return "-"
}

// RenderDetail renders detailed data source information
func (r *DataSourceRenderer) RenderDetail(resource dao.Resource) string {
	ds, ok := resource.(*DataSourceResource)
	if !ok {
		return ""
	}

	d := render.NewDetailBuilder()

	d.Title("Bedrock Data Source", ds.GetName())

	// Basic Info
	d.Section("Basic Information")
	d.Field("Name", ds.GetName())
	d.Field("ID", ds.GetID())
	d.Field("Status", ds.Status())
	d.Field("Knowledge Base ID", ds.KnowledgeBaseId())

	if desc := ds.Description(); desc != "" {
		d.Field("Description", desc)
	}

	// Configuration
	d.Section("Configuration")
	if dsType := ds.DataSourceType(); dsType != "" {
		d.Field("Type", dsType)
	}
	if s3Bucket := ds.S3BucketArn(); s3Bucket != "" {
		d.Field("S3 Bucket ARN", s3Bucket)
	}
	if delPolicy := ds.DataDeletionPolicy(); delPolicy != "" {
		d.Field("Deletion Policy", delPolicy)
	}

	// Timestamps
	d.Section("Timestamps")
	if created := ds.CreatedAt(); created != nil {
		d.Field("Created", created.Format("2006-01-02 15:04:05"))
	}
	if updated := ds.UpdatedAt(); updated != nil {
		d.Field("Updated", updated.Format("2006-01-02 15:04:05"))
	}

	// Failure Reasons
	if failures := ds.FailureReasons(); len(failures) > 0 {
		d.Section("Failure Reasons")
		for _, reason := range failures {
			d.Field("", reason)
		}
	}

	return d.String()
}

// RenderSummary returns summary fields for the header panel
func (r *DataSourceRenderer) RenderSummary(resource dao.Resource) []render.SummaryField {
	ds, ok := resource.(*DataSourceResource)
	if !ok {
		return r.BaseRenderer.RenderSummary(resource)
	}

	fields := []render.SummaryField{
		{Label: "Name", Value: ds.GetName()},
		{Label: "ID", Value: ds.GetID()},
		{Label: "Status", Value: ds.Status()},
		{Label: "Knowledge Base ID", Value: ds.KnowledgeBaseId()},
	}

	if desc := ds.Description(); desc != "" {
		fields = append(fields, render.SummaryField{Label: "Description", Value: desc})
	}

	if created := ds.CreatedAt(); created != nil {
		fields = append(fields, render.SummaryField{Label: "Created", Value: fmt.Sprintf("%s ago", render.FormatAge(*created))})
	}

	return fields
}

// Navigations returns navigation shortcuts
func (r *DataSourceRenderer) Navigations(resource dao.Resource) []render.Navigation {
	return nil
}
