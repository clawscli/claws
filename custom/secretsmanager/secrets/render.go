package secrets

import (
	"fmt"
	"time"

	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/render"
	"github.com/clawscli/claws/internal/ui"
)

// SecretRenderer renders Secrets Manager secrets
// Ensure SecretRenderer implements render.Navigator
var _ render.Navigator = (*SecretRenderer)(nil)

type SecretRenderer struct {
	render.BaseRenderer
}

// NewSecretRenderer creates a new SecretRenderer
func NewSecretRenderer() render.Renderer {
	return &SecretRenderer{
		BaseRenderer: render.BaseRenderer{
			Service:  "secretsmanager",
			Resource: "secrets",
			Cols: []render.Column{
				{Name: "NAME", Width: 40, Getter: func(r dao.Resource) string { return r.GetName() }},
				{Name: "DESCRIPTION", Width: 35, Getter: getDescription},
				{Name: "LAST ACCESSED", Width: 14, Getter: getLastAccessed},
				{Name: "AGE", Width: 10, Getter: getAge},
			},
		},
	}
}

func getDescription(r dao.Resource) string {
	if secret, ok := r.(*SecretResource); ok {
		desc := secret.Description()
		if len(desc) > 32 {
			return desc[:32] + "..."
		}
		return desc
	}
	return ""
}

func getLastAccessed(r dao.Resource) string {
	if secret, ok := r.(*SecretResource); ok {
		return secret.LastAccessedDate()
	}
	return "-"
}

func getAge(r dao.Resource) string {
	if secret, ok := r.(*SecretResource); ok {
		if secret.Item.CreatedDate != nil {
			return render.FormatAge(*secret.Item.CreatedDate)
		}
	}
	return "-"
}

// RenderDetail renders detailed secret information
func (r *SecretRenderer) RenderDetail(resource dao.Resource) string {
	secret, ok := resource.(*SecretResource)
	if !ok {
		return ""
	}

	d := render.NewDetailBuilder()

	d.Title("Secret", secret.GetName())

	// Basic Info
	d.Section("Basic Information")
	d.Field("Name", secret.GetName())
	d.Field("ARN", secret.GetARN())

	if desc := secret.Description(); desc != "" {
		d.Field("Description", desc)
	}

	if secret.PrimaryRegion != "" {
		d.Field("Primary Region", secret.PrimaryRegion)
	}

	// Deletion Status
	if secret.DeletedDate != nil {
		d.Section("Deletion")
		d.FieldStyled("Status", "SCHEDULED FOR DELETION", ui.DangerStyle())
		d.Field("Deletion Date", *secret.DeletedDate)
	}

	// Encryption
	d.Section("Encryption")
	if secret.KmsKeyId != "" {
		d.Field("KMS Key", secret.KmsKeyId)
	} else {
		d.Field("KMS Key", "aws/secretsmanager (default)")
	}

	// Rotation
	d.Section("Rotation")
	if secret.RotationEnabled {
		d.FieldStyled("Rotation", "Enabled", ui.SuccessStyle())
		if secret.RotationLambdaARN != "" {
			d.Field("Lambda ARN", secret.RotationLambdaARN)
		}
		if secret.RotationRules != nil {
			if secret.RotationRules.AutomaticallyAfterDays != nil {
				d.Field("Rotate Every", fmt.Sprintf("%d days", *secret.RotationRules.AutomaticallyAfterDays))
			}
			if secret.RotationRules.ScheduleExpression != nil {
				d.Field("Schedule", *secret.RotationRules.ScheduleExpression)
			}
		}
	} else {
		d.Field("Rotation", "Disabled")
	}

	// Version Info
	d.Section("Version Information")
	d.Field("Version Count", fmt.Sprintf("%d", secret.VersionCount()))
	if versionId := secret.CurrentVersionId(); versionId != "" {
		// Truncate long version IDs
		if len(versionId) > 36 {
			versionId = versionId[:36] + "..."
		}
		d.Field("Current Version", versionId)
	}

	// Timestamps
	d.Section("Timestamps")
	if created := secret.CreatedDate(); created != "" {
		d.Field("Created", created)
	}
	if changed := secret.LastChangedDate(); changed != "" {
		d.Field("Last Changed", changed)
	}
	if accessed := secret.LastAccessedDate(); accessed != "" {
		d.Field("Last Accessed", accessed)
	}
	if secret.Item.CreatedDate != nil {
		d.Field("Age", time.Since(*secret.Item.CreatedDate).Truncate(time.Second).String())
	}

	// Tags
	d.Tags(secret.GetTags())

	return d.String()
}

// RenderSummary returns summary fields for the header panel
func (r *SecretRenderer) RenderSummary(resource dao.Resource) []render.SummaryField {
	secret, ok := resource.(*SecretResource)
	if !ok {
		return r.BaseRenderer.RenderSummary(resource)
	}

	fields := []render.SummaryField{
		{Label: "Name", Value: secret.GetName()},
		{Label: "ARN", Value: secret.GetARN()},
	}

	if desc := secret.Description(); desc != "" {
		fields = append(fields, render.SummaryField{Label: "Description", Value: desc})
	}

	fields = append(fields, render.SummaryField{Label: "Versions", Value: fmt.Sprintf("%d", secret.VersionCount())})

	if created := secret.CreatedDate(); created != "" {
		fields = append(fields, render.SummaryField{Label: "Created", Value: created})
	}

	if accessed := secret.LastAccessedDate(); accessed != "" {
		fields = append(fields, render.SummaryField{Label: "Last Accessed", Value: accessed})
	}

	return fields
}

// Navigations returns navigation shortcuts
func (r *SecretRenderer) Navigations(resource dao.Resource) []render.Navigation {
	// No navigations for now
	return nil
}
