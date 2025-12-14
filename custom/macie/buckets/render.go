package buckets

import (
	"fmt"

	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/render"
)

// BucketRenderer renders Macie buckets.
// Ensure BucketRenderer implements render.Navigator
var _ render.Navigator = (*BucketRenderer)(nil)

type BucketRenderer struct {
	render.BaseRenderer
}

// NewBucketRenderer creates a new BucketRenderer.
func NewBucketRenderer() render.Renderer {
	return &BucketRenderer{
		BaseRenderer: render.BaseRenderer{
			Service:  "macie",
			Resource: "buckets",
			Cols: []render.Column{
				{Name: "BUCKET NAME", Width: 40, Getter: func(r dao.Resource) string { return r.GetID() }},
				{Name: "REGION", Width: 15, Getter: getRegion},
				{Name: "OBJECTS", Width: 12, Getter: getObjects},
				{Name: "SIZE", Width: 15, Getter: getSize},
			},
		},
	}
}

func getRegion(r dao.Resource) string {
	bucket, ok := r.(*BucketResource)
	if !ok {
		return ""
	}
	return bucket.Region()
}

func getObjects(r dao.Resource) string {
	bucket, ok := r.(*BucketResource)
	if !ok {
		return ""
	}
	return fmt.Sprintf("%d", bucket.ClassifiableObjectCount())
}

func getSize(r dao.Resource) string {
	bucket, ok := r.(*BucketResource)
	if !ok {
		return ""
	}
	return formatBytes(bucket.SizeInBytes())
}

// formatBytes formats bytes to human-readable format.
func formatBytes(bytes int64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
		TB = GB * 1024
	)

	switch {
	case bytes >= TB:
		return fmt.Sprintf("%.2f TB", float64(bytes)/float64(TB))
	case bytes >= GB:
		return fmt.Sprintf("%.2f GB", float64(bytes)/float64(GB))
	case bytes >= MB:
		return fmt.Sprintf("%.2f MB", float64(bytes)/float64(MB))
	case bytes >= KB:
		return fmt.Sprintf("%.2f KB", float64(bytes)/float64(KB))
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}

// RenderDetail renders the detail view for a bucket.
func (r *BucketRenderer) RenderDetail(resource dao.Resource) string {
	bucket, ok := resource.(*BucketResource)
	if !ok {
		return ""
	}

	d := render.NewDetailBuilder()

	d.Title("Macie Bucket", bucket.Name())

	// Basic Info
	d.Section("Basic Information")
	d.Field("Bucket Name", bucket.Name())
	d.Field("ARN", bucket.GetARN())
	d.Field("Account ID", bucket.AccountId())
	d.Field("Region", bucket.Region())

	// Statistics
	d.Section("Statistics")
	d.Field("Classifiable Objects", fmt.Sprintf("%d", bucket.ClassifiableObjectCount()))
	d.Field("Size", formatBytes(bucket.SizeInBytes()))

	return d.String()
}

// RenderSummary renders summary fields for a bucket.
func (r *BucketRenderer) RenderSummary(resource dao.Resource) []render.SummaryField {
	bucket, ok := resource.(*BucketResource)
	if !ok {
		return r.BaseRenderer.RenderSummary(resource)
	}

	return []render.SummaryField{
		{Label: "Bucket Name", Value: bucket.Name()},
		{Label: "Region", Value: bucket.Region()},
		{Label: "Size", Value: formatBytes(bucket.SizeInBytes())},
	}
}

// Navigations returns available navigations from a bucket.
func (r *BucketRenderer) Navigations(resource dao.Resource) []render.Navigation {
	bucket, ok := resource.(*BucketResource)
	if !ok {
		return nil
	}
	return []render.Navigation{
		{
			Key:         "f",
			Label:       "Findings",
			Service:     "macie",
			Resource:    "findings",
			FilterField: "BucketName",
			FilterValue: bucket.Name(),
		},
	}
}
