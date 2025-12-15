package repositories

import (
	"time"

	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/render"
)

// RepositoryRenderer renders ECR repositories with custom columns
// Ensure RepositoryRenderer implements render.Navigator
var _ render.Navigator = (*RepositoryRenderer)(nil)

type RepositoryRenderer struct {
	render.BaseRenderer
}

// NewRepositoryRenderer creates a new RepositoryRenderer
func NewRepositoryRenderer() render.Renderer {
	return &RepositoryRenderer{
		BaseRenderer: render.BaseRenderer{
			Service:  "ecr",
			Resource: "repositories",
			Cols: []render.Column{
				{
					Name:  "NAME",
					Width: 35,
					Getter: func(r dao.Resource) string {
						return r.GetID()
					},
					Priority: 0,
				},
				{
					Name:  "URI",
					Width: 60,
					Getter: func(r dao.Resource) string {
						if rr, ok := r.(*RepositoryResource); ok {
							return rr.URI()
						}
						return ""
					},
					Priority: 1,
				},
				{
					Name:  "TAG MUTABILITY",
					Width: 15,
					Getter: func(r dao.Resource) string {
						if rr, ok := r.(*RepositoryResource); ok {
							return rr.ImageTagMutability()
						}
						return ""
					},
					Priority: 2,
				},
				{
					Name:  "SCAN",
					Width: 6,
					Getter: func(r dao.Resource) string {
						if rr, ok := r.(*RepositoryResource); ok {
							if rr.ScanOnPush() {
								return "Yes"
							}
							return "No"
						}
						return ""
					},
					Priority: 3,
				},
				{
					Name:  "ENCRYPTION",
					Width: 12,
					Getter: func(r dao.Resource) string {
						if rr, ok := r.(*RepositoryResource); ok {
							return rr.EncryptionType()
						}
						return ""
					},
					Priority: 4,
				},
				{
					Name:  "AGE",
					Width: 8,
					Getter: func(r dao.Resource) string {
						if rr, ok := r.(*RepositoryResource); ok {
							if rr.Item.CreatedAt != nil {
								return render.FormatAge(*rr.Item.CreatedAt)
							}
						}
						return ""
					},
					Priority: 5,
				},
			},
		},
	}
}

// RenderDetail renders detailed repository information
func (r *RepositoryRenderer) RenderDetail(resource dao.Resource) string {
	rr, ok := resource.(*RepositoryResource)
	if !ok {
		return ""
	}

	d := render.NewDetailBuilder()

	d.Title("ECR Repository", rr.GetID())

	// Basic Info
	d.Section("Basic Information")
	d.Field("Name", rr.GetID())
	d.Field("URI", rr.URI())
	d.Field("ARN", rr.ARN())
	if rr.Item.CreatedAt != nil {
		d.Field("Created", rr.Item.CreatedAt.Format(time.RFC3339))
		d.Field("Age", render.FormatAge(*rr.Item.CreatedAt))
	}

	// Configuration
	d.Section("Configuration")
	d.Field("Image Tag Mutability", rr.ImageTagMutability())
	if rr.ScanOnPush() {
		d.Field("Scan on Push", "Enabled")
	} else {
		d.Field("Scan on Push", "Disabled")
	}

	// Encryption
	d.Section("Encryption")
	d.Field("Encryption Type", rr.EncryptionType())
	if rr.Item.EncryptionConfiguration != nil && rr.Item.EncryptionConfiguration.KmsKey != nil {
		d.Field("KMS Key", *rr.Item.EncryptionConfiguration.KmsKey)
	}

	return d.String()
}

// RenderSummary returns summary fields for the header panel
func (r *RepositoryRenderer) RenderSummary(resource dao.Resource) []render.SummaryField {
	rr, ok := resource.(*RepositoryResource)
	if !ok {
		return nil
	}

	fields := []render.SummaryField{
		{Label: "Name", Value: rr.GetID()},
		{Label: "URI", Value: rr.URI()},
	}

	fields = append(fields, render.SummaryField{Label: "Tag Mutability", Value: rr.ImageTagMutability()})

	scanStatus := "Disabled"
	if rr.ScanOnPush() {
		scanStatus = "Enabled"
	}
	fields = append(fields, render.SummaryField{Label: "Scan on Push", Value: scanStatus})

	fields = append(fields, render.SummaryField{Label: "Encryption", Value: rr.EncryptionType()})

	if rr.Item.CreatedAt != nil {
		fields = append(fields, render.SummaryField{
			Label: "Created",
			Value: rr.Item.CreatedAt.Format("2006-01-02 15:04") + " (" + render.FormatAge(*rr.Item.CreatedAt) + ")",
		})
	}

	return fields
}

// Navigations returns navigation shortcuts
func (r *RepositoryRenderer) Navigations(resource dao.Resource) []render.Navigation {
	rr, ok := resource.(*RepositoryResource)
	if !ok {
		return nil
	}

	return []render.Navigation{
		{
			Key: "i", Label: "Images", Service: "ecr", Resource: "images",
			FilterField: "RepositoryName", FilterValue: rr.GetID(),
		},
	}
}
