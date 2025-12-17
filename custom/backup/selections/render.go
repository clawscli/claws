package selections

import (
	"fmt"
	"strings"

	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/render"
)

// Ensure SelectionRenderer implements render.Navigator
var _ render.Navigator = (*SelectionRenderer)(nil)

// SelectionRenderer renders AWS Backup selections
type SelectionRenderer struct {
	render.BaseRenderer
}

// NewSelectionRenderer creates a new SelectionRenderer
func NewSelectionRenderer() *SelectionRenderer {
	return &SelectionRenderer{
		BaseRenderer: render.BaseRenderer{
			Service:  "backup",
			Resource: "selections",
			Cols: []render.Column{
				{Name: "NAME", Width: 30, Getter: getName},
				{Name: "SELECTION ID", Width: 40, Getter: getSelectionId},
				{Name: "CREATED", Width: 20, Getter: getCreated},
			},
		},
	}
}

func getName(r dao.Resource) string {
	if s, ok := r.(*SelectionResource); ok {
		return s.SelectionName()
	}
	return ""
}

func getSelectionId(r dao.Resource) string {
	if s, ok := r.(*SelectionResource); ok {
		return s.SelectionId()
	}
	return ""
}

func getCreated(r dao.Resource) string {
	if s, ok := r.(*SelectionResource); ok {
		return s.CreationDate()
	}
	return "-"
}

// RenderDetail renders detailed selection information
func (r *SelectionRenderer) RenderDetail(resource dao.Resource) string {
	sel, ok := resource.(*SelectionResource)
	if !ok {
		return ""
	}

	d := render.NewDetailBuilder()

	d.Title("Backup Selection", sel.SelectionName())

	// Basic Info
	d.Section("Basic Information")
	d.Field("Name", sel.SelectionName())
	d.Field("Selection ID", sel.SelectionId())
	d.Field("Backup Plan ID", sel.BackupPlanId)
	if created := sel.CreationDate(); created != "" {
		d.Field("Created", created)
	}

	// IAM
	if roleArn := sel.IamRoleArn(); roleArn != "" {
		d.Section("IAM")
		d.Field("Role ARN", roleArn)
	}

	// Resources
	if resources := sel.Resources(); len(resources) > 0 {
		d.Section("Resources (Included)")
		for i, res := range resources {
			d.Field(fmt.Sprintf("Resource %d", i+1), res)
		}
	}

	// Not Resources
	if notResources := sel.NotResources(); len(notResources) > 0 {
		d.Section("Resources (Excluded)")
		for i, res := range notResources {
			d.Field(fmt.Sprintf("Excluded %d", i+1), res)
		}
	}

	// Tag Conditions
	if tags := sel.ListOfTags(); len(tags) > 0 {
		d.Section("Tag Conditions")
		for i, tag := range tags {
			condType := string(tag.ConditionType)
			key := deref(tag.ConditionKey)
			value := deref(tag.ConditionValue)
			d.Field(fmt.Sprintf("Condition %d", i+1), fmt.Sprintf("%s: %s %s %s", condType, key, condType, value))
		}
	}

	// Advanced Conditions
	if conditions := sel.Conditions(); conditions != nil {
		d.Section("Advanced Conditions")
		if len(conditions.StringEquals) > 0 {
			for _, c := range conditions.StringEquals {
				d.Field("StringEquals", fmt.Sprintf("%s = %s", deref(c.ConditionKey), deref(c.ConditionValue)))
			}
		}
		if len(conditions.StringNotEquals) > 0 {
			for _, c := range conditions.StringNotEquals {
				d.Field("StringNotEquals", fmt.Sprintf("%s != %s", deref(c.ConditionKey), deref(c.ConditionValue)))
			}
		}
		if len(conditions.StringLike) > 0 {
			for _, c := range conditions.StringLike {
				d.Field("StringLike", fmt.Sprintf("%s ~ %s", deref(c.ConditionKey), deref(c.ConditionValue)))
			}
		}
		if len(conditions.StringNotLike) > 0 {
			for _, c := range conditions.StringNotLike {
				d.Field("StringNotLike", fmt.Sprintf("%s !~ %s", deref(c.ConditionKey), deref(c.ConditionValue)))
			}
		}
	}

	return d.String()
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// RenderSummary returns summary fields for the header panel
func (r *SelectionRenderer) RenderSummary(resource dao.Resource) []render.SummaryField {
	sel, ok := resource.(*SelectionResource)
	if !ok {
		return r.BaseRenderer.RenderSummary(resource)
	}

	fields := []render.SummaryField{
		{Label: "Name", Value: sel.SelectionName()},
		{Label: "Selection ID", Value: sel.SelectionId()},
		{Label: "Plan ID", Value: sel.BackupPlanId},
	}

	if resources := sel.Resources(); len(resources) > 0 {
		fields = append(fields, render.SummaryField{
			Label: "Resources",
			Value: fmt.Sprintf("%d included", len(resources)),
		})
	}

	if roleArn := sel.IamRoleArn(); roleArn != "" {
		// Extract role name from ARN
		parts := strings.Split(roleArn, "/")
		roleName := roleArn
		if len(parts) > 1 {
			roleName = parts[len(parts)-1]
		}
		fields = append(fields, render.SummaryField{Label: "IAM Role", Value: roleName})
	}

	if created := sel.CreationDate(); created != "" {
		fields = append(fields, render.SummaryField{Label: "Created", Value: created})
	}

	return fields
}

// Navigations returns navigation shortcuts
func (r *SelectionRenderer) Navigations(resource dao.Resource) []render.Navigation {
	sel, ok := resource.(*SelectionResource)
	if !ok {
		return nil
	}

	var navs []render.Navigation

	// Navigate back to plan
	if sel.BackupPlanId != "" {
		navs = append(navs, render.Navigation{
			Key: "p", Label: "Plan", Service: "backup", Resource: "plans",
			FilterField: "PlanId", FilterValue: sel.BackupPlanId,
		})
	}

	return navs
}
