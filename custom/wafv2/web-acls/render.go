package webacls

import (
	"fmt"

	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/render"
)

// WebACLRenderer renders WAFv2 Web ACLs
// Ensure WebACLRenderer implements render.Navigator
var _ render.Navigator = (*WebACLRenderer)(nil)

type WebACLRenderer struct {
	render.BaseRenderer
}

// NewWebACLRenderer creates a new WebACLRenderer
func NewWebACLRenderer() *WebACLRenderer {
	return &WebACLRenderer{
		BaseRenderer: render.BaseRenderer{
			Service:  "wafv2",
			Resource: "web-acls",
			Cols: []render.Column{
				{Name: "NAME", Width: 30, Getter: getName},
				{Name: "ID", Width: 40, Getter: getWebACLId},
				{Name: "SCOPE", Width: 12, Getter: getScope},
				{Name: "RULES", Width: 8, Getter: getRuleCount},
				{Name: "DEFAULT", Width: 10, Getter: getDefaultAction},
			},
		},
	}
}

func getName(r dao.Resource) string {
	if w, ok := r.(*WebACLResource); ok {
		return w.WebACLName()
	}
	return ""
}

func getWebACLId(r dao.Resource) string {
	if w, ok := r.(*WebACLResource); ok {
		return w.WebACLId()
	}
	return ""
}

func getScope(r dao.Resource) string {
	if w, ok := r.(*WebACLResource); ok {
		return w.ScopeString()
	}
	return ""
}

func getRuleCount(r dao.Resource) string {
	if w, ok := r.(*WebACLResource); ok {
		if count := w.RuleCount(); count > 0 {
			return fmt.Sprintf("%d", count)
		}
	}
	return "-"
}

func getDefaultAction(r dao.Resource) string {
	if w, ok := r.(*WebACLResource); ok {
		return w.DefaultAction()
	}
	return ""
}

// RenderDetail renders detailed web ACL information
func (r *WebACLRenderer) RenderDetail(resource dao.Resource) string {
	webacl, ok := resource.(*WebACLResource)
	if !ok {
		return ""
	}

	d := render.NewDetailBuilder()

	d.Title("WAFv2 Web ACL", webacl.WebACLName())

	// Basic Info
	d.Section("Basic Information")
	d.Field("Name", webacl.WebACLName())
	d.Field("ID", webacl.WebACLId())
	if arn := webacl.GetARN(); arn != "" {
		d.Field("ARN", arn)
	}
	d.Field("Scope", webacl.ScopeString())
	if desc := webacl.Description(); desc != "" {
		d.Field("Description", desc)
	}

	// Configuration
	d.Section("Configuration")
	d.Field("Default Action", webacl.DefaultAction())
	d.Field("Rules Count", fmt.Sprintf("%d", webacl.RuleCount()))
	if capacity := webacl.Capacity(); capacity > 0 {
		d.Field("Capacity (WCU)", fmt.Sprintf("%d", capacity))
	}

	// Rules
	if rules := webacl.Rules(); len(rules) > 0 {
		d.Section("Rules")
		for _, rule := range rules {
			priority := ""
			if rule.Priority != 0 {
				priority = fmt.Sprintf(" (Priority: %d)", rule.Priority)
			}
			action := "Custom"
			if rule.Action != nil {
				if rule.Action.Allow != nil {
					action = "ALLOW"
				} else if rule.Action.Block != nil {
					action = "BLOCK"
				} else if rule.Action.Count != nil {
					action = "COUNT"
				} else if rule.Action.Captcha != nil {
					action = "CAPTCHA"
				} else if rule.Action.Challenge != nil {
					action = "CHALLENGE"
				}
			}
			if rule.OverrideAction != nil {
				action = "Override"
			}
			d.Field(deref(rule.Name)+priority, action)
		}
	}

	// Management
	d.Section("Management")
	d.Field("Managed by Firewall Manager", fmt.Sprintf("%v", webacl.ManagedByFirewallManager()))
	if ns := webacl.LabelNamespace(); ns != "" {
		d.Field("Label Namespace", ns)
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
func (r *WebACLRenderer) RenderSummary(resource dao.Resource) []render.SummaryField {
	webacl, ok := resource.(*WebACLResource)
	if !ok {
		return r.BaseRenderer.RenderSummary(resource)
	}

	fields := []render.SummaryField{
		{Label: "Name", Value: webacl.WebACLName()},
		{Label: "ID", Value: webacl.WebACLId()},
		{Label: "Scope", Value: webacl.ScopeString()},
		{Label: "Default Action", Value: webacl.DefaultAction()},
	}

	if count := webacl.RuleCount(); count > 0 {
		fields = append(fields, render.SummaryField{Label: "Rules", Value: fmt.Sprintf("%d", count)})
	}

	if arn := webacl.GetARN(); arn != "" {
		fields = append(fields, render.SummaryField{Label: "ARN", Value: arn})
	}

	if capacity := webacl.Capacity(); capacity > 0 {
		fields = append(fields, render.SummaryField{Label: "Capacity", Value: fmt.Sprintf("%d WCU", capacity)})
	}

	return fields
}

// Navigations returns navigation shortcuts
func (r *WebACLRenderer) Navigations(resource dao.Resource) []render.Navigation {
	return nil
}
