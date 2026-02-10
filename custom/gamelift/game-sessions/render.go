package gamesessions

import (
	"fmt"

	appaws "github.com/clawscli/claws/internal/aws"
	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/render"
)

// Ensure GameSessionRenderer implements render.Navigator
var _ render.Navigator = (*GameSessionRenderer)(nil)

// GameSessionRenderer renders GameLift game sessions.
type GameSessionRenderer struct {
	render.BaseRenderer
}

// NewGameSessionRenderer creates a new GameSessionRenderer.
func NewGameSessionRenderer() render.Renderer {
	return &GameSessionRenderer{
		BaseRenderer: render.BaseRenderer{
			Service:  "gamelift",
			Resource: "game-sessions",
			Cols: []render.Column{
				{Name: "SESSION NAME", Width: 28, Getter: func(r dao.Resource) string { return r.GetName() }},
				{Name: "STATUS", Width: 14, Getter: getSessionStatus},
				{Name: "PLAYERS", Width: 10, Getter: getSessionPlayers},
				{Name: "IP:PORT", Width: 22, Getter: getSessionEndpoint},
				{Name: "LOCATION", Width: 16, Getter: getSessionLocation, Priority: 2},
				{Name: "POLICY", Width: 14, Getter: getSessionPolicy, Priority: 3},
				{Name: "CREATED", Width: 20, Getter: getSessionCreated, Priority: 2},
			},
		},
	}
}

func getSessionStatus(r dao.Resource) string {
	session, ok := r.(*GameSessionResource)
	if !ok {
		return ""
	}
	return session.Status()
}

func getSessionPlayers(r dao.Resource) string {
	session, ok := r.(*GameSessionResource)
	if !ok {
		return ""
	}
	return fmt.Sprintf("%d/%d", session.CurrentPlayerSessionCount(), session.MaximumPlayerSessionCount())
}

func getSessionEndpoint(r dao.Resource) string {
	session, ok := r.(*GameSessionResource)
	if !ok {
		return ""
	}
	ip := session.IpAddress()
	port := session.Port()
	if ip == "" {
		return ""
	}
	if port > 0 {
		return fmt.Sprintf("%s:%d", ip, port)
	}
	return ip
}

func getSessionLocation(r dao.Resource) string {
	session, ok := r.(*GameSessionResource)
	if !ok {
		return ""
	}
	return session.Location()
}

func getSessionPolicy(r dao.Resource) string {
	session, ok := r.(*GameSessionResource)
	if !ok {
		return ""
	}
	return session.PlayerSessionCreationPolicy()
}

func getSessionCreated(r dao.Resource) string {
	session, ok := r.(*GameSessionResource)
	if !ok {
		return ""
	}
	if t := session.CreationTime(); t != nil {
		return t.Format("2006-01-02 15:04")
	}
	return ""
}

// RenderDetail renders the detail view for a GameLift game session.
func (rr *GameSessionRenderer) RenderDetail(resource dao.Resource) string {
	session, ok := resource.(*GameSessionResource)
	if !ok {
		return ""
	}

	d := render.NewDetailBuilder()

	d.Title("GameLift Game Session", session.GetName())

	d.Section("Basic Information")
	d.Field("Name", session.GetName())
	d.Field("Game Session ID", session.GetID())
	d.Field("Status", session.Status())
	if reason := session.StatusReason(); reason != "" {
		d.Field("Status Reason", reason)
	}

	d.Section("Fleet")
	d.Field("Fleet ID", session.FleetId())
	if arn := session.FleetArn(); arn != "" {
		d.Field("Fleet ARN", arn)
	}
	if loc := session.Location(); loc != "" {
		d.Field("Location", loc)
	}

	d.Section("Connection")
	if ip := session.IpAddress(); ip != "" {
		d.Field("IP Address", ip)
	}
	if dns := session.DnsName(); dns != "" {
		d.Field("DNS Name", dns)
	}
	if port := session.Port(); port > 0 {
		d.Field("Port", fmt.Sprintf("%d", port))
	}

	d.Section("Players")
	d.Field("Current Players", fmt.Sprintf("%d", session.CurrentPlayerSessionCount()))
	d.Field("Max Players", fmt.Sprintf("%d", session.MaximumPlayerSessionCount()))
	d.Field("Creation Policy", session.PlayerSessionCreationPolicy())
	if creator := session.CreatorId(); creator != "" {
		d.Field("Creator ID", creator)
	}

	// Game Properties
	if props := session.GameProperties(); len(props) > 0 {
		d.Section("Game Properties")
		for _, prop := range props {
			d.Field(appaws.Str(prop.Key), appaws.Str(prop.Value))
		}
	}

	if data := session.GameSessionData(); data != "" {
		d.Section("Game Session Data")
		d.Field("Data", data)
	}

	d.Section("Timestamps")
	if t := session.CreationTime(); t != nil {
		d.Field("Created", t.Format("2006-01-02 15:04:05"))
	}
	if t := session.TerminationTime(); t != nil {
		d.Field("Terminated", t.Format("2006-01-02 15:04:05"))
	}

	return d.String()
}

// RenderSummary renders summary fields for a GameLift game session.
func (rr *GameSessionRenderer) RenderSummary(resource dao.Resource) []render.SummaryField {
	session, ok := resource.(*GameSessionResource)
	if !ok {
		return rr.BaseRenderer.RenderSummary(resource)
	}

	fields := []render.SummaryField{
		{Label: "Name", Value: session.GetName()},
		{Label: "Status", Value: session.Status()},
		{Label: "Players", Value: fmt.Sprintf("%d/%d", session.CurrentPlayerSessionCount(), session.MaximumPlayerSessionCount())},
		{Label: "Fleet ID", Value: session.FleetId()},
	}

	if ip := session.IpAddress(); ip != "" {
		port := session.Port()
		if port > 0 {
			fields = append(fields, render.SummaryField{Label: "Endpoint", Value: fmt.Sprintf("%s:%d", ip, port)})
		}
	}

	return fields
}

// Navigations returns available navigations from a GameLift game session.
func (rr *GameSessionRenderer) Navigations(resource dao.Resource) []render.Navigation {
	session, ok := resource.(*GameSessionResource)
	if !ok {
		return nil
	}

	navs := []render.Navigation{
		{
			Key:         "f",
			Label:       fmt.Sprintf("Fleet (%s)", session.FleetId()),
			Service:     "gamelift",
			Resource:    "fleets",
			FilterField: "FleetId",
			FilterValue: session.FleetId(),
		},
	}

	return navs
}
