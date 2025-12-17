package vaults

import (
	"fmt"

	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/render"
)

// Ensure VaultRenderer implements render.Navigator
var _ render.Navigator = (*VaultRenderer)(nil)

// VaultRenderer renders AWS Backup vaults
type VaultRenderer struct {
	render.BaseRenderer
}

// NewVaultRenderer creates a new VaultRenderer
func NewVaultRenderer() *VaultRenderer {
	return &VaultRenderer{
		BaseRenderer: render.BaseRenderer{
			Service:  "backup",
			Resource: "vaults",
			Cols: []render.Column{
				{Name: "NAME", Width: 35, Getter: getVaultName},
				{Name: "RECOVERY POINTS", Width: 16, Getter: getRecoveryPointCount},
				{Name: "LOCKED", Width: 8, Getter: getLocked},
				{Name: "AGE", Width: 12, Getter: getAge},
			},
		},
	}
}

func getVaultName(r dao.Resource) string {
	if v, ok := r.(*VaultResource); ok {
		return v.VaultName()
	}
	return ""
}

func getRecoveryPointCount(r dao.Resource) string {
	if v, ok := r.(*VaultResource); ok {
		return fmt.Sprintf("%d", v.RecoveryPointCount())
	}
	return "0"
}

func getLocked(r dao.Resource) string {
	if v, ok := r.(*VaultResource); ok {
		if v.Locked() {
			return "Yes"
		}
		return "No"
	}
	return "-"
}

func getAge(r dao.Resource) string {
	if v, ok := r.(*VaultResource); ok {
		if t := v.CreatedAtTime(); t != nil {
			return render.FormatAge(*t)
		}
	}
	return "-"
}

// RenderDetail renders detailed vault information
func (r *VaultRenderer) RenderDetail(resource dao.Resource) string {
	vault, ok := resource.(*VaultResource)
	if !ok {
		return ""
	}

	d := render.NewDetailBuilder()

	d.Title("AWS Backup Vault", vault.VaultName())

	// Basic Info
	d.Section("Basic Information")
	d.Field("Name", vault.VaultName())
	if arn := vault.VaultArn(); arn != "" {
		d.Field("ARN", arn)
	}
	if vaultType := vault.VaultType(); vaultType != "" {
		d.Field("Type", vaultType)
	}
	if state := vault.VaultState(); state != "" {
		d.Field("State", state)
	}

	// Recovery Points
	d.Section("Recovery Points")
	d.Field("Count", fmt.Sprintf("%d", vault.RecoveryPointCount()))

	// Encryption
	if keyArn := vault.EncryptionKeyArn(); keyArn != "" {
		d.Section("Encryption")
		d.Field("KMS Key ARN", keyArn)
		if keyType := vault.EncryptionKeyType(); keyType != "" {
			d.Field("Key Type", keyType)
		}
	}

	// Source Vault (for logically air-gapped vaults)
	if sourceArn := vault.SourceBackupVaultArn(); sourceArn != "" {
		d.Section("Source")
		d.Field("Source Vault ARN", sourceArn)
	}

	// Lock Configuration
	d.Section("Lock Configuration")
	if vault.Locked() {
		d.Field("Locked", "Yes")
		if lockDate := vault.LockDate(); lockDate != "" {
			d.Field("Lock Date", lockDate)
		}
	} else {
		d.Field("Locked", "No")
	}
	if minDays := vault.MinRetentionDays(); minDays > 0 {
		d.Field("Min Retention", fmt.Sprintf("%d days", minDays))
	}
	if maxDays := vault.MaxRetentionDays(); maxDays > 0 {
		d.Field("Max Retention", fmt.Sprintf("%d days", maxDays))
	}

	// Timestamps
	d.Section("Timestamps")
	if created := vault.CreatedAt(); created != "" {
		d.Field("Created", created)
	}

	// Creator
	if creatorId := vault.CreatorRequestId(); creatorId != "" {
		d.Section("Other")
		d.Field("Creator Request ID", creatorId)
	}

	return d.String()
}

// RenderSummary returns summary fields for the header panel
func (r *VaultRenderer) RenderSummary(resource dao.Resource) []render.SummaryField {
	vault, ok := resource.(*VaultResource)
	if !ok {
		return r.BaseRenderer.RenderSummary(resource)
	}

	fields := []render.SummaryField{
		{Label: "Name", Value: vault.VaultName()},
	}

	if arn := vault.VaultArn(); arn != "" {
		fields = append(fields, render.SummaryField{Label: "ARN", Value: arn})
	}

	fields = append(fields, render.SummaryField{
		Label: "Recovery Points",
		Value: fmt.Sprintf("%d", vault.RecoveryPointCount()),
	})

	if vault.Locked() {
		fields = append(fields, render.SummaryField{Label: "Locked", Value: "Yes"})
	}

	if created := vault.CreatedAt(); created != "" {
		fields = append(fields, render.SummaryField{Label: "Created", Value: created})
	}

	return fields
}

// Navigations returns navigation shortcuts
func (r *VaultRenderer) Navigations(resource dao.Resource) []render.Navigation {
	vault, ok := resource.(*VaultResource)
	if !ok {
		return nil
	}

	return []render.Navigation{
		{
			Key: "r", Label: "Recovery Points", Service: "backup", Resource: "recovery-points",
			FilterField: "VaultName", FilterValue: vault.VaultName(),
		},
	}
}
