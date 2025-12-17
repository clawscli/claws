package backupjobs

import (
	"fmt"

	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/render"
)

// Ensure BackupJobRenderer implements render.Navigator
var _ render.Navigator = (*BackupJobRenderer)(nil)

// BackupJobRenderer renders AWS Backup jobs
type BackupJobRenderer struct {
	render.BaseRenderer
}

// NewBackupJobRenderer creates a new BackupJobRenderer
func NewBackupJobRenderer() *BackupJobRenderer {
	return &BackupJobRenderer{
		BaseRenderer: render.BaseRenderer{
			Service:  "backup",
			Resource: "backup-jobs",
			Cols: []render.Column{
				{Name: "JOB ID", Width: 36, Getter: getJobId},
				{Name: "STATE", Width: 12, Getter: getState},
				{Name: "RESOURCE TYPE", Width: 15, Getter: getResourceType},
				{Name: "SIZE", Width: 12, Getter: getSize},
				{Name: "PROGRESS", Width: 10, Getter: getProgress},
				{Name: "CREATED", Width: 20, Getter: getCreated},
			},
		},
	}
}

func getJobId(r dao.Resource) string {
	if j, ok := r.(*BackupJobResource); ok {
		return j.JobId()
	}
	return ""
}

func getState(r dao.Resource) string {
	if j, ok := r.(*BackupJobResource); ok {
		return j.State()
	}
	return ""
}

func getResourceType(r dao.Resource) string {
	if j, ok := r.(*BackupJobResource); ok {
		return j.ResourceType()
	}
	return ""
}

func getSize(r dao.Resource) string {
	if j, ok := r.(*BackupJobResource); ok {
		return j.BackupSizeFormatted()
	}
	return "-"
}

func getProgress(r dao.Resource) string {
	if j, ok := r.(*BackupJobResource); ok {
		if pct := j.PercentDone(); pct != "" {
			return pct + "%"
		}
	}
	return "-"
}

func getCreated(r dao.Resource) string {
	if j, ok := r.(*BackupJobResource); ok {
		return j.CreationDate()
	}
	return "-"
}

// RenderDetail renders detailed job information
func (r *BackupJobRenderer) RenderDetail(resource dao.Resource) string {
	job, ok := resource.(*BackupJobResource)
	if !ok {
		return ""
	}

	d := render.NewDetailBuilder()

	d.Title("Backup Job", job.JobId())

	// Basic Info
	d.Section("Basic Information")
	d.Field("Job ID", job.JobId())
	d.Field("State", job.State())
	if backupType := job.BackupType(); backupType != "" {
		d.Field("Backup Type", backupType)
	}
	if msgCategory := job.MessageCategory(); msgCategory != "" {
		d.Field("Message Category", msgCategory)
	}

	// Resource
	d.Section("Resource")
	d.Field("Type", job.ResourceType())
	if name := job.ResourceName(); name != "" {
		d.Field("Name", name)
	}
	if arn := job.ResourceArn(); arn != "" {
		d.Field("ARN", arn)
	}

	// Backup Plan
	if planId := job.BackupPlanId(); planId != "" {
		d.Section("Backup Plan")
		d.Field("Plan ID", planId)
	}

	// Backup Details
	d.Section("Backup Details")
	if vault := job.BackupVaultName(); vault != "" {
		d.Field("Vault", vault)
	}
	if vaultArn := job.BackupVaultArn(); vaultArn != "" {
		d.Field("Vault ARN", vaultArn)
	}
	if rpArn := job.RecoveryPointArn(); rpArn != "" {
		d.Field("Recovery Point ARN", rpArn)
	}
	if size := job.BackupSizeFormatted(); size != "-" {
		d.Field("Size", size)
	}
	if pct := job.PercentDone(); pct != "" {
		d.Field("Progress", pct+"%")
	}
	if msg := job.StatusMessage(); msg != "" {
		d.Field("Status Message", msg)
	}
	if bytes := job.BytesTransferred(); bytes > 0 {
		d.Field("Bytes Transferred", render.FormatSize(bytes))
	}

	// Encryption
	if job.IsEncrypted() {
		d.Section("Encryption")
		d.Field("Encrypted", "Yes")
		if keyArn := job.EncryptionKeyArn(); keyArn != "" {
			d.Field("Key ARN", keyArn)
		}
	}

	// Hierarchy
	if job.IsParent() || job.ParentJobId() != "" {
		d.Section("Hierarchy")
		if job.IsParent() {
			d.Field("Is Parent", "Yes")
		}
		if parentId := job.ParentJobId(); parentId != "" {
			d.Field("Parent Job ID", parentId)
		}
	}

	// IAM
	if roleArn := job.IamRoleArn(); roleArn != "" {
		d.Section("IAM")
		d.Field("Role ARN", roleArn)
	}

	// Timestamps
	d.Section("Timestamps")
	if created := job.CreationDate(); created != "" {
		d.Field("Created", created)
	}
	if initiation := job.InitiationDate(); initiation != "" {
		d.Field("Initiated", initiation)
	}
	if startBy := job.StartBy(); startBy != "" {
		d.Field("Start By", startBy)
	}
	if completed := job.CompletionDate(); completed != "" {
		d.Field("Completed", completed)
	}
	if expected := job.ExpectedCompletionDate(); expected != "" {
		d.Field("Expected Completion", expected)
	}

	// Account
	if accountId := job.AccountId(); accountId != "" {
		d.Section("Account")
		d.Field("Account ID", accountId)
	}

	return d.String()
}

