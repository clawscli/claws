package indexes

import (
	"fmt"

	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/render"
)

// VectorIndexRenderer renders S3 Vector Indexes
// Ensure VectorIndexRenderer implements render.Navigator
var _ render.Navigator = (*VectorIndexRenderer)(nil)

type VectorIndexRenderer struct {
	render.BaseRenderer
}

// NewVectorIndexRenderer creates a new VectorIndexRenderer
func NewVectorIndexRenderer() *VectorIndexRenderer {
	return &VectorIndexRenderer{
		BaseRenderer: render.BaseRenderer{
			Service:  "s3vectors",
			Resource: "indexes",
			Cols: []render.Column{
				{Name: "NAME", Width: 35, Getter: getName},
				{Name: "BUCKET", Width: 30, Getter: getBucket},
				{Name: "DIMENSION", Width: 12, Getter: getDimension},
				{Name: "METRIC", Width: 12, Getter: getMetric},
				{Name: "AGE", Width: 10, Getter: getAge},
			},
		},
	}
}

func getName(r dao.Resource) string {
	if index, ok := r.(*VectorIndexResource); ok {
		return index.IndexName()
	}
	return r.GetName()
}

func getBucket(r dao.Resource) string {
	if index, ok := r.(*VectorIndexResource); ok {
		return index.GetBucketName()
	}
	return ""
}

func getDimension(r dao.Resource) string {
	if index, ok := r.(*VectorIndexResource); ok {
		dim := index.Dimension()
		if dim > 0 {
			return fmt.Sprintf("%d", dim)
		}
	}
	return "-"
}

func getMetric(r dao.Resource) string {
	if index, ok := r.(*VectorIndexResource); ok {
		return index.DistanceMetric()
	}
	return ""
}

func getAge(r dao.Resource) string {
	if index, ok := r.(*VectorIndexResource); ok {
		if index.fromDetail && index.Item != nil && index.Item.CreationTime != nil {
			return render.FormatAge(*index.Item.CreationTime)
		}
		if index.Summary != nil && index.Summary.CreationTime != nil {
			return render.FormatAge(*index.Summary.CreationTime)
		}
	}
	return "-"
}

// RenderDetail renders detailed vector index information
func (r *VectorIndexRenderer) RenderDetail(resource dao.Resource) string {
	index, ok := resource.(*VectorIndexResource)
	if !ok {
		return ""
	}

	d := render.NewDetailBuilder()

	d.Title("Vector Index", index.IndexName())

	// Basic Info
	d.Section("Basic Information")
	d.Field("Name", index.IndexName())
	d.Field("ARN", index.GetARN())
	if bucket := index.GetBucketName(); bucket != "" {
		d.Field("Vector Bucket", bucket)
	}

	// Vector Configuration
	d.Section("Vector Configuration")
	if dim := index.Dimension(); dim > 0 {
		d.Field("Dimension", fmt.Sprintf("%d", dim))
	}
	if dataType := index.DataType(); dataType != "" {
		d.Field("Data Type", dataType)
	}
	if metric := index.DistanceMetric(); metric != "" {
		d.Field("Distance Metric", metric)
	}

	// Encryption
	if encType := index.EncryptionType(); encType != "" {
		d.Section("Encryption")
		d.Field("Type", encType)
		if kmsKey := index.KmsKeyArn(); kmsKey != "" {
			d.Field("KMS Key ARN", kmsKey)
		}
	}

	// Metadata Configuration
	if metaKeys := index.NonFilterableMetadataKeys(); len(metaKeys) > 0 {
		d.Section("Metadata Configuration")
		d.Field("Non-Filterable Keys", fmt.Sprintf("%d keys", len(metaKeys)))
		for _, key := range metaKeys {
			d.Field("", key)
		}
	}

	// Timestamps
	d.Section("Timestamps")
	if created := index.CreationDate(); created != "" {
		d.Field("Created", created)
	}

	return d.String()
}

// RenderSummary returns summary fields for the header panel
func (r *VectorIndexRenderer) RenderSummary(resource dao.Resource) []render.SummaryField {
	index, ok := resource.(*VectorIndexResource)
	if !ok {
		return r.BaseRenderer.RenderSummary(resource)
	}

	fields := []render.SummaryField{
		{Label: "Name", Value: index.IndexName()},
		{Label: "ARN", Value: index.GetARN()},
	}

	if bucket := index.GetBucketName(); bucket != "" {
		fields = append(fields, render.SummaryField{Label: "Bucket", Value: bucket})
	}

	if dim := index.Dimension(); dim > 0 {
		fields = append(fields, render.SummaryField{Label: "Dimension", Value: fmt.Sprintf("%d", dim)})
	}

	if metric := index.DistanceMetric(); metric != "" {
		fields = append(fields, render.SummaryField{Label: "Metric", Value: metric})
	}

	if created := index.CreationDate(); created != "" {
		fields = append(fields, render.SummaryField{Label: "Created", Value: created})
	}

	return fields
}

// Navigations returns navigation shortcuts
func (r *VectorIndexRenderer) Navigations(resource dao.Resource) []render.Navigation {
	index, ok := resource.(*VectorIndexResource)
	if !ok {
		return nil
	}

	var navs []render.Navigation

	// Navigate back to bucket
	if bucket := index.GetBucketName(); bucket != "" {
		navs = append(navs, render.Navigation{
			Key:         "b",
			Label:       "Bucket",
			Service:     "s3vectors",
			Resource:    "buckets",
			FilterField: "Name",
			FilterValue: bucket,
		})
	}

	return navs
}
