package rules

import (
	"encoding/json"
	"fmt"
	"strings"

	appaws "github.com/clawscli/claws/internal/aws"
	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/render"
)

// Ensure RuleRenderer implements render.Navigator
var _ render.Navigator = (*RuleRenderer)(nil)

// RuleRenderer renders EventBridge rules with custom columns
type RuleRenderer struct {
	render.BaseRenderer
}

// NewRuleRenderer creates a new RuleRenderer
func NewRuleRenderer() render.Renderer {
	return &RuleRenderer{
		BaseRenderer: render.BaseRenderer{
			Service:  "eventbridge",
			Resource: "rules",
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
					Name:  "STATE",
					Width: 10,
					Getter: func(r dao.Resource) string {
						if rr, ok := r.(*RuleResource); ok {
							return rr.State()
						}
						return ""
					},
					Priority: 1,
				},
				{
					Name:  "TYPE",
					Width: 10,
					Getter: func(r dao.Resource) string {
						if rr, ok := r.(*RuleResource); ok {
							return rr.RuleType()
						}
						return ""
					},
					Priority: 2,
				},
				{
					Name:  "BUS",
					Width: 20,
					Getter: func(r dao.Resource) string {
						if rr, ok := r.(*RuleResource); ok {
							return rr.EventBusName()
						}
						return ""
					},
					Priority: 3,
				},
				{
					Name:  "SCHEDULE/PATTERN",
					Width: 40,
					Getter: func(r dao.Resource) string {
						if rr, ok := r.(*RuleResource); ok {
							if rr.ScheduleExpression() != "" {
								return rr.ScheduleExpression()
							}
							// Return truncated event pattern
							pattern := rr.EventPattern()
							if len(pattern) > 37 {
								return pattern[:37] + "..."
							}
							return pattern
						}
						return ""
					},
					Priority: 4,
				},
			},
		},
	}
}

// RenderDetail renders detailed rule information
func (r *RuleRenderer) RenderDetail(resource dao.Resource) string {
	rr, ok := resource.(*RuleResource)
	if !ok {
		return ""
	}

	d := render.NewDetailBuilder()

	d.Title("EventBridge Rule", rr.GetName())

	// Basic Info
	d.Section("Basic Information")
	d.Field("Name", rr.GetName())
	d.Field("ARN", rr.ARN())
	d.FieldStyled("State", rr.State(), render.StateColorer()(rr.State()))
	d.Field("Event Bus", rr.EventBusName())
	if rr.Description() != "" {
		d.Field("Description", rr.Description())
	}

	// Trigger
	d.Section("Trigger")
	d.Field("Type", rr.RuleType())
	if rr.ScheduleExpression() != "" {
		d.Field("Schedule Expression", rr.ScheduleExpression())
	}

	// IAM Role
	if rr.RoleArn != "" {
		d.Section("IAM")
		d.Field("Role ARN", rr.RoleArn)
	}

	// Targets
	d.Section("Targets")
	if len(rr.Targets) == 0 {
		d.Field("Targets", "None configured")
	} else {
		d.Field("Target Count", fmt.Sprintf("%d", len(rr.Targets)))
		for i, target := range rr.Targets {
			targetId := appaws.Str(target.Id)
			targetArn := appaws.Str(target.Arn)

			// Extract service and resource type from ARN
			targetType := "Unknown"
			if parts := strings.Split(targetArn, ":"); len(parts) >= 6 {
				targetType = parts[2] // e.g., lambda, sqs, sns
			}

			d.Field(fmt.Sprintf("  Target %d", i+1), fmt.Sprintf("[%s] %s", targetId, targetType))
			d.Field("    ARN", targetArn)

			if target.RoleArn != nil {
				d.Field("    Role", *target.RoleArn)
			}
			if target.Input != nil {
				d.Field("    Input", truncate(*target.Input, 80))
			}
			if target.InputPath != nil {
				d.Field("    Input Path", *target.InputPath)
			}
		}
	}

	// Event Pattern
	if rr.EventPattern() != "" {
		d.Section("Event Pattern")
		// Pretty print JSON
		var prettyJSON map[string]any
		if err := json.Unmarshal([]byte(rr.EventPattern()), &prettyJSON); err == nil {
			if formatted, err := json.MarshalIndent(prettyJSON, "", "  "); err == nil {
				for _, line := range strings.Split(string(formatted), "\n") {
					d.Line(line)
				}
			} else {
				d.Line(rr.EventPattern())
			}
		} else {
			d.Line(rr.EventPattern())
		}
	}

	return d.String()
}

func truncate(s string, maxLen int) string {
	if len(s) > maxLen {
		return s[:maxLen] + "..."
	}
	return s
}

// RenderSummary returns summary fields for the header panel
func (r *RuleRenderer) RenderSummary(resource dao.Resource) []render.SummaryField {
	rr, ok := resource.(*RuleResource)
	if !ok {
		return nil
	}

	stateStyle := render.StateColorer()(rr.State())

	fields := []render.SummaryField{
		{Label: "Name", Value: rr.GetName()},
		{Label: "State", Value: rr.State(), Style: stateStyle},
		{Label: "Type", Value: rr.RuleType()},
		{Label: "Event Bus", Value: rr.EventBusName()},
	}

	if rr.ScheduleExpression() != "" {
		fields = append(fields, render.SummaryField{Label: "Schedule", Value: rr.ScheduleExpression()})
	}

	if rr.Description() != "" {
		fields = append(fields, render.SummaryField{Label: "Description", Value: rr.Description()})
	}

	return fields
}

// Navigations returns navigation shortcuts for rules
func (r *RuleRenderer) Navigations(resource dao.Resource) []render.Navigation {
	rr, ok := resource.(*RuleResource)
	if !ok {
		return nil
	}

	var navs []render.Navigation

	// Event Bus navigation
	navs = append(navs, render.Navigation{
		Key: "b", Label: "Event Bus", Service: "eventbridge", Resource: "buses",
		FilterField: "Name", FilterValue: rr.EventBusName(),
	})

	return navs
}
