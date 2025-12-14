package topics

import (
	"bytes"
	"encoding/json"

	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/render"
)

// Ensure TopicRenderer implements render.Navigator
var _ render.Navigator = (*TopicRenderer)(nil)

// TopicRenderer renders SNS topics with custom columns
type TopicRenderer struct {
	render.BaseRenderer
}

// NewTopicRenderer creates a new TopicRenderer
func NewTopicRenderer() render.Renderer {
	return &TopicRenderer{
		BaseRenderer: render.BaseRenderer{
			Service:  "sns",
			Resource: "topics",
			Cols: []render.Column{
				{
					Name:  "NAME",
					Width: 40,
					Getter: func(r dao.Resource) string {
						return r.GetName()
					},
					Priority: 0,
				},
				{
					Name:  "DISPLAY NAME",
					Width: 25,
					Getter: func(r dao.Resource) string {
						if tr, ok := r.(*TopicResource); ok {
							return tr.DisplayName()
						}
						return ""
					},
					Priority: 1,
				},
				{
					Name:  "SUBS",
					Width: 6,
					Getter: func(r dao.Resource) string {
						if tr, ok := r.(*TopicResource); ok {
							return tr.SubscriptionCount()
						}
						return ""
					},
					Priority: 2,
				},
				{
					Name:  "PENDING",
					Width: 8,
					Getter: func(r dao.Resource) string {
						if tr, ok := r.(*TopicResource); ok {
							return tr.PendingSubscriptions()
						}
						return ""
					},
					Priority: 3,
				},
				{
					Name:  "FIFO",
					Width: 5,
					Getter: func(r dao.Resource) string {
						if tr, ok := r.(*TopicResource); ok {
							if tr.IsFIFO() {
								return "Yes"
							}
							return "No"
						}
						return ""
					},
					Priority: 4,
				},
				{
					Name:  "OWNER",
					Width: 14,
					Getter: func(r dao.Resource) string {
						if tr, ok := r.(*TopicResource); ok {
							return tr.Owner()
						}
						return ""
					},
					Priority: 5,
				},
			},
		},
	}
}

// RenderDetail renders detailed topic information
func (r *TopicRenderer) RenderDetail(resource dao.Resource) string {
	tr, ok := resource.(*TopicResource)
	if !ok {
		return ""
	}

	d := render.NewDetailBuilder()

	d.Title("SNS Topic", tr.GetName())

	// Basic Info
	d.Section("Basic Information")
	d.Field("Name", tr.GetName())
	d.Field("ARN", tr.ARN())
	if tr.DisplayName() != "" {
		d.Field("Display Name", tr.DisplayName())
	}
	d.Field("Owner", tr.Owner())

	// Configuration
	d.Section("Configuration")
	if tr.IsFIFO() {
		d.Field("Type", "FIFO")
		if dedup, ok := tr.Attrs["ContentBasedDeduplication"]; ok {
			d.Field("Content-Based Deduplication", dedup)
		}
	} else {
		d.Field("Type", "Standard")
	}

	// Subscriptions
	d.Section("Subscriptions")
	d.Field("Confirmed", tr.SubscriptionCount())
	d.Field("Pending", tr.PendingSubscriptions())
	if deleted, ok := tr.Attrs["SubscriptionsDeleted"]; ok {
		d.Field("Deleted", deleted)
	}

	// Delivery Policy
	if _, ok := tr.Attrs["EffectiveDeliveryPolicy"]; ok {
		d.Section("Delivery Policy")
		d.Field("Effective Policy", "Configured")
	}

	// Delivery Status Logging
	hasLogging := false
	loggingSection := func() {
		if !hasLogging {
			d.Section("Delivery Status Logging")
			hasLogging = true
		}
	}
	if role, ok := tr.Attrs["HTTPSuccessFeedbackRoleArn"]; ok && role != "" {
		loggingSection()
		d.FieldStyled("HTTP", "Enabled", render.SuccessStyle())
	}
	if role, ok := tr.Attrs["LambdaSuccessFeedbackRoleArn"]; ok && role != "" {
		loggingSection()
		d.FieldStyled("Lambda", "Enabled", render.SuccessStyle())
	}
	if role, ok := tr.Attrs["SQSSuccessFeedbackRoleArn"]; ok && role != "" {
		loggingSection()
		d.FieldStyled("SQS", "Enabled", render.SuccessStyle())
	}
	if role, ok := tr.Attrs["FirehoseSuccessFeedbackRoleArn"]; ok && role != "" {
		loggingSection()
		d.FieldStyled("Firehose", "Enabled", render.SuccessStyle())
	}
	if role, ok := tr.Attrs["ApplicationSuccessFeedbackRoleArn"]; ok && role != "" {
		loggingSection()
		d.FieldStyled("Application", "Enabled", render.SuccessStyle())
	}

	// Encryption
	d.Section("Encryption")
	if kmsKey, ok := tr.Attrs["KmsMasterKeyId"]; ok && kmsKey != "" {
		d.Field("KMS Key ID", kmsKey)
	} else {
		d.Field("Server-Side Encryption", "Disabled")
	}

	// Access Policy
	if policy, ok := tr.Attrs["Policy"]; ok && policy != "" {
		d.Section("Access Policy")
		d.Line(prettyJSON(policy))
	}

	return d.String()
}

// prettyJSON formats JSON string with indentation
func prettyJSON(s string) string {
	var buf bytes.Buffer
	if err := json.Indent(&buf, []byte(s), "", "  "); err != nil {
		return s
	}
	return buf.String()
}

// RenderSummary returns summary fields for the header panel
func (r *TopicRenderer) RenderSummary(resource dao.Resource) []render.SummaryField {
	tr, ok := resource.(*TopicResource)
	if !ok {
		return nil
	}

	fields := []render.SummaryField{
		{Label: "Name", Value: tr.GetName()},
		{Label: "ARN", Value: tr.ARN()},
	}

	if tr.DisplayName() != "" {
		fields = append(fields, render.SummaryField{Label: "Display Name", Value: tr.DisplayName()})
	}

	topicType := "Standard"
	if tr.IsFIFO() {
		topicType = "FIFO"
	}
	fields = append(fields, render.SummaryField{Label: "Type", Value: topicType})

	fields = append(fields, render.SummaryField{Label: "Subscriptions", Value: tr.SubscriptionCount()})
	fields = append(fields, render.SummaryField{Label: "Owner", Value: tr.Owner()})

	return fields
}

// Navigations returns navigation shortcuts for SNS topics
func (r *TopicRenderer) Navigations(resource dao.Resource) []render.Navigation {
	tr, ok := resource.(*TopicResource)
	if !ok {
		return nil
	}

	var navs []render.Navigation

	// Subscriptions navigation
	navs = append(navs, render.Navigation{
		Key: "s", Label: "Subscriptions", Service: "sns", Resource: "subscriptions",
		FilterField: "TopicArn", FilterValue: tr.ARN(),
	})

	return navs
}
