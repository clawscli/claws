package protectedresources

import (
	"fmt"
	"strings"

	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/render"
)

// Ensure ProtectedResourceRenderer implements render.Navigator
var _ render.Navigator = (*ProtectedResourceRenderer)(nil)

// ProtectedResourceRenderer renders AWS Backup protected resources
type ProtectedResourceRenderer struct {
	render.BaseRenderer
}

// NewProtectedResourceRenderer creates a new ProtectedResourceRenderer
func NewProtectedResourceRenderer() *ProtectedResourceRenderer {
	return &ProtectedResourceRenderer{
		BaseRenderer: render.BaseRenderer{
			Service:  "backup",
			Resource: "protected-resources",
			Cols: []render.Column{
				{Name: "RESOURCE TYPE", Width: 18, Getter: getResourceType},
				{Name: "RESOURCE NAME", Width: 30, Getter: getResourceName},
				{Name: "LAST BACKUP", Width: 20, Getter: getLastBackup},
				{Name: "RESOURCE ARN", Width: 60, Getter: getResourceArn},
			},
		},
	}
}

func getResourceType(r dao.Resource) string {
	if pr, ok := r.(*ProtectedResourceResource); ok {
		return pr.ResourceType()
	}
	return ""
}

func getResourceName(r dao.Resource) string {
	if pr, ok := r.(*ProtectedResourceResource); ok {
		name := pr.ResourceName()
		if name != "" {
			return name
		}
		// Extract from ARN if no name
		arn := pr.ResourceArn()
		parts := strings.Split(arn, "/")
		if len(parts) > 1 {
			return parts[len(parts)-1]
		}
		parts = strings.Split(arn, ":")
		if len(parts) > 0 {
			return parts[len(parts)-1]
		}
	}
	return ""
}

func getLastBackup(r dao.Resource) string {
	if pr, ok := r.(*ProtectedResourceResource); ok {
		return pr.LastBackupTime()
	}
	return "-"
}

func getResourceArn(r dao.Resource) string {
	if pr, ok := r.(*ProtectedResourceResource); ok {
		arn := pr.ResourceArn()
		if len(arn) > 60 {
			return "..." + arn[len(arn)-57:]
		}
		return arn
	}
	return ""
}

// RenderDetail renders detailed protected resource information
func (r *ProtectedResourceRenderer) RenderDetail(resource dao.Resource) string {
	pr, ok := resource.(*ProtectedResourceResource)
	if !ok {
		return ""
	}

	d := render.NewDetailBuilder()

	d.Title("Protected Resource", pr.ResourceType())

	// Basic Info
	d.Section("Resource Information")
	d.Field("ARN", pr.ResourceArn())
	d.Field("Type", pr.ResourceType())
	if name := pr.ResourceName(); name != "" {
		d.Field("Name", name)
	}

	// Last Backup
	d.Section("Last Backup")
	if lastBackup := pr.LastBackupTime(); lastBackup != "" {
		d.Field("Time", lastBackup)
	}
	if vaultArn := pr.LastBackupVaultArn(); vaultArn != "" {
		d.Field("Vault ARN", vaultArn)
	}
	if rpArn := pr.LastRecoveryPointArn(); rpArn != "" {
		d.Field("Recovery Point ARN", rpArn)
	}

	// Latest Restore Info
	if latestRestoreDate := pr.LatestRestoreJobCreationDate(); latestRestoreDate != "" {
		d.Section("Latest Restore")
		d.Field("Job Created", latestRestoreDate)
		if rpDate := pr.LatestRestoreRecoveryPointCreationDate(); rpDate != "" {
			d.Field("Recovery Point Date", rpDate)
		}
		if execMin := pr.LatestRestoreExecutionTimeMinutes(); execMin > 0 {
			d.Field("Execution Time", fmt.Sprintf("%d minutes", execMin))
		}
	}

	return d.String()
}

// RenderSummary returns summary fields for the header panel
func (r *ProtectedResourceRenderer) RenderSummary(resource dao.Resource) []render.SummaryField {
	pr, ok := resource.(*ProtectedResourceResource)
	if !ok {
		return r.BaseRenderer.RenderSummary(resource)
	}

	fields := []render.SummaryField{
		{Label: "Resource Type", Value: pr.ResourceType()},
		{Label: "Resource ARN", Value: pr.ResourceArn()},
	}

	if name := pr.ResourceName(); name != "" {
		fields = append(fields, render.SummaryField{Label: "Name", Value: name})
	}

	if lastBackup := pr.LastBackupTime(); lastBackup != "" {
		fields = append(fields, render.SummaryField{Label: "Last Backup", Value: lastBackup})
	}

	return fields
}

// Navigations returns navigation shortcuts
func (r *ProtectedResourceRenderer) Navigations(resource dao.Resource) []render.Navigation {
	pr, ok := resource.(*ProtectedResourceResource)
	if !ok {
		return nil
	}

	var navs []render.Navigation

	// Navigate to last backup vault
	if vaultArn := pr.LastBackupVaultArn(); vaultArn != "" {
		vaultName := extractVaultNameFromArn(vaultArn)
		if vaultName != "" {
			navs = append(navs, render.Navigation{
				Key: "v", Label: "Vault", Service: "backup", Resource: "vaults",
				FilterField: "Name", FilterValue: vaultName,
			})
		}
	}

	return navs
}

// extractVaultNameFromArn extracts vault name from ARN
func extractVaultNameFromArn(arn string) string {
	parts := strings.Split(arn, ":")
	if len(parts) >= 7 {
		return parts[len(parts)-1]
	}
	return ""
}
