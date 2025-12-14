package notifications

import (
	"fmt"

	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/render"
)

// NotificationRenderer renders Budget notifications.
type NotificationRenderer struct {
	render.BaseRenderer
}

// NewNotificationRenderer creates a new NotificationRenderer.
func NewNotificationRenderer() render.Renderer {
	return &NotificationRenderer{
		BaseRenderer: render.BaseRenderer{
			Service:  "budgets",
			Resource: "notifications",
			Cols: []render.Column{
				{Name: "TYPE", Width: 12, Getter: getType},
				{Name: "COMPARISON", Width: 18, Getter: getComparison},
				{Name: "THRESHOLD", Width: 12, Getter: getThreshold},
				{Name: "THRESHOLD TYPE", Width: 16, Getter: getThresholdType},
				{Name: "STATE", Width: 12, Getter: getState},
			},
		},
	}
}

func getType(r dao.Resource) string {
	notif, ok := r.(*NotificationResource)
	if !ok {
		return ""
	}
	return notif.NotificationType()
}

func getComparison(r dao.Resource) string {
	notif, ok := r.(*NotificationResource)
	if !ok {
		return ""
	}
	return notif.ComparisonOperator()
}

func getThreshold(r dao.Resource) string {
	notif, ok := r.(*NotificationResource)
	if !ok {
		return ""
	}
	return fmt.Sprintf("%.2f%%", notif.Threshold())
}

func getThresholdType(r dao.Resource) string {
	notif, ok := r.(*NotificationResource)
	if !ok {
		return ""
	}
	return notif.ThresholdType()
}

func getState(r dao.Resource) string {
	notif, ok := r.(*NotificationResource)
	if !ok {
		return ""
	}
	return notif.NotificationState()
}

// RenderDetail renders the detail view for a Budget notification.
func (r *NotificationRenderer) RenderDetail(resource dao.Resource) string {
	notif, ok := resource.(*NotificationResource)
	if !ok {
		return ""
	}

	d := render.NewDetailBuilder()

	d.Title("Budget Notification", notif.GetID())

	// Basic Info
	d.Section("Basic Information")
	d.Field("Budget Name", notif.BudgetName)
	d.Field("Notification Type", notif.NotificationType())
	d.Field("State", notif.NotificationState())

	// Threshold
	d.Section("Threshold Configuration")
	d.Field("Comparison Operator", notif.ComparisonOperator())
	d.Field("Threshold", fmt.Sprintf("%.2f", notif.Threshold()))
	d.Field("Threshold Type", notif.ThresholdType())

	return d.String()
}

// RenderSummary renders summary fields for a Budget notification.
func (r *NotificationRenderer) RenderSummary(resource dao.Resource) []render.SummaryField {
	notif, ok := resource.(*NotificationResource)
	if !ok {
		return r.BaseRenderer.RenderSummary(resource)
	}

	return []render.SummaryField{
		{Label: "Budget Name", Value: notif.BudgetName},
		{Label: "Type", Value: notif.NotificationType()},
		{Label: "Threshold", Value: fmt.Sprintf("%.2f%%", notif.Threshold())},
		{Label: "State", Value: notif.NotificationState()},
	}
}
