package stacks

import (
	"strings"
	"time"

	appaws "github.com/clawscli/claws/internal/aws"
	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/render"
)

// Ensure StackRenderer implements render.Navigator
var _ render.Navigator = (*StackRenderer)(nil)

// StackRenderer renders CloudFormation stacks
type StackRenderer struct {
	render.BaseRenderer
}

// NewStackRenderer creates a new StackRenderer
func NewStackRenderer() render.Renderer {
	return &StackRenderer{
		BaseRenderer: render.BaseRenderer{
			Service:  "cloudformation",
			Resource: "stacks",
			Cols: []render.Column{
				{
					Name:  "NAME",
					Width: 35,
					Getter: func(r dao.Resource) string {
						return r.GetName()
					},
					Priority: 0,
				},
				{
					Name:  "STATUS",
					Width: 28,
					Getter: func(r dao.Resource) string {
						if sr, ok := r.(*StackResource); ok {
							return sr.Status()
						}
						return ""
					},
					Priority: 1,
				},
				{
					Name:  "DRIFT",
					Width: 12,
					Getter: func(r dao.Resource) string {
						if sr, ok := r.(*StackResource); ok {
							return sr.DriftStatus()
						}
						return ""
					},
					Priority: 2,
				},
				{
					Name:  "CREATED",
					Width: 10,
					Getter: func(r dao.Resource) string {
						if sr, ok := r.(*StackResource); ok {
							if sr.Item.CreationTime != nil {
								return render.FormatAge(*sr.Item.CreationTime)
							}
						}
						return ""
					},
					Priority: 3,
				},
				{
					Name:  "UPDATED",
					Width: 10,
					Getter: func(r dao.Resource) string {
						if sr, ok := r.(*StackResource); ok {
							if sr.Item.LastUpdatedTime != nil {
								return render.FormatAge(*sr.Item.LastUpdatedTime)
							}
						}
						return ""
					},
					Priority: 4,
				},
				render.TagsColumn(30, 5),
			},
		},
	}
}

// RenderDetail renders detailed stack information
func (r *StackRenderer) RenderDetail(resource dao.Resource) string {
	sr, ok := resource.(*StackResource)
	if !ok {
		return ""
	}

	d := render.NewDetailBuilder()
	styles := d.Styles()

	d.Title("CloudFormation Stack", sr.GetName())

	// Basic Info
	d.Section("Basic Information")
	d.Field("Stack Name", sr.GetName())
	d.Field("Stack ID", sr.GetID())
	d.FieldStyled("Status", sr.Status(), cfnStateColorer(sr.Status()))
	if sr.Item.StackStatusReason != nil && *sr.Item.StackStatusReason != "" {
		d.Field("Status Reason", *sr.Item.StackStatusReason)
	}
	if sr.Item.Description != nil && *sr.Item.Description != "" {
		d.Field("Description", *sr.Item.Description)
	}

	// Timestamps
	d.Section("Timestamps")
	if sr.Item.CreationTime != nil {
		d.Field("Created", sr.Item.CreationTime.Format(time.RFC3339))
		d.Field("Age", render.FormatAge(*sr.Item.CreationTime))
	}
	if sr.Item.LastUpdatedTime != nil {
		d.Field("Last Updated", sr.Item.LastUpdatedTime.Format(time.RFC3339))
	}

	// Drift Information
	if sr.Item.DriftInformation != nil {
		d.Section("Drift Information")
		d.FieldStyled("Drift Status", string(sr.Item.DriftInformation.StackDriftStatus),
			driftColorer(string(sr.Item.DriftInformation.StackDriftStatus)))
		if sr.Item.DriftInformation.LastCheckTimestamp != nil {
			d.Field("Last Check", sr.Item.DriftInformation.LastCheckTimestamp.Format(time.RFC3339))
		}
	}

	// Configuration
	d.Section("Configuration")
	if sr.Item.EnableTerminationProtection != nil {
		if *sr.Item.EnableTerminationProtection {
			d.FieldStyled("Termination Protection", "Enabled", styles.Success)
		} else {
			d.Field("Termination Protection", "Disabled")
		}
	}
	if sr.Item.DisableRollback != nil && *sr.Item.DisableRollback {
		d.Field("Rollback", "Disabled")
	}
	if sr.Item.RoleARN != nil {
		d.Field("IAM Role", *sr.Item.RoleARN)
	}

	// Capabilities
	if len(sr.Item.Capabilities) > 0 {
		caps := make([]string, len(sr.Item.Capabilities))
		for i, cap := range sr.Item.Capabilities {
			caps[i] = string(cap)
		}
		d.Field("Capabilities", strings.Join(caps, ", "))
	}

	// Outputs
	if len(sr.Item.Outputs) > 0 {
		d.Section("Outputs")
		for _, output := range sr.Item.Outputs {
			key := appaws.Str(output.OutputKey)
			val := appaws.Str(output.OutputValue)
			d.Line("  " + styles.Label.Render(key+":") + " " + styles.Value.Render(val))
			if output.Description != nil && *output.Description != "" {
				d.Line("    " + styles.Dim.Render(*output.Description))
			}
		}
	}

	// Parameters
	if len(sr.Item.Parameters) > 0 {
		d.Section("Parameters")
		for _, param := range sr.Item.Parameters {
			key := appaws.Str(param.ParameterKey)
			val := appaws.Str(param.ParameterValue)
			d.Tag(key, val)
		}
	}

	// Tags
	d.Tags(appaws.TagsToMap(sr.Item.Tags))

	// Nested Stack Info
	if sr.Item.ParentId != nil || sr.Item.RootId != nil {
		d.Section("Nested Stack Info")
		d.FieldIf("Parent Stack", sr.Item.ParentId)
		d.FieldIf("Root Stack", sr.Item.RootId)
	}

	return d.String()
}

