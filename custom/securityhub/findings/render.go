package findings

import (
	"fmt"

	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/render"
)

// FindingRenderer renders Security Hub findings.
type FindingRenderer struct {
	render.BaseRenderer
}

// NewFindingRenderer creates a new FindingRenderer.
func NewFindingRenderer() render.Renderer {
	return &FindingRenderer{
		BaseRenderer: render.BaseRenderer{
			Service:  "securityhub",
			Resource: "findings",
			Cols: []render.Column{
				{Name: "TITLE", Width: 50, Getter: getTitle},
				{Name: "SEVERITY", Width: 12, Getter: getSeverity},
				{Name: "STATUS", Width: 15, Getter: getStatus},
				{Name: "PRODUCT", Width: 20, Getter: getProduct},
				{Name: "RESOURCE TYPE", Width: 20, Getter: getResourceType},
			},
		},
	}
}

func getTitle(r dao.Resource) string {
	finding, ok := r.(*FindingResource)
	if !ok {
		return ""
	}
	title := finding.Title()
	if len(title) > 47 {
		return title[:47] + "..."
	}
	return title
}

func getSeverity(r dao.Resource) string {
	finding, ok := r.(*FindingResource)
	if !ok {
		return ""
	}
	return finding.Severity()
}

func getStatus(r dao.Resource) string {
	finding, ok := r.(*FindingResource)
	if !ok {
		return ""
	}
	return finding.Status()
}

func getProduct(r dao.Resource) string {
	finding, ok := r.(*FindingResource)
	if !ok {
		return ""
	}
	return finding.ProductName()
}

func getResourceType(r dao.Resource) string {
	finding, ok := r.(*FindingResource)
	if !ok {
		return ""
	}
	return finding.ResourceType()
}

// RenderDetail renders the detail view for a finding.
func (r *FindingRenderer) RenderDetail(resource dao.Resource) string {
	finding, ok := resource.(*FindingResource)
	if !ok {
		return ""
	}

	d := render.NewDetailBuilder()

	d.Title("Security Hub Finding", finding.Title())

	// Basic Info
	d.Section("Basic Information")
	d.Field("Title", finding.Title())
	d.Field("Finding ID", finding.GetID())
	if account := finding.AwsAccountId(); account != "" {
		d.Field("AWS Account", account)
	}
	if region := finding.Region(); region != "" {
		d.Field("Region", region)
	}

	// Severity
	d.Section("Severity")
	d.Field("Label", finding.Severity())
	if conf := finding.Confidence(); conf > 0 {
		d.Field("Confidence", fmt.Sprintf("%d%%", conf))
	}
	if crit := finding.Criticality(); crit > 0 {
		d.Field("Criticality", fmt.Sprintf("%d", crit))
	}

	// Status
	d.Section("Status")
	d.Field("Workflow Status", finding.Status())
	d.Field("Record State", finding.RecordState())
	if compliance := finding.ComplianceStatus(); compliance != "" {
		d.Field("Compliance Status", compliance)
	}
	if verif := finding.VerificationState(); verif != "" {
		d.Field("Verification State", verif)
	}

	// Finding Types
	if types := finding.Types(); len(types) > 0 {
		d.Section("Finding Types")
		for i, t := range types {
			if i >= 5 {
				d.Field("", fmt.Sprintf("... and %d more", len(types)-5))
				break
			}
			d.Field(fmt.Sprintf("Type %d", i+1), t)
		}
	}

	// Source
	d.Section("Source")
	d.Field("Product", finding.ProductName())
	if company := finding.CompanyName(); company != "" {
		d.Field("Company", company)
	}
	if generator := finding.GeneratorId(); generator != "" {
		d.Field("Generator", generator)
	}
	if productArn := finding.ProductArn(); productArn != "" {
		d.Field("Product ARN", productArn)
	}
	if sourceUrl := finding.SourceUrl(); sourceUrl != "" {
		d.Field("Source URL", sourceUrl)
	}

	// Affected Resource
	d.Section("Affected Resource")
	d.Field("Resource Type", finding.ResourceType())
	d.Field("Resource ID", finding.ResourceId())

	// Description
	if desc := finding.Description(); desc != "" {
		d.Section("Description")
		d.Field("Details", desc)
	}

	// Remediation
	if remediation := finding.Remediation(); remediation != "" {
		d.Section("Remediation")
		d.Field("Recommendation", remediation)
		if url := finding.RemediationUrl(); url != "" {
			d.Field("URL", url)
		}
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

	fields := []render.SummaryField{
		{Label: "Title", Value: finding.Title()},
		{Label: "Severity", Value: finding.Severity()},
		{Label: "Status", Value: finding.Status()},
		{Label: "Product", Value: finding.ProductName()},
	}

	if resourceType := finding.ResourceType(); resourceType != "" {
		fields = append(fields, render.SummaryField{Label: "Resource Type", Value: resourceType})
	}

	return fields
}
