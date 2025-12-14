package buckets

import (
	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/render"
)

// VectorBucketRenderer renders S3 Vector Buckets
// Ensure VectorBucketRenderer implements render.Navigator
var _ render.Navigator = (*VectorBucketRenderer)(nil)

type VectorBucketRenderer struct {
	render.BaseRenderer
}

// NewVectorBucketRenderer creates a new VectorBucketRenderer
func NewVectorBucketRenderer() *VectorBucketRenderer {
	return &VectorBucketRenderer{
		BaseRenderer: render.BaseRenderer{
			Service:  "s3vectors",
			Resource: "buckets",
			Cols: []render.Column{
				{Name: "NAME", Width: 50, Getter: getName},
				{Name: "ENCRYPTION", Width: 12, Getter: getEncryption},
				{Name: "CREATED", Width: 20, Getter: getCreated},
				{Name: "AGE", Width: 10, Getter: getAge},
			},
		},
	}
}

func getName(r dao.Resource) string {
	if bucket, ok := r.(*VectorBucketResource); ok {
		return bucket.BucketName()
	}
	return r.GetName()
}

func getEncryption(r dao.Resource) string {
	if bucket, ok := r.(*VectorBucketResource); ok {
		return bucket.EncryptionType()
	}
	return ""
}

func getCreated(r dao.Resource) string {
	if bucket, ok := r.(*VectorBucketResource); ok {
		return bucket.CreationDate()
	}
	return ""
}

func getAge(r dao.Resource) string {
	if bucket, ok := r.(*VectorBucketResource); ok {
		if bucket.fromDetail && bucket.Item != nil && bucket.Item.CreationTime != nil {
			return render.FormatAge(*bucket.Item.CreationTime)
		}
		if bucket.Summary != nil && bucket.Summary.CreationTime != nil {
			return render.FormatAge(*bucket.Summary.CreationTime)
		}
	}
	return "-"
}

// RenderDetail renders detailed vector bucket information
func (r *VectorBucketRenderer) RenderDetail(resource dao.Resource) string {
	bucket, ok := resource.(*VectorBucketResource)
	if !ok {
		return ""
	}

	d := render.NewDetailBuilder()

	d.Title("Vector Bucket", bucket.BucketName())

	// Basic Info
	d.Section("Basic Information")
	d.Field("Name", bucket.BucketName())
	d.Field("ARN", bucket.GetARN())

	// Encryption
	d.Section("Encryption")
	d.Field("Type", bucket.EncryptionType())
	if kmsKey := bucket.KmsKeyArn(); kmsKey != "" {
		d.Field("KMS Key ARN", kmsKey)
	}

	// Timestamps
	d.Section("Timestamps")
	if created := bucket.CreationDate(); created != "" {
		d.Field("Created", created)
	}

	return d.String()
}

// RenderSummary returns summary fields for the header panel
func (r *VectorBucketRenderer) RenderSummary(resource dao.Resource) []render.SummaryField {
	bucket, ok := resource.(*VectorBucketResource)
	if !ok {
		return r.BaseRenderer.RenderSummary(resource)
	}

	fields := []render.SummaryField{
		{Label: "Name", Value: bucket.BucketName()},
		{Label: "ARN", Value: bucket.GetARN()},
		{Label: "Encryption", Value: bucket.EncryptionType()},
	}

	if created := bucket.CreationDate(); created != "" {
		fields = append(fields, render.SummaryField{Label: "Created", Value: created})
	}

	return fields
}

// Navigations returns navigation shortcuts
func (r *VectorBucketRenderer) Navigations(resource dao.Resource) []render.Navigation {
	bucket, ok := resource.(*VectorBucketResource)
	if !ok {
		return nil
	}

	return []render.Navigation{
		{
			Key:         "i",
			Label:       "Indexes",
			Service:     "s3vectors",
			Resource:    "indexes",
			FilterField: "VectorBucketName",
			FilterValue: bucket.BucketName(),
		},
	}
}
