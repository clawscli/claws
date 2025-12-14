package subscriptions

import (
	"bytes"
	"encoding/json"
	"strings"

	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/render"
)

// Ensure SubscriptionRenderer implements render.Navigator
var _ render.Navigator = (*SubscriptionRenderer)(nil)

// SubscriptionRenderer renders SNS subscriptions with custom columns
type SubscriptionRenderer struct {
	render.BaseRenderer
}

// NewSubscriptionRenderer creates a new SubscriptionRenderer
func NewSubscriptionRenderer() render.Renderer {
	return &SubscriptionRenderer{
		BaseRenderer: render.BaseRenderer{
			Service:  "sns",
			Resource: "subscriptions",
			Cols: []render.Column{
				{
					Name:  "TOPIC",
					Width: 30,
					Getter: func(r dao.Resource) string {
						if sr, ok := r.(*SubscriptionResource); ok {
							return sr.TopicName()
						}
						return ""
					},
					Priority: 0,
				},
				{
					Name:  "PROTOCOL",
					Width: 12,
					Getter: func(r dao.Resource) string {
						if sr, ok := r.(*SubscriptionResource); ok {
							return sr.Protocol()
						}
						return ""
					},
					Priority: 1,
				},
				{
					Name:  "ENDPOINT",
					Width: 50,
					Getter: func(r dao.Resource) string {
						if sr, ok := r.(*SubscriptionResource); ok {
							return sr.Endpoint()
						}
						return ""
					},
					Priority: 2,
				},
				{
					Name:  "STATUS",
					Width: 12,
					Getter: func(r dao.Resource) string {
						if sr, ok := r.(*SubscriptionResource); ok {
							if sr.IsPending() {
								return "Pending"
							}
							return "Confirmed"
						}
						return ""
					},
					Priority: 3,
				},
				{
					Name:  "OWNER",
					Width: 14,
					Getter: func(r dao.Resource) string {
						if sr, ok := r.(*SubscriptionResource); ok {
							return sr.Owner()
						}
						return ""
					},
					Priority: 4,
				},
			},
		},
	}
}

// RenderDetail renders detailed subscription information
func (r *SubscriptionRenderer) RenderDetail(resource dao.Resource) string {
	sr, ok := resource.(*SubscriptionResource)
	if !ok {
		return ""
	}

	d := render.NewDetailBuilder()

	status := "Confirmed"
	if sr.IsPending() {
		status = "Pending Confirmation"
	}

	d.Title("SNS Subscription", sr.Protocol()+" → "+sr.Endpoint())

	// Basic Info
	d.Section("Basic Information")
	d.Field("ARN", sr.ARN())
	d.Field("Status", status)
	d.Field("Owner", sr.Owner())

	// Topic
	d.Section("Topic")
	d.Field("Topic Name", sr.TopicName())
	d.Field("Topic ARN", sr.TopicARN())

	// Delivery
	d.Section("Delivery")
	d.Field("Protocol", sr.Protocol())
	d.Field("Endpoint", sr.Endpoint())

	// Attributes (if available)
	if sr.Attrs != nil {
		if rawDelivery, ok := sr.Attrs["RawMessageDelivery"]; ok {
			d.Field("Raw Message Delivery", rawDelivery)
		}
	}

	// Filter Policy (at bottom for readability)
	if sr.Attrs != nil {
		if filterPolicy, ok := sr.Attrs["FilterPolicy"]; ok && filterPolicy != "" {
			d.Section("Filter Policy")
			d.Line(prettyJSON(filterPolicy))
		}
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
func (r *SubscriptionRenderer) RenderSummary(resource dao.Resource) []render.SummaryField {
	sr, ok := resource.(*SubscriptionResource)
	if !ok {
		return nil
	}

	status := "Confirmed"
	if sr.IsPending() {
		status = "Pending"
	}

	fields := []render.SummaryField{
		{Label: "ARN", Value: sr.ARN()},
		{Label: "Status", Value: status},
		{Label: "Topic", Value: sr.TopicName()},
	}

	fields = append(fields, render.SummaryField{Label: "Protocol", Value: sr.Protocol()})
	fields = append(fields, render.SummaryField{Label: "Endpoint", Value: sr.Endpoint()})
	fields = append(fields, render.SummaryField{Label: "Owner", Value: sr.Owner()})

	return fields
}

// Navigations returns navigation shortcuts for SNS subscriptions
func (r *SubscriptionRenderer) Navigations(resource dao.Resource) []render.Navigation {
	sr, ok := resource.(*SubscriptionResource)
	if !ok {
		return nil
	}

	var navs []render.Navigation

	// Topic navigation
	if sr.TopicARN() != "" {
		navs = append(navs, render.Navigation{
			Key: "t", Label: "Topic", Service: "sns", Resource: "topics",
			FilterField: "TopicArn", FilterValue: sr.TopicARN(),
		})
	}

	// Lambda navigation if endpoint is Lambda
	if sr.Protocol() == "lambda" && sr.Endpoint() != "" {
		// Extract function name from ARN
		parts := strings.Split(sr.Endpoint(), ":")
		if len(parts) > 0 {
			funcName := parts[len(parts)-1]
			navs = append(navs, render.Navigation{
				Key: "l", Label: "Lambda", Service: "lambda", Resource: "functions",
				FilterField: "FunctionName", FilterValue: funcName,
			})
		}
	}

	// SQS navigation if endpoint is SQS
	if sr.Protocol() == "sqs" && sr.Endpoint() != "" {
		// Extract queue name from ARN
		parts := strings.Split(sr.Endpoint(), ":")
		if len(parts) > 0 {
			queueName := parts[len(parts)-1]
			navs = append(navs, render.Navigation{
				Key: "q", Label: "SQS Queue", Service: "sqs", Resource: "queues",
				FilterField: "QueueName", FilterValue: queueName,
			})
		}
	}

	return navs
}