// RenderSummary returns summary fields for the header panel
func (r *StackRenderer) RenderSummary(resource dao.Resource) []render.SummaryField {
	sr, ok := resource.(*StackResource)
	if !ok {
		return nil
	}

	fields := []render.SummaryField{
		{Label: "Name", Value: sr.GetName()},
		{Label: "Status", Value: sr.Status(), Style: cfnStateColorer(sr.Status())},
	}

	if sr.DriftStatus() != "" {
		fields = append(fields, render.SummaryField{
			Label: "Drift",
			Value: sr.DriftStatus(),
			Style: driftColorer(sr.DriftStatus()),
		})
	}

	if sr.Item.Description != nil && *sr.Item.Description != "" {
		desc := *sr.Item.Description
		if len(desc) > 60 {
			desc = desc[:57] + "..."
		}
		fields = append(fields, render.SummaryField{Label: "Description", Value: desc})
	}

	if sr.Item.CreationTime != nil {
		fields = append(fields, render.SummaryField{
			Label: "Created",
			Value: sr.Item.CreationTime.Format("2006-01-02 15:04") + " (" + render.FormatAge(*sr.Item.CreationTime) + ")",
		})
	}

	if sr.Item.LastUpdatedTime != nil {
		fields = append(fields, render.SummaryField{
			Label: "Updated",
			Value: sr.Item.LastUpdatedTime.Format("2006-01-02 15:04") + " (" + render.FormatAge(*sr.Item.LastUpdatedTime) + ")",
		})
	}

	if sr.Item.EnableTerminationProtection != nil && *sr.Item.EnableTerminationProtection {
		fields = append(fields, render.SummaryField{Label: "Protection", Value: "Enabled"})
	}

	return fields
}

// cfnStateColorer returns a style for CloudFormation stack status
func cfnStateColorer(status string) render.Style {
	switch {
	case strings.HasSuffix(status, "_COMPLETE") && !strings.Contains(status, "ROLLBACK") && !strings.Contains(status, "DELETE"):
		return render.SuccessStyle()
	case strings.Contains(status, "IN_PROGRESS"):
		return render.WarningStyle()
	case strings.Contains(status, "FAILED") || strings.Contains(status, "ROLLBACK"):
		return render.DangerStyle()
	case strings.Contains(status, "DELETE_COMPLETE"):
		return render.DimStyle()
	default:
		return render.DefaultStyle()
	}
}

// driftColorer returns a style for drift status
func driftColorer(status string) render.Style {
	switch status {
	case "IN_SYNC":
		return render.SuccessStyle()
	case "DRIFTED":
		return render.DangerStyle()
	case "NOT_CHECKED":
		return render.DimStyle()
	default:
		return render.DefaultStyle()
	}
}

// Navigations returns navigation shortcuts for CloudFormation stacks
func (r *StackRenderer) Navigations(resource dao.Resource) []render.Navigation {
	sr, ok := resource.(*StackResource)
	if !ok {
		return nil
	}

	stackName := sr.GetName()

	return []render.Navigation{
		{
			Key: "e", Label: "Events", Service: "cloudformation", Resource: "events",
			FilterField: "StackName", FilterValue: stackName,
			AutoReload: true, // Events auto-refresh every 3s
		},
		{
			Key: "r", Label: "Resources", Service: "cloudformation", Resource: "resources",
			FilterField: "StackName", FilterValue: stackName,
		},
		{
			Key: "o", Label: "Outputs", Service: "cloudformation", Resource: "outputs",
			FilterField: "StackName", FilterValue: stackName,
		},
	}
}
