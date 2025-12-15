package streams

import (
	"fmt"
	"strings"

	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/render"
)

// StreamRenderer renders Kinesis streams
// Ensure StreamRenderer implements render.Navigator
var _ render.Navigator = (*StreamRenderer)(nil)

type StreamRenderer struct {
	render.BaseRenderer
}

// NewStreamRenderer creates a new StreamRenderer
func NewStreamRenderer() *StreamRenderer {
	return &StreamRenderer{
		BaseRenderer: render.BaseRenderer{
			Service:  "kinesis",
			Resource: "streams",
			Cols: []render.Column{
				{Name: "NAME", Width: 30, Getter: getStreamName},
				{Name: "STATUS", Width: 10, Getter: getStatus},
				{Name: "MODE", Width: 12, Getter: getMode},
				{Name: "SHARDS", Width: 8, Getter: getShards},
				{Name: "RETENTION", Width: 10, Getter: getRetention},
				{Name: "AGE", Width: 12, Getter: getAge},
			},
		},
	}
}

func getStreamName(r dao.Resource) string {
	if stream, ok := r.(*StreamResource); ok {
		return stream.StreamName()
	}
	return ""
}

func getStatus(r dao.Resource) string {
	if stream, ok := r.(*StreamResource); ok {
		return stream.Status()
	}
	return ""
}

func getMode(r dao.Resource) string {
	if stream, ok := r.(*StreamResource); ok {
		return stream.StreamMode()
	}
	return ""
}

func getShards(r dao.Resource) string {
	if stream, ok := r.(*StreamResource); ok {
		count := stream.ShardCount()
		if count > 0 {
			return fmt.Sprintf("%d", count)
		}
		return "-"
	}
	return ""
}

func getRetention(r dao.Resource) string {
	if stream, ok := r.(*StreamResource); ok {
		hours := stream.RetentionPeriodHours()
		if hours > 0 {
			return fmt.Sprintf("%dh", hours)
		}
	}
	return ""
}

func getAge(r dao.Resource) string {
	if stream, ok := r.(*StreamResource); ok {
		if stream.Summary != nil && stream.Summary.StreamCreationTimestamp != nil {
			return render.FormatAge(*stream.Summary.StreamCreationTimestamp)
		}
	}
	return "-"
}

// RenderDetail renders detailed stream information
func (r *StreamRenderer) RenderDetail(resource dao.Resource) string {
	stream, ok := resource.(*StreamResource)
	if !ok {
		return ""
	}

	d := render.NewDetailBuilder()

	d.Title("Kinesis Stream", stream.StreamName())

	// Basic Info
	d.Section("Basic Information")
	d.Field("Stream Name", stream.StreamName())
	d.Field("ARN", stream.GetARN())
	d.Field("Status", stream.Status())

	// Configuration
	d.Section("Configuration")
	d.Field("Stream Mode", stream.StreamMode())
	if shards := stream.ShardCount(); shards > 0 {
		d.Field("Open Shards", fmt.Sprintf("%d", shards))
	}
	d.Field("Retention Period", fmt.Sprintf("%d hours", stream.RetentionPeriodHours()))

	// Consumers
	if consumers := stream.ConsumerCount(); consumers > 0 {
		d.Field("Consumers", fmt.Sprintf("%d", consumers))
	}

	// Encryption
	d.Section("Encryption")
	encType := stream.EncryptionType()
	if encType != "" && encType != "NONE" {
		d.Field("Encryption Type", encType)
		if keyId := stream.KeyId(); keyId != "" {
			d.Field("KMS Key", keyId)
		}
	} else {
		d.Field("Encryption", "Disabled")
	}

	// Enhanced Monitoring
	if metrics := stream.EnhancedMonitoring(); len(metrics) > 0 {
		d.Section("Enhanced Monitoring")
		d.Field("Shard-Level Metrics", strings.Join(metrics, ", "))
	}

	// Timestamps
	d.Section("Timestamps")
	if created := stream.CreatedAt(); created != "" {
		d.Field("Created", created)
	}

	return d.String()
}

// RenderSummary returns summary fields for the header panel
func (r *StreamRenderer) RenderSummary(resource dao.Resource) []render.SummaryField {
	stream, ok := resource.(*StreamResource)
	if !ok {
		return r.BaseRenderer.RenderSummary(resource)
	}

	fields := []render.SummaryField{
		{Label: "Stream Name", Value: stream.StreamName()},
		{Label: "ARN", Value: stream.GetARN()},
		{Label: "Status", Value: stream.Status()},
		{Label: "Mode", Value: stream.StreamMode()},
	}

	if shards := stream.ShardCount(); shards > 0 {
		fields = append(fields, render.SummaryField{Label: "Shards", Value: fmt.Sprintf("%d", shards)})
	}

	fields = append(fields, render.SummaryField{
		Label: "Retention",
		Value: fmt.Sprintf("%d hours", stream.RetentionPeriodHours()),
	})

	if created := stream.CreatedAt(); created != "" {
		fields = append(fields, render.SummaryField{Label: "Created", Value: created})
	}

	return fields
}

// Navigations returns navigation shortcuts
func (r *StreamRenderer) Navigations(resource dao.Resource) []render.Navigation {
	// No navigations for now
	return nil
}
