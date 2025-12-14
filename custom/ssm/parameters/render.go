package parameters

import (
	"fmt"
	"time"

	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/render"
)

// ParameterRenderer renders SSM Parameters
// Ensure ParameterRenderer implements render.Navigator
var _ render.Navigator = (*ParameterRenderer)(nil)

type ParameterRenderer struct {
	render.BaseRenderer
}

// NewParameterRenderer creates a new ParameterRenderer
func NewParameterRenderer() render.Renderer {
	return &ParameterRenderer{
		BaseRenderer: render.BaseRenderer{
			Service:  "ssm",
			Resource: "parameters",
			Cols: []render.Column{
				{Name: "NAME", Width: 45, Getter: func(r dao.Resource) string { return r.GetName() }},
				{Name: "TYPE", Width: 14, Getter: getType},
				{Name: "TIER", Width: 10, Getter: getTier},
				{Name: "VER", Width: 5, Getter: getVersion},
				{Name: "MODIFIED", Width: 12, Getter: getModified},
			},
		},
	}
}

func getType(r dao.Resource) string {
	if param, ok := r.(*ParameterResource); ok {
		t := param.Type()
		switch t {
		case "SecureString":
			return "Secure"
		default:
			return t
		}
	}
	return ""
}

func getTier(r dao.Resource) string {
	if param, ok := r.(*ParameterResource); ok {
		return param.Tier()
	}
	return ""
}

func getVersion(r dao.Resource) string {
	if param, ok := r.(*ParameterResource); ok {
		return fmt.Sprintf("%d", param.Version())
	}
	return ""
}

func getModified(r dao.Resource) string {
	if param, ok := r.(*ParameterResource); ok {
		if param.Item.LastModifiedDate != nil {
			return render.FormatAge(*param.Item.LastModifiedDate)
		}
	}
	return "-"
}

// RenderDetail renders detailed parameter information
func (r *ParameterRenderer) RenderDetail(resource dao.Resource) string {
	param, ok := resource.(*ParameterResource)
	if !ok {
		return ""
	}

	d := render.NewDetailBuilder()

	d.Title("SSM Parameter", param.GetName())

	// Basic Info
	d.Section("Basic Information")
	d.Field("Name", param.GetName())
	d.Field("ARN", param.GetARN())
	d.Field("Type", param.Type())
	d.Field("Tier", param.Tier())
	d.Field("Data Type", param.DataType())

	if desc := param.Description(); desc != "" {
		d.Field("Description", desc)
	}

	// Version Info
	d.Section("Version Information")
	d.Field("Version", fmt.Sprintf("%d", param.Version()))

	// Timestamps
	d.Section("Timestamps")
	if modified := param.LastModifiedDate(); modified != "" {
		d.Field("Last Modified", modified)
	}
	if param.Item.LastModifiedDate != nil {
		d.Field("Age", time.Since(*param.Item.LastModifiedDate).Truncate(time.Second).String())
	}
	if user := param.LastModifiedUser(); user != "" {
		d.Field("Modified By", user)
	}

	return d.String()
}

// RenderSummary returns summary fields for the header panel
func (r *ParameterRenderer) RenderSummary(resource dao.Resource) []render.SummaryField {
	param, ok := resource.(*ParameterResource)
	if !ok {
		return r.BaseRenderer.RenderSummary(resource)
	}

	fields := []render.SummaryField{
		{Label: "Name", Value: param.GetName()},
		{Label: "ARN", Value: param.GetARN()},
		{Label: "Type", Value: param.Type()},
		{Label: "Tier", Value: param.Tier()},
		{Label: "Version", Value: fmt.Sprintf("%d", param.Version())},
	}

	if desc := param.Description(); desc != "" {
		fields = append(fields, render.SummaryField{Label: "Description", Value: desc})
	}

	if modified := param.LastModifiedDate(); modified != "" {
		fields = append(fields, render.SummaryField{Label: "Last Modified", Value: modified})
	}

	return fields
}

// Navigations returns navigation shortcuts
func (r *ParameterRenderer) Navigations(resource dao.Resource) []render.Navigation {
	// No navigations for now
	return nil
}
