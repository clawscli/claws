package buckets

import (
	"fmt"

	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/render"
)

// BucketRenderer renders S3 buckets
type BucketRenderer struct {
	render.BaseRenderer
}

// NewBucketRenderer creates a new BucketRenderer
func NewBucketRenderer() render.Renderer {
	return &BucketRenderer{
		BaseRenderer: render.BaseRenderer{
			Service:  "s3",
			Resource: "buckets",
			Cols: []render.Column{
				{
					Name:  "NAME",
					Width: 50,
					Getter: func(r dao.Resource) string {
						return r.GetName()
					},
					Priority: 0,
				},
				{
					Name:  "REGION",
					Width: 15,
					Getter: func(r dao.Resource) string {
						if b, ok := r.(*BucketResource); ok {
							return b.Region
						}
						return ""
					},
					Priority: 1,
				},
				{
					Name:  "CREATED",
					Width: 20,
					Getter: func(r dao.Resource) string {
						if b, ok := r.(*BucketResource); ok {
							if !b.CreationDate.IsZero() {
								return b.CreationDate.Format("2006-01-02 15:04")
							}
						}
						return ""
					},
					Priority: 2,
				},
				{
					Name:  "AGE",
					Width: 10,
					Getter: func(r dao.Resource) string {
						if b, ok := r.(*BucketResource); ok {
							return render.FormatAge(b.CreationDate)
						}
						return ""
					},
					Priority: 3,
				},
				render.TagsColumn(35, 4),
			},
		},
	}
}

// RenderDetail renders detailed bucket information
func (r *BucketRenderer) RenderDetail(resource dao.Resource) string {
	b, ok := resource.(*BucketResource)
	if !ok {
		return ""
	}

	d := render.NewDetailBuilder()

	d.Title("S3 Bucket", b.GetName())

	// Basic Info
	d.Section("Basic Information")
	d.Field("Bucket Name", b.BucketName)
	d.Field("Region", b.Region)

	// URIs
	d.Section("Access")
	d.Field("ARN", fmt.Sprintf("arn:aws:s3:::%s", b.BucketName))
	d.Field("S3 URI", fmt.Sprintf("s3://%s", b.BucketName))

	// Website endpoint (if standard naming)
	if b.Region != "" {
		websiteEndpoint := fmt.Sprintf("%s.s3.%s.amazonaws.com", b.BucketName, b.Region)
		d.Field("Endpoint", websiteEndpoint)
	}

	// Versioning
	d.Section("Versioning")
	if b.Versioning != "" {
		d.Field("Status", b.Versioning)
	} else {
		d.Field("Status", "Not configured")
	}
	if b.MFADelete != "" {
		d.Field("MFA Delete", b.MFADelete)
	}

	// Encryption
	d.Section("Server-Side Encryption")
	if b.EncryptionEnabled {
		d.Field("Status", "Enabled")
		d.Field("Algorithm", b.EncryptionAlgorithm)
		if b.EncryptionKMSKeyID != "" {
			d.Field("KMS Key ID", b.EncryptionKMSKeyID)
		}
		if b.BucketKeyEnabled {
			d.Field("Bucket Key", "Enabled")
		}
	} else {
		d.Field("Status", "Not configured")
	}

	// Public Access Block
	d.Section("Block Public Access")
	if b.PublicAccessBlock != nil {
		pab := b.PublicAccessBlock
		allBlocked := pab.BlockPublicAcls && pab.IgnorePublicAcls && pab.BlockPublicPolicy && pab.RestrictPublicBuckets
		if allBlocked {
			d.Field("Status", "All public access blocked")
		} else {
			if pab.BlockPublicAcls {
				d.Field("Block Public ACLs", "On")
			} else {
				d.Field("Block Public ACLs", "Off")
			}
			if pab.IgnorePublicAcls {
				d.Field("Ignore Public ACLs", "On")
			} else {
				d.Field("Ignore Public ACLs", "Off")
			}
			if pab.BlockPublicPolicy {
				d.Field("Block Public Policy", "On")
			} else {
				d.Field("Block Public Policy", "Off")
			}
			if pab.RestrictPublicBuckets {
				d.Field("Restrict Public Buckets", "On")
			} else {
				d.Field("Restrict Public Buckets", "Off")
			}
		}
	} else {
		d.Field("Status", "Not configured")
	}

	// Object Lock
	if b.ObjectLockEnabled {
		d.Section("Object Lock")
		d.Field("Status", "Enabled")
		if b.ObjectLockMode != "" {
			d.Field("Default Mode", b.ObjectLockMode)
		}
		if b.ObjectLockRetention != "" {
			d.Field("Default Retention", b.ObjectLockRetention)
		}
	}

	// Lifecycle Rules
	if b.LifecycleRulesCount > 0 {
		d.Section("Lifecycle")
		d.Field("Rules", fmt.Sprintf("%d lifecycle rules configured", b.LifecycleRulesCount))
	}

	// Timestamps (only shown if creation date is available)
	if !b.CreationDate.IsZero() {
		d.Section("Timestamps")
		d.Field("Created", b.CreationDate.Format("2006-01-02 15:04:05"))
		d.Field("Age", render.FormatAge(b.CreationDate))
	}

	// Tags
	d.Tags(b.GetTags())

	return d.String()
}

// RenderSummary returns summary fields for the header panel
func (r *BucketRenderer) RenderSummary(resource dao.Resource) []render.SummaryField {
	b, ok := resource.(*BucketResource)
	if !ok {
		return nil
	}

	// Row 1: Bucket name and region
	fields := []render.SummaryField{
		{Label: "Bucket", Value: b.GetName()},
		{Label: "Region", Value: b.Region},
	}

	// Versioning (if fetched)
	if b.Versioning != "" {
		fields = append(fields, render.SummaryField{Label: "Versioning", Value: b.Versioning})
	}

	// Encryption (if fetched)
	if b.EncryptionEnabled {
		fields = append(fields, render.SummaryField{Label: "Encryption", Value: b.EncryptionAlgorithm})
	}

	// Public Access Block (if fetched)
	if b.PublicAccessBlock != nil {
		pab := b.PublicAccessBlock
		allBlocked := pab.BlockPublicAcls && pab.IgnorePublicAcls && pab.BlockPublicPolicy && pab.RestrictPublicBuckets
		if allBlocked {
			fields = append(fields, render.SummaryField{Label: "Public Access", Value: "Blocked"})
		} else {
			fields = append(fields, render.SummaryField{Label: "Public Access", Value: "Partial"})
		}
	}

	// Object Lock (if enabled)
	if b.ObjectLockEnabled {
		fields = append(fields, render.SummaryField{Label: "Object Lock", Value: "Enabled"})
	}

	// Lifecycle rules count (if fetched)
	if b.LifecycleRulesCount > 0 {
		fields = append(fields, render.SummaryField{
			Label: "Lifecycle Rules",
			Value: fmt.Sprintf("%d", b.LifecycleRulesCount),
		})
	}

	// Creation info
	if !b.CreationDate.IsZero() {
		fields = append(fields, render.SummaryField{
			Label: "Created",
			Value: b.CreationDate.Format("2006-01-02 15:04"),
		})
		fields = append(fields, render.SummaryField{
			Label: "Age",
			Value: render.FormatAge(b.CreationDate),
		})
	}

	return fields
}
