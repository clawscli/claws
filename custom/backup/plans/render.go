package plans

import (
	"fmt"

	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/render"
)

// BackupPlanRenderer renders AWS Backup plans
// Ensure BackupPlanRenderer implements render.Navigator
var _ render.Navigator = (*BackupPlanRenderer)(nil)

type BackupPlanRenderer struct {
	render.BaseRenderer
}

// NewBackupPlanRenderer creates a new BackupPlanRenderer
func NewBackupPlanRenderer() *BackupPlanRenderer {
	return &BackupPlanRenderer{
		BaseRenderer: render.BaseRenderer{
			Service:  "backup",
			Resource: "plans",
			Cols: []render.Column{
				{Name: "NAME", Width: 30, Getter: getName},
				{Name: "ID", Width: 40, Getter: getPlanId},
				{Name: "LAST RUN", Width: 20, Getter: getLastRun},
				{Name: "AGE", Width: 12, Getter: getAge},
			},
		},
	}
}

func getName(r dao.Resource) string {
	if p, ok := r.(*BackupPlanResource); ok {
		return p.PlanName()
	}
	return ""
}

func getPlanId(r dao.Resource) string {
	if p, ok := r.(*BackupPlanResource); ok {
		return p.PlanId()
	}
	return ""
}

func getLastRun(r dao.Resource) string {
	if p, ok := r.(*BackupPlanResource); ok {
		if lastRun := p.LastExecutionDate(); lastRun != "" {
			return lastRun
		}
	}
	return "-"
}

func getAge(r dao.Resource) string {
	if p, ok := r.(*BackupPlanResource); ok {
		if t := p.CreatedAtTime(); t != nil {
			return render.FormatAge(*t)
		}
	}
	return "-"
}

// RenderDetail renders detailed backup plan information
func (r *BackupPlanRenderer) RenderDetail(resource dao.Resource) string {
	plan, ok := resource.(*BackupPlanResource)
	if !ok {
		return ""
	}

	d := render.NewDetailBuilder()

	d.Title("AWS Backup Plan", plan.PlanName())

	// Basic Info
	d.Section("Basic Information")
	d.Field("Name", plan.PlanName())
	d.Field("Plan ID", plan.PlanId())
	if arn := plan.GetARN(); arn != "" {
		d.Field("ARN", arn)
	}
	if versionId := plan.VersionId(); versionId != "" {
		d.Field("Version ID", versionId)
	}

	// Rules
	if rules := plan.Rules(); len(rules) > 0 {
		d.Section("Backup Rules")
		for i, rule := range rules {
			d.Field(fmt.Sprintf("Rule %d Name", i+1), deref(rule.RuleName))

			if rule.TargetBackupVaultName != nil {
				d.Field(fmt.Sprintf("Rule %d Vault", i+1), *rule.TargetBackupVaultName)
			}

			if rule.ScheduleExpression != nil {
				d.Field(fmt.Sprintf("Rule %d Schedule", i+1), *rule.ScheduleExpression)
			}

			if rule.StartWindowMinutes != nil {
				d.Field(fmt.Sprintf("Rule %d Start Window", i+1), fmt.Sprintf("%d minutes", *rule.StartWindowMinutes))
			}

			if rule.CompletionWindowMinutes != nil {
				d.Field(fmt.Sprintf("Rule %d Completion Window", i+1), fmt.Sprintf("%d minutes", *rule.CompletionWindowMinutes))
			}

			if rule.Lifecycle != nil {
				if rule.Lifecycle.DeleteAfterDays != nil {
					d.Field(fmt.Sprintf("Rule %d Retention", i+1), fmt.Sprintf("%d days", *rule.Lifecycle.DeleteAfterDays))
				}
				if rule.Lifecycle.MoveToColdStorageAfterDays != nil {
					d.Field(fmt.Sprintf("Rule %d Cold Storage After", i+1), fmt.Sprintf("%d days", *rule.Lifecycle.MoveToColdStorageAfterDays))
				}
			}

			d.Field("", "") // Spacer between rules
		}
	}

	// Advanced Settings
	if settings := plan.AdvancedBackupSettings(); len(settings) > 0 {
		d.Section("Advanced Settings")
		for _, setting := range settings {
			if setting.ResourceType != nil {
				d.Field("Resource Type", *setting.ResourceType)
			}
		}
	}

	// Timestamps
	d.Section("Timestamps")
	if created := plan.CreatedAt(); created != "" {
		d.Field("Created", created)
	}
	if lastRun := plan.LastExecutionDate(); lastRun != "" {
		d.Field("Last Execution", lastRun)
	}
	if deleted := plan.DeletionDate(); deleted != "" {
		d.Field("Deletion Scheduled", deleted)
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
func (r *BackupPlanRenderer) RenderSummary(resource dao.Resource) []render.SummaryField {
	plan, ok := resource.(*BackupPlanResource)
	if !ok {
		return r.BaseRenderer.RenderSummary(resource)
	}

	fields := []render.SummaryField{
		{Label: "Name", Value: plan.PlanName()},
		{Label: "Plan ID", Value: plan.PlanId()},
	}

	if arn := plan.GetARN(); arn != "" {
		fields = append(fields, render.SummaryField{Label: "ARN", Value: arn})
	}

	if rules := plan.Rules(); len(rules) > 0 {
		fields = append(fields, render.SummaryField{Label: "Rules", Value: fmt.Sprintf("%d", len(rules))})
	}

	if lastRun := plan.LastExecutionDate(); lastRun != "" {
		fields = append(fields, render.SummaryField{Label: "Last Run", Value: lastRun})
	}

	if created := plan.CreatedAt(); created != "" {
		fields = append(fields, render.SummaryField{Label: "Created", Value: created})
	}

	return fields
}

// Navigations returns navigation shortcuts
func (r *BackupPlanRenderer) Navigations(resource dao.Resource) []render.Navigation {
	plan, ok := resource.(*BackupPlanResource)
	if !ok {
		return nil
	}

	return []render.Navigation{
		{
			Key: "o", Label: "Jobs", Service: "backup", Resource: "jobs",
			FilterField: "BackupPlanId", FilterValue: plan.PlanId(),
		},
	}
}
