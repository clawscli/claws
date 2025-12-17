package recoverypoints

import (
	"fmt"

	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/render"
)

// Ensure RecoveryPointRenderer implements render.Navigator
var _ render.Navigator = (*RecoveryPointRenderer)(nil)

// RecoveryPointRenderer renders AWS Backup recovery points
type RecoveryPointRenderer struct {
	render.BaseRenderer
}

// NewRecoveryPointRenderer creates a new RecoveryPointRenderer
func NewRecoveryPointRenderer() *RecoveryPointRenderer {
	return &RecoveryPointRenderer{
		BaseRenderer: render.BaseRenderer{
			Service:  "backup",
			Resource: "recovery-points",
			Cols: []render.Column{
				{Name: "RESOURCE TYPE", Width: 15, Getter: getResourceType},
				{Name: "STATUS", Width: 12, Getter: getStatus},
				{Name: "SIZE", Width: 12, Getter: getSize},
				{Name: "CREATED", Width: 20, Getter: getCreated},
				{Name: "RECOVERY POINT", Width: 50, Getter: getRecoveryPointId},
			},
		},
	}
}

func getResourceType(r dao.Resource) string {
	if rp, ok := r.(*RecoveryPointResource); ok {
		return rp.ResourceType()
	}
	return ""
}

func getStatus(r dao.Resource) string {
	if rp, ok := r.(*RecoveryPointResource); ok {
		return rp.Status()
	}
	return ""
}

func getSize(r dao.Resource) string {
	if rp, ok := r.(*RecoveryPointResource); ok {
		return rp.BackupSizeFormatted()
	}
	return "-"
}

func getCreated(r dao.Resource) string {
	if rp, ok := r.(*RecoveryPointResource); ok {
		return rp.CreationDate()
	}
	return ""
}

func getRecoveryPointId(r dao.Resource) string {
	if rp, ok := r.(*RecoveryPointResource); ok {
		arn := rp.RecoveryPointArn()
		if len(arn) > 50 {
			return "..." + arn[len(arn)-47:]
		}
		return arn
	}
	return ""
}