// RenderSummary returns summary fields for the header panel
func (r *BackupJobRenderer) RenderSummary(resource dao.Resource) []render.SummaryField {
	job, ok := resource.(*BackupJobResource)
	if !ok {
		return r.BaseRenderer.RenderSummary(resource)
	}

	fields := []render.SummaryField{
		{Label: "Job ID", Value: job.JobId()},
		{Label: "State", Value: job.State()},
		{Label: "Resource Type", Value: job.ResourceType()},
	}

	if vault := job.BackupVaultName(); vault != "" {
		fields = append(fields, render.SummaryField{Label: "Vault", Value: vault})
	}

	if size := job.BackupSizeFormatted(); size != "-" {
		fields = append(fields, render.SummaryField{Label: "Size", Value: size})
	}

	if pct := job.PercentDone(); pct != "" {
		fields = append(fields, render.SummaryField{Label: "Progress", Value: fmt.Sprintf("%s%%", pct)})
	}

	if created := job.CreationDate(); created != "" {
		fields = append(fields, render.SummaryField{Label: "Created", Value: created})
	}

	return fields
}

// Navigations returns navigation shortcuts
func (r *BackupJobRenderer) Navigations(resource dao.Resource) []render.Navigation {
	job, ok := resource.(*BackupJobResource)
	if !ok {
		return nil
	}

	var navs []render.Navigation

	// Navigate to vault
	if vaultName := job.BackupVaultName(); vaultName != "" {
		navs = append(navs, render.Navigation{
			Key: "v", Label: "Vault", Service: "backup", Resource: "vaults",
			FilterField: "Name", FilterValue: vaultName,
		})
	}

	// Navigate to backup plan
	if planId := job.BackupPlanId(); planId != "" {
		navs = append(navs, render.Navigation{
			Key: "p", Label: "Plan", Service: "backup", Resource: "plans",
			FilterField: "PlanId", FilterValue: planId,
		})
	}

	// Navigate to recovery point
	if rpArn := job.RecoveryPointArn(); rpArn != "" && job.BackupVaultName() != "" {
		navs = append(navs, render.Navigation{
			Key: "r", Label: "Recovery Point", Service: "backup", Resource: "recovery-points",
			FilterField: "VaultName", FilterValue: job.BackupVaultName(),
		})
	}

	return navs
}
