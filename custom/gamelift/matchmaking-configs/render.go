package matchmakingconfigs

import (
	"fmt"
	"strings"

	appaws "github.com/clawscli/claws/internal/aws"
	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/render"
)

// Ensure MatchmakingConfigRenderer implements render.Navigator
var _ render.Navigator = (*MatchmakingConfigRenderer)(nil)

// MatchmakingConfigRenderer renders GameLift matchmaking configurations.
type MatchmakingConfigRenderer struct {
	render.BaseRenderer
}

// NewMatchmakingConfigRenderer creates a new MatchmakingConfigRenderer.
func NewMatchmakingConfigRenderer() render.Renderer {
	return &MatchmakingConfigRenderer{
		BaseRenderer: render.BaseRenderer{
			Service:  "gamelift",
			Resource: "matchmaking-configs",
			Cols: []render.Column{
				{Name: "NAME", Width: 30, Getter: func(r dao.Resource) string { return r.GetName() }},
				{Name: "MODE", Width: 14, Getter: getMatchmakingMode},
				{Name: "BACKFILL", Width: 12, Getter: getMatchmakingBackfill},
				{Name: "ACCEPTANCE", Width: 12, Getter: getMatchmakingAcceptance},
				{Name: "RULE SET", Width: 24, Getter: getMatchmakingRuleSet},
				{Name: "TIMEOUT (s)", Width: 12, Getter: getMatchmakingTimeout, Priority: 2},
				{Name: "CREATED", Width: 20, Getter: getMatchmakingCreated, Priority: 2},
			},
		},
	}
}

func getMatchmakingMode(r dao.Resource) string {
	config, ok := r.(*MatchmakingConfigResource)
	if !ok {
		return ""
	}
	return config.FlexMatchMode()
}

func getMatchmakingBackfill(r dao.Resource) string {
	config, ok := r.(*MatchmakingConfigResource)
	if !ok {
		return ""
	}
	return config.BackfillMode()
}

func getMatchmakingAcceptance(r dao.Resource) string {
	config, ok := r.(*MatchmakingConfigResource)
	if !ok {
		return ""
	}
	if config.AcceptanceRequired() {
		return "Required"
	}
	return "Not required"
}

func getMatchmakingRuleSet(r dao.Resource) string {
	config, ok := r.(*MatchmakingConfigResource)
	if !ok {
		return ""
	}
	return config.RuleSetName()
}

func getMatchmakingTimeout(r dao.Resource) string {
	config, ok := r.(*MatchmakingConfigResource)
	if !ok {
		return ""
	}
	timeout := config.RequestTimeoutSeconds()
	if timeout == 0 {
		return "-"
	}
	return fmt.Sprintf("%d", timeout)
}

func getMatchmakingCreated(r dao.Resource) string {
	config, ok := r.(*MatchmakingConfigResource)
	if !ok {
		return ""
	}
	if t := config.CreationTime(); t != nil {
		return t.Format("2006-01-02 15:04")
	}
	return ""
}

// RenderDetail renders the detail view for a GameLift matchmaking configuration.
func (rr *MatchmakingConfigRenderer) RenderDetail(resource dao.Resource) string {
	config, ok := resource.(*MatchmakingConfigResource)
	if !ok {
		return ""
	}

	d := render.NewDetailBuilder()

	d.Title("GameLift Matchmaking Configuration", config.GetName())

	d.Section("Basic Information")
	d.Field("Name", config.GetName())
	d.Field("ARN", config.GetARN())
	if desc := config.Description(); desc != "" {
		d.Field("Description", desc)
	}

	d.Section("Configuration")
	d.Field("FlexMatch Mode", config.FlexMatchMode())
	d.Field("Backfill Mode", config.BackfillMode())
	if config.AcceptanceRequired() {
		d.Field("Acceptance Required", "Yes")
		if timeout := config.AcceptanceTimeoutSeconds(); timeout > 0 {
			d.Field("Acceptance Timeout", fmt.Sprintf("%d seconds", timeout))
		}
	} else {
		d.Field("Acceptance Required", "No")
	}
	if timeout := config.RequestTimeoutSeconds(); timeout > 0 {
		d.Field("Request Timeout", fmt.Sprintf("%d seconds", timeout))
	}
	if count := config.AdditionalPlayerCount(); count > 0 {
		d.Field("Additional Player Count", fmt.Sprintf("%d", count))
	}

	d.Section("Rule Set")
	d.Field("Rule Set Name", config.RuleSetName())
	if arn := config.RuleSetArn(); arn != "" {
		d.Field("Rule Set ARN", arn)
	}

	// Game Session Queues
	if queues := config.GameSessionQueueArns(); len(queues) > 0 {
		d.Section("Game Session Queues")
		for i, queueArn := range queues {
			d.Field(fmt.Sprintf("Queue %d", i+1), queueArn)
		}
	}

	// Game Properties
	if props := config.GameProperties(); len(props) > 0 {
		d.Section("Game Properties")
		for _, prop := range props {
			d.Field(appaws.Str(prop.Key), appaws.Str(prop.Value))
		}
	}

	// Notifications
	if target := config.NotificationTarget(); target != "" {
		d.Section("Notifications")
		d.Field("SNS Target", target)
	}
	if data := config.CustomEventData(); data != "" {
		d.Field("Custom Event Data", data)
	}

	d.Section("Timestamps")
	if t := config.CreationTime(); t != nil {
		d.Field("Created", t.Format("2006-01-02 15:04:05"))
	}

	return d.String()
}

// RenderSummary renders summary fields for a GameLift matchmaking configuration.
func (rr *MatchmakingConfigRenderer) RenderSummary(resource dao.Resource) []render.SummaryField {
	config, ok := resource.(*MatchmakingConfigResource)
	if !ok {
		return rr.BaseRenderer.RenderSummary(resource)
	}

	return []render.SummaryField{
		{Label: "Name", Value: config.GetName()},
		{Label: "ARN", Value: config.GetARN()},
		{Label: "Mode", Value: config.FlexMatchMode()},
		{Label: "Rule Set", Value: config.RuleSetName()},
	}
}

// Navigations returns available navigations from a GameLift matchmaking configuration.
func (rr *MatchmakingConfigRenderer) Navigations(resource dao.Resource) []render.Navigation {
	config, ok := resource.(*MatchmakingConfigResource)
	if !ok {
		return nil
	}

	var navs []render.Navigation

	// Navigate to game session queues
	if queues := config.GameSessionQueueArns(); len(queues) > 0 {
		queueName := appaws.ExtractResourceName(queues[0])
		navs = append(navs, render.Navigation{
			Key:         "q",
			Label:       fmt.Sprintf("Queue (%s)", queueName),
			Service:     "gamelift",
			Resource:    "game-session-queues",
			FilterField: "QueueName",
			FilterValue: strings.TrimPrefix(queueName, "gamesessionqueue/"),
		})
	}

	return navs
}