// RenderDetail renders detailed recovery point information
func (r *RecoveryPointRenderer) RenderDetail(resource dao.Resource) string {
	rp, ok := resource.(*RecoveryPointResource)
	if !ok {
		return ""
	}

	d := render.NewDetailBuilder()

	d.Title("AWS Backup Recovery Point", rp.ResourceType())

	// Basic Info
	d.Section("Basic Information")
	d.Field("Recovery Point ARN", rp.RecoveryPointArn())
	d.Field("Status", rp.Status())
	if msg := rp.StatusMessage(); msg != "" {
		d.Field("Status Message", msg)
	}
	d.Field("Vault", rp.VaultName)

	// Resource Info
	d.Section("Resource")
	d.Field("Type", rp.ResourceType())
	if arn := rp.ResourceArn(); arn != "" {
		d.Field("ARN", arn)
	}
	if name := rp.ResourceName(); name != "" {
		d.Field("Name", name)
	}

	// Backup Details
	d.Section("Backup Details")
	if size := rp.BackupSizeFormatted(); size != "-" {
		d.Field("Size", size)
	}
	if storageClass := rp.StorageClass(); storageClass != "" {
		d.Field("Storage Class", storageClass)
	}
	if rp.IsEncrypted() {
		d.Field("Encrypted", "Yes")
		if keyArn := rp.EncryptionKeyArn(); keyArn != "" {
			d.Field("Encryption Key", keyArn)
		}
	} else {
		d.Field("Encrypted", "No")
	}

	// Hierarchy
	if rp.IsParent() || rp.ParentRecoveryPointArn() != "" || rp.CompositeMemberIdentifier() != "" {
		d.Section("Hierarchy")
		if rp.IsParent() {
			d.Field("Is Parent", "Yes")
		}
		if parentArn := rp.ParentRecoveryPointArn(); parentArn != "" {
			d.Field("Parent Recovery Point", parentArn)
		}
		if compositeMember := rp.CompositeMemberIdentifier(); compositeMember != "" {
			d.Field("Composite Member", compositeMember)
		}
	}

	// Lifecycle
	if lifecycle := rp.Lifecycle(); lifecycle != nil {
		d.Section("Lifecycle")
		if lifecycle.DeleteAfterDays != nil {
			d.Field("Delete After", fmt.Sprintf("%d days", *lifecycle.DeleteAfterDays))
		}
		if lifecycle.MoveToColdStorageAfterDays != nil {
			d.Field("Move to Cold Storage After", fmt.Sprintf("%d days", *lifecycle.MoveToColdStorageAfterDays))
		}
		if lifecycle.OptInToArchiveForSupportedResources != nil && *lifecycle.OptInToArchiveForSupportedResources {
			d.Field("Archive Opt-In", "Yes")
		}
	}

	// Calculated Lifecycle
	if calcLifecycle := rp.CalculatedLifecycle(); calcLifecycle != nil {
		d.Section("Calculated Lifecycle")
		if calcLifecycle.DeleteAt != nil {
			d.Field("Delete At", calcLifecycle.DeleteAt.Format("2006-01-02 15:04:05"))
		}
		if calcLifecycle.MoveToColdStorageAt != nil {
			d.Field("Move to Cold Storage At", calcLifecycle.MoveToColdStorageAt.Format("2006-01-02 15:04:05"))
		}
	}

	// Created By
	if createdBy := rp.CreatedBy(); createdBy != nil {
		d.Section("Created By")
		if createdBy.BackupPlanId != nil {
			d.Field("Backup Plan ID", *createdBy.BackupPlanId)
		}
		if createdBy.BackupPlanArn != nil {
			d.Field("Backup Plan ARN", *createdBy.BackupPlanArn)
		}
		if createdBy.BackupPlanVersion != nil {
			d.Field("Backup Plan Version", *createdBy.BackupPlanVersion)
		}
		if createdBy.BackupRuleId != nil {
			d.Field("Backup Rule ID", *createdBy.BackupRuleId)
		}
	}

	// IAM Role
	if roleArn := rp.IamRoleArn(); roleArn != "" {
		d.Section("IAM")
		d.Field("Role ARN", roleArn)
	}

	// Timestamps
	d.Section("Timestamps")
	if created := rp.CreationDate(); created != "" {
		d.Field("Created", created)
	}
	if completed := rp.CompletionDate(); completed != "" {
		d.Field("Completed", completed)
	}
	if lastRestore := rp.LastRestoreTime(); lastRestore != "" {
		d.Field("Last Restore", lastRestore)
	}

	return d.String()
}

// RenderSummary returns summary fields for the header panel
func (r *RecoveryPointRenderer) RenderSummary(resource dao.Resource) []render.SummaryField {
	rp, ok := resource.(*RecoveryPointResource)
	if !ok {
		return r.BaseRenderer.RenderSummary(resource)
	}

	fields := []render.SummaryField{
		{Label: "Recovery Point", Value: rp.RecoveryPointArn()},
		{Label: "Status", Value: rp.Status()},
		{Label: "Resource Type", Value: rp.ResourceType()},
	}

	if size := rp.BackupSizeFormatted(); size != "-" {
		fields = append(fields, render.SummaryField{Label: "Size", Value: size})
	}

	if created := rp.CreationDate(); created != "" {
		fields = append(fields, render.SummaryField{Label: "Created", Value: created})
	}

	return fields
}

// Navigations returns navigation shortcuts
func (r *RecoveryPointRenderer) Navigations(resource dao.Resource) []render.Navigation {
	rp, ok := resource.(*RecoveryPointResource)
	if !ok {
		return nil
	}

	var navs []render.Navigation

	// Navigate to vault
	navs = append(navs, render.Navigation{
		Key: "v", Label: "Vault", Service: "backup", Resource: "vaults",
		FilterField: "Name", FilterValue: rp.VaultName,
	})

	// Navigate to backup plan if available
	if createdBy := rp.CreatedBy(); createdBy != nil && createdBy.BackupPlanId != nil {
		navs = append(navs, render.Navigation{
			Key: "p", Label: "Plan", Service: "backup", Resource: "plans",
			FilterField: "PlanId", FilterValue: *createdBy.BackupPlanId,
		})
	}

	return navs
}
