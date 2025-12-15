package trails

import (
	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/render"
)

// TrailRenderer renders CloudTrail trails.
type TrailRenderer struct {
	render.BaseRenderer
}

// NewTrailRenderer creates a new TrailRenderer.
func NewTrailRenderer() render.Renderer {
	return &TrailRenderer{
		BaseRenderer: render.BaseRenderer{
			Service:  "cloudtrail",
			Resource: "trails",
			Cols: []render.Column{
				{Name: "TRAIL NAME", Width: 35, Getter: func(r dao.Resource) string { return r.GetID() }},
				{Name: "S3 BUCKET", Width: 35, Getter: getS3Bucket},
				{Name: "MULTI-REGION", Width: 14, Getter: getMultiRegion},
				{Name: "ORG TRAIL", Width: 11, Getter: getOrgTrail},
				{Name: "HOME REGION", Width: 15, Getter: getHomeRegion},
			},
		},
	}
}

func getS3Bucket(r dao.Resource) string {
	trail, ok := r.(*TrailResource)
	if !ok {
		return ""
	}
	return trail.S3BucketName()
}

func getMultiRegion(r dao.Resource) string {
	trail, ok := r.(*TrailResource)
	if !ok {
		return ""
	}
	if trail.IsMultiRegionTrail() {
		return "Yes"
	}
	return "No"
}

func getOrgTrail(r dao.Resource) string {
	trail, ok := r.(*TrailResource)
	if !ok {
		return ""
	}
	if trail.IsOrganizationTrail() {
		return "Yes"
	}
	return "No"
}

func getHomeRegion(r dao.Resource) string {
	trail, ok := r.(*TrailResource)
	if !ok {
		return ""
	}
	return trail.HomeRegion()
}

// RenderDetail renders the detail view for a trail.
func (r *TrailRenderer) RenderDetail(resource dao.Resource) string {
	trail, ok := resource.(*TrailResource)
	if !ok {
		return ""
	}

	d := render.NewDetailBuilder()

	d.Title("CloudTrail Trail", trail.Name())

	// Basic Info
	d.Section("Basic Information")
	d.Field("Trail Name", trail.Name())
	d.Field("ARN", trail.GetARN())
	d.Field("Home Region", trail.HomeRegion())

	// Trail Configuration
	d.Section("Trail Configuration")
	d.Field("Multi-Region Trail", formatBool(trail.IsMultiRegionTrail()))
	d.Field("Organization Trail", formatBool(trail.IsOrganizationTrail()))
	d.Field("Include Global Events", formatBool(trail.IncludeGlobalServiceEvents()))
	d.Field("Log File Validation", formatBool(trail.LogFileValidationEnabled()))

	// S3 Configuration
	d.Section("S3 Configuration")
	d.Field("S3 Bucket", trail.S3BucketName())
	if prefix := trail.S3KeyPrefix(); prefix != "" {
		d.Field("S3 Key Prefix", prefix)
	}

	// CloudWatch Logs Integration
	if logGroup := trail.CloudWatchLogsLogGroupArn(); logGroup != "" {
		d.Section("CloudWatch Logs Integration")
		d.Field("Log Group ARN", logGroup)
		if roleArn := trail.CloudWatchLogsRoleArn(); roleArn != "" {
			d.Field("Role ARN", roleArn)
		}
	}

	// SNS Notification
	if topic := trail.SnsTopicARN(); topic != "" {
		d.Section("SNS Notification")
		d.Field("Topic ARN", topic)
	}

	// Encryption
	if kmsKey := trail.KMSKeyId(); kmsKey != "" {
		d.Section("Encryption")
		d.Field("KMS Key ID", kmsKey)
	}

	// Advanced Features
	d.Section("Advanced Features")
	d.Field("Custom Event Selectors", formatBool(trail.HasCustomEventSelectors()))
	d.Field("Insight Selectors", formatBool(trail.HasInsightSelectors()))

	return d.String()
}

// RenderSummary renders summary fields for a trail.
func (r *TrailRenderer) RenderSummary(resource dao.Resource) []render.SummaryField {
	trail, ok := resource.(*TrailResource)
	if !ok {
		return r.BaseRenderer.RenderSummary(resource)
	}

	fields := []render.SummaryField{
		{Label: "Trail Name", Value: trail.Name()},
		{Label: "ARN", Value: trail.GetARN()},
		{Label: "S3 Bucket", Value: trail.S3BucketName()},
		{Label: "Home Region", Value: trail.HomeRegion()},
	}

	if trail.IsMultiRegionTrail() {
		fields = append(fields, render.SummaryField{Label: "Multi-Region", Value: "Yes"})
	} else {
		fields = append(fields, render.SummaryField{Label: "Multi-Region", Value: "No"})
	}

	if trail.IsOrganizationTrail() {
		fields = append(fields, render.SummaryField{Label: "Organization Trail", Value: "Yes"})
	}

	if kmsKey := trail.KMSKeyId(); kmsKey != "" {
		fields = append(fields, render.SummaryField{Label: "Encrypted", Value: "Yes"})
	}

	return fields
}

func formatBool(b bool) string {
	if b {
		return "Yes"
	}
	return "No"
}
