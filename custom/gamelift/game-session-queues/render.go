package gamesessionqueues

import (
	"fmt"
	"strings"

	appaws "github.com/clawscli/claws/internal/aws"
	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/render"
)

// QueueRenderer renders GameLift game session queues.
type QueueRenderer struct {
	render.BaseRenderer
}

// NewQueueRenderer creates a new QueueRenderer.
func NewQueueRenderer() render.Renderer {
	return &QueueRenderer{
		BaseRenderer: render.BaseRenderer{
			Service:  "gamelift",
			Resource: "game-session-queues",
			Cols: []render.Column{
				{Name: "NAME", Width: 30, Getter: func(r dao.Resource) string { return r.GetName() }},
				{Name: "TIMEOUT (s)", Width: 12, Getter: getQueueTimeout},
				{Name: "DESTINATIONS", Width: 14, Getter: getQueueDestinationCount},
				{Name: "NOTIFICATION", Width: 40, Getter: getQueueNotification, Priority: 3},
			},
		},
	}
}

func getQueueTimeout(r dao.Resource) string {
	queue, ok := r.(*QueueResource)
	if !ok {
		return ""
	}
	timeout := queue.TimeoutInSeconds()
	if timeout == 0 {
		return "-"
	}
	return fmt.Sprintf("%d", timeout)
}

func getQueueDestinationCount(r dao.Resource) string {
	queue, ok := r.(*QueueResource)
	if !ok {
		return ""
	}
	return fmt.Sprintf("%d", queue.DestinationCount())
}

func getQueueNotification(r dao.Resource) string {
	queue, ok := r.(*QueueResource)
	if !ok {
		return ""
	}
	return queue.NotificationTarget()
}

// RenderDetail renders the detail view for a GameLift game session queue.
func (rr *QueueRenderer) RenderDetail(resource dao.Resource) string {
	queue, ok := resource.(*QueueResource)
	if !ok {
		return ""
	}

	d := render.NewDetailBuilder()

	d.Title("GameLift Game Session Queue", queue.GetName())

	d.Section("Basic Information")
	d.Field("Name", queue.GetName())
	d.Field("ARN", queue.GetARN())
	d.Field("Timeout", fmt.Sprintf("%d seconds", queue.TimeoutInSeconds()))

	// Destinations
	if dests := queue.Destinations(); len(dests) > 0 {
		d.Section("Destinations")
		for i, dest := range dests {
			d.Field(fmt.Sprintf("Destination %d", i+1), appaws.Str(dest.DestinationArn))
		}
	}

	// Player Latency Policies
	if policies := queue.PlayerLatencyPolicies(); len(policies) > 0 {
		d.Section("Player Latency Policies")
		for i, policy := range policies {
			d.Field(fmt.Sprintf("Policy %d", i+1),
				fmt.Sprintf("Max latency: %dms, eval period: %ds",
					appaws.Int32(policy.MaximumIndividualPlayerLatencyMilliseconds),
					appaws.Int32(policy.PolicyDurationSeconds)))
		}
	}

	// Filter Configuration
	if fc := queue.FilterConfiguration(); fc != nil && len(fc.AllowedLocations) > 0 {
		d.Section("Filter Configuration")
		d.Field("Allowed Locations", strings.Join(fc.AllowedLocations, ", "))
	}

	// Notification
	if target := queue.NotificationTarget(); target != "" {
		d.Section("Notifications")
		d.Field("SNS Target", target)
	}

	if data := queue.CustomEventData(); data != "" {
		d.Field("Custom Event Data", data)
	}

	return d.String()
}

// RenderSummary renders summary fields for a GameLift game session queue.
func (rr *QueueRenderer) RenderSummary(resource dao.Resource) []render.SummaryField {
	queue, ok := resource.(*QueueResource)
	if !ok {
		return rr.BaseRenderer.RenderSummary(resource)
	}

	return []render.SummaryField{
		{Label: "Name", Value: queue.GetName()},
		{Label: "ARN", Value: queue.GetARN()},
		{Label: "Timeout", Value: fmt.Sprintf("%d seconds", queue.TimeoutInSeconds())},
		{Label: "Destinations", Value: fmt.Sprintf("%d", queue.DestinationCount())},
	}
}
