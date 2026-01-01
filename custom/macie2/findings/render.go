package findings

import (
	"fmt"

	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/render"
)

// FindingRenderer renders Macie findings.
type FindingRenderer struct {
	render.BaseRenderer
}

// NewFindingRenderer creates a new FindingRenderer.
func NewFindingRenderer() render.Renderer {
	return &FindingRenderer{
		BaseRenderer: render.BaseRenderer{
			Service:  "macie2",
			Resource: "findings",
			Cols: []render.Column{
				{Name: "FINDING ID", Width: 35, Getter: func(r dao.Resource) string { return r.GetID() }},
				{Name: "TITLE", Width: 30, Getter: getTitle},
				{Name: "SEVERITY", Width: 12, Getter: getSeverity},
				{Name: "BUCKET", Width: 25, Getter: getBucket},
				{Name: "UPDATED", Width: 12, Getter: getUpdated},
			},
		},
	}
}

func getTitle(r dao.Resource) string {
	finding, ok := r.(*FindingResource)
	if !ok {
		return ""
	}
	return finding.Title()
}

func getSeverity(r dao.Resource) string {
	finding, ok := r.(*FindingResource)
	if !ok {
		return ""
	}
	return finding.Severity()
}

func getBucket(r dao.Resource) string {
	finding, ok := r.(*FindingResource)
	if !ok {
		return ""
	}
	return finding.BucketName()
}

func getUpdated(r dao.Resource) string {
	finding, ok := r.(*FindingResource)
	if !ok {
		return ""
	}
	if t := finding.UpdatedAt(); t != nil {
		return render.FormatAge(*t)
	}
	return ""
}

// RenderDetail renders the detail view for a finding.
func (r *FindingRenderer) RenderDetail(resource dao.Resource) string {
	finding, ok := resource.(*FindingResource)
	if !ok {
		return ""
	}

	d := render.NewDetailBuilder()

	d.Title("Macie Finding", finding.GetID())

	// Basic Info
	d.Section("Basic Information")
	d.Field("Finding ID", finding.GetID())
	d.Field("Title", finding.Title())
	d.Field("Description", finding.Description())
	d.Field("Type", finding.Type())
	d.Field("Category", finding.Category())
	d.Field("Severity", fmt.Sprintf("%s (score: %d)", finding.Severity(), finding.SeverityScore()))

	// Account & Region
	d.Section("Account & Region")
	d.Field("Account ID", finding.AccountId())
	d.Field("Region", finding.Region())

	// Affected S3 Bucket
	if finding.BucketName() != "" {
		d.Section("Affected S3 Bucket")
		d.Field("Bucket Name", finding.BucketName())
		d.Field("Bucket ARN", finding.BucketArn())
		if finding.BucketOwner() != "" {
			d.Field("Bucket Owner", finding.BucketOwner())
		}
	}

	// Affected S3 Object
	if finding.ObjectKey() != "" {
		d.Section("Affected S3 Object")
		d.Field("Object Key", finding.ObjectKey())
		if finding.ObjectPath() != "" {
			d.Field("Object Path", finding.ObjectPath())
		}
		if finding.ObjectSize() > 0 {
			d.Field("Object Size", render.FormatSize(finding.ObjectSize()))
		}
		if finding.ObjectStorageClass() != "" {
			d.Field("Storage Class", finding.ObjectStorageClass())
		}
	}

	// Status
	d.Section("Status")
	d.Field("Count", fmt.Sprintf("%d", finding.Count()))
	if finding.Archived() {
		d.Field("Archived", "Yes")
	}
	if finding.Sample() {
		d.Field("Sample", "Yes")
	}

	// Timestamps
	d.Section("Timestamps")
	if t := finding.CreatedAt(); t != nil {
		d.Field("Created", t.Format("2006-01-02 15:04:05"))
	}
	if t := finding.UpdatedAt(); t != nil {
		d.Field("Updated", t.Format("2006-01-02 15:04:05"))
	}

	return d.String()
}

// RenderSummary renders summary fields for a finding.
func (r *FindingRenderer) RenderSummary(resource dao.Resource) []render.SummaryField {
	finding, ok := resource.(*FindingResource)
	if !ok {
		return r.BaseRenderer.RenderSummary(resource)
	}

	return []render.SummaryField{
		{Label: "Finding ID", Value: finding.GetID()},
		{Label: "Severity", Value: finding.Severity()},
		{Label: "Bucket", Value: finding.BucketName()},
		{Label: "Object", Value: finding.ObjectKey()},
	}
}
