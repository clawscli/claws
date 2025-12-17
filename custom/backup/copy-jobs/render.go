package copyjobs

import (
	"fmt"
	"strings"

	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/render"
)

// Ensure CopyJobRenderer implements render.Navigator
var _ render.Navigator = (*CopyJobRenderer)(nil)

// CopyJobRenderer renders AWS Backup copy jobs
type CopyJobRenderer struct {
	render.BaseRenderer
}

// NewCopyJobRenderer creates a new CopyJobRenderer
func NewCopyJobRenderer() *CopyJobRenderer {
	return &CopyJobRenderer{
		BaseRenderer: render.BaseRenderer{
			Service:  "backup",
			Resource: "copy-jobs",
			Cols: []render.Column{
				{Name: "JOB ID", Width: 36, Getter: getJobId},
				{Name: "STATE", Width: 12, Getter: getState},
				{Name: "RESOURCE TYPE", Width: 15, Getter: getResourceType},
				{Name: "SIZE", Width: 12, Getter: getSize},
				{Name: "CREATED", Width: 20, Getter: getCreated},
			},
		},
	}
}

func getJobId(r dao.Resource) string {
	if j, ok := r.(*CopyJobResource); ok {
		return j.JobId()
	}
	return ""
}

func getState(r dao.Resource) string {
	if j, ok := r.(*CopyJobResource); ok {
		return j.State()
	}
	return ""
}

func getResourceType(r dao.Resource) string {
	if j, ok := r.(*CopyJobResource); ok {
		return j.ResourceType()
	}
	return ""
}

func getSize(r dao.Resource) string {
	if j, ok := r.(*CopyJobResource); ok {
		return j.BackupSizeFormatted()
	}
	return "-"
}

func getCreated(r dao.Resource) string {
	if j, ok := r.(*CopyJobResource); ok {
		return j.CreationDate()
	}
	return "-"
}

// RenderDetail renders detailed copy job information
func (r *CopyJobRenderer) RenderDetail(resource dao.Resource) string {
	job, ok := resource.(*CopyJobResource)
	if !ok {
		return ""
	}

	d := render.NewDetailBuilder()

	d.Title("Copy Job", job.JobId())

	// Basic Info
	d.Section("Basic Information")
	d.Field("Job ID", job.JobId())
	d.Field("State", job.State())
	if msg := job.StatusMessage(); msg != "" {
		d.Field("Status Message", msg)
	}
	if msgCategory := job.MessageCategory(); msgCategory != "" {
		d.Field("Message Category", msgCategory)
	}

	// Resource
	d.Section("Resource")
	d.Field("Type", job.ResourceType())
	if arn := job.ResourceArn(); arn != "" {
		d.Field("ARN", arn)
	}

	// Source
	d.Section("Source")
	if srcVault := job.SourceBackupVaultArn(); srcVault != "" {
		d.Field("Vault ARN", srcVault)
	}
	if srcRP := job.SourceRecoveryPointArn(); srcRP != "" {
		d.Field("Recovery Point ARN", srcRP)
	}

	// Destination
	d.Section("Destination")
	if dstVault := job.DestinationBackupVaultArn(); dstVault != "" {
		d.Field("Vault ARN", dstVault)
	}
	if dstRP := job.DestinationRecoveryPointArn(); dstRP != "" {
		d.Field("Recovery Point ARN", dstRP)
	}

	// Copy Details
	d.Section("Copy Details")
	if size := job.BackupSizeFormatted(); size != "-" {
		d.Field("Size", size)
	}
	if accountId := job.AccountId(); accountId != "" {
		d.Field("Account ID", accountId)
	}

	// Hierarchy
	if job.IsParent() || job.ParentJobId() != "" || job.CompositeMemberIdentifier() != "" {
		d.Section("Hierarchy")
		if job.IsParent() {
			d.Field("Is Parent", "Yes")
			if numChildren := job.NumberOfChildJobs(); numChildren > 0 {
				d.Field("Child Jobs", fmt.Sprintf("%d", numChildren))
			}
		}
		if parentId := job.ParentJobId(); parentId != "" {
			d.Field("Parent Job ID", parentId)
		}
		if compositeMember := job.CompositeMemberIdentifier(); compositeMember != "" {
			d.Field("Composite Member", compositeMember)
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
	if completed := job.CompletionDate(); completed != "" {
		d.Field("Completed", completed)
	}

	return d.String()
}

// RenderSummary returns summary fields for the header panel
func (r *CopyJobRenderer) RenderSummary(resource dao.Resource) []render.SummaryField {
	job, ok := resource.(*CopyJobResource)
	if !ok {
		return r.BaseRenderer.RenderSummary(resource)
	}

	fields := []render.SummaryField{
		{Label: "Job ID", Value: job.JobId()},
		{Label: "State", Value: job.State()},
		{Label: "Resource Type", Value: job.ResourceType()},
	}

	if size := job.BackupSizeFormatted(); size != "-" {
		fields = append(fields, render.SummaryField{Label: "Size", Value: size})
	}

	if created := job.CreationDate(); created != "" {
		fields = append(fields, render.SummaryField{Label: "Created", Value: created})
	}

	return fields
}

// Navigations returns navigation shortcuts
func (r *CopyJobRenderer) Navigations(resource dao.Resource) []render.Navigation {
	job, ok := resource.(*CopyJobResource)
	if !ok {
		return nil
	}

	var navs []render.Navigation

	// Navigate to source vault
	if srcVaultArn := job.SourceBackupVaultArn(); srcVaultArn != "" {
		vaultName := extractVaultNameFromArn(srcVaultArn)
		if vaultName != "" {
			navs = append(navs, render.Navigation{
				Key: "s", Label: "Source Vault", Service: "backup", Resource: "vaults",
				FilterField: "Name", FilterValue: vaultName,
			})
		}
	}

	// Navigate to destination vault
	if dstVaultArn := job.DestinationBackupVaultArn(); dstVaultArn != "" {
		vaultName := extractVaultNameFromArn(dstVaultArn)
		if vaultName != "" {
			navs = append(navs, render.Navigation{
				Key: "t", Label: "Dest Vault", Service: "backup", Resource: "vaults",
				FilterField: "Name", FilterValue: vaultName,
			})
		}
	}

	return navs
}

// extractVaultNameFromArn extracts vault name from ARN
// ARN format: arn:aws:backup:region:account:backup-vault:vault-name
func extractVaultNameFromArn(arn string) string {
	parts := strings.Split(arn, ":")
	if len(parts) >= 7 {
		return parts[len(parts)-1]
	}
	return ""
}
