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
			Service:  "macie2",
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
	return render.FormatSize(bucket.SizeInBytes())
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
	d.Field("Size", render.FormatSize(bucket.SizeInBytes()))

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
		{Label: "Size", Value: render.FormatSize(bucket.SizeInBytes())},
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
			Service:     "macie2",
			Resource:    "findings",
			FilterField: "BucketName",
			FilterValue: bucket.Name(),
		},
	}
}
