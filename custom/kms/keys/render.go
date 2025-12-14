package keys

import (
	"strings"

	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/render"
)

// KeyRenderer renders KMS keys
// Ensure KeyRenderer implements render.Navigator
var _ render.Navigator = (*KeyRenderer)(nil)

type KeyRenderer struct {
	render.BaseRenderer
}

// NewKeyRenderer creates a new KeyRenderer
func NewKeyRenderer() *KeyRenderer {
	return &KeyRenderer{
		BaseRenderer: render.BaseRenderer{
			Service:  "kms",
			Resource: "keys",
			Cols: []render.Column{
				{Name: "KEY ID", Width: 38, Getter: getKeyId},
				{Name: "DESCRIPTION", Width: 30, Getter: getDescription},
				{Name: "STATE", Width: 12, Getter: getState},
				{Name: "USAGE", Width: 18, Getter: getUsage},
				{Name: "MANAGER", Width: 10, Getter: getManager},
				{Name: "AGE", Width: 8, Getter: getAge},
			},
		},
	}
}

func getKeyId(r dao.Resource) string {
	if key, ok := r.(*KeyResource); ok {
		return key.KeyId()
	}
	return ""
}

func getDescription(r dao.Resource) string {
	if key, ok := r.(*KeyResource); ok {
		desc := key.Description()
		if len(desc) > 28 {
			return desc[:28] + "..."
		}
		return desc
	}
	return ""
}

func getState(r dao.Resource) string {
	if key, ok := r.(*KeyResource); ok {
		return key.KeyState()
	}
	return ""
}

func getUsage(r dao.Resource) string {
	if key, ok := r.(*KeyResource); ok {
		return key.KeyUsage()
	}
	return ""
}

func getManager(r dao.Resource) string {
	if key, ok := r.(*KeyResource); ok {
		return key.KeyManager()
	}
	return ""
}

func getAge(r dao.Resource) string {
	if key, ok := r.(*KeyResource); ok {
		if key.Item.CreationDate != nil {
			return render.FormatAge(*key.Item.CreationDate)
		}
	}
	return "-"
}

// RenderDetail renders detailed key information
func (r *KeyRenderer) RenderDetail(resource dao.Resource) string {
	key, ok := resource.(*KeyResource)
	if !ok {
		return ""
	}

	d := render.NewDetailBuilder()

	d.Title("KMS Key", key.KeyId())

	// Basic Info
	d.Section("Basic Information")
	d.Field("Key ID", key.KeyId())
	d.Field("ARN", key.GetARN())
	if account := key.AWSAccountId(); account != "" {
		d.Field("Account ID", account)
	}

	if desc := key.Description(); desc != "" {
		d.Field("Description", desc)
	}

	// Key Configuration
	d.Section("Key Configuration")
	d.Field("Key State", key.KeyState())
	d.Field("Key Spec", key.KeySpec())
	d.Field("Key Usage", key.KeyUsage())
	d.Field("Origin", key.Origin())
	d.Field("Key Manager", key.KeyManager())

	if key.IsEnabled() {
		d.Field("Enabled", "Yes")
	} else {
		d.Field("Enabled", "No")
	}

	if key.IsMultiRegion() {
		d.Field("Multi-Region", "Yes")
	} else {
		d.Field("Multi-Region", "No")
	}

	// Algorithms
	if encAlgs := key.EncryptionAlgorithms(); len(encAlgs) > 0 {
		d.Section("Encryption Algorithms")
		d.Field("Supported", strings.Join(encAlgs, ", "))
	}

	if signAlgs := key.SigningAlgorithms(); len(signAlgs) > 0 {
		d.Section("Signing Algorithms")
		d.Field("Supported", strings.Join(signAlgs, ", "))
	}

	if macAlgs := key.MacAlgorithms(); len(macAlgs) > 0 {
		d.Section("MAC Algorithms")
		d.Field("Supported", strings.Join(macAlgs, ", "))
	}

	// Key Material Expiration
	if expModel := key.ExpirationModel(); expModel != "" {
		d.Section("Key Material")
		d.Field("Expiration Model", expModel)
		if validTo := key.ValidTo(); validTo != "" {
			d.Field("Valid Until", validTo)
		}
	}

	// Timestamps
	d.Section("Timestamps")
	if created := key.CreationDate(); created != "" {
		d.Field("Created", created)
	}

	// Deletion Info
	if deletion := key.DeletionDate(); deletion != "" {
		d.Section("Deletion")
		d.Field("Scheduled Deletion", deletion)
	}

	return d.String()
}

// RenderSummary returns summary fields for the header panel
func (r *KeyRenderer) RenderSummary(resource dao.Resource) []render.SummaryField {
	key, ok := resource.(*KeyResource)
	if !ok {
		return r.BaseRenderer.RenderSummary(resource)
	}

	fields := []render.SummaryField{
		{Label: "Key ID", Value: key.KeyId()},
		{Label: "ARN", Value: key.GetARN()},
	}

	if desc := key.Description(); desc != "" {
		fields = append(fields, render.SummaryField{Label: "Description", Value: desc})
	}

	fields = append(fields,
		render.SummaryField{Label: "State", Value: key.KeyState()},
		render.SummaryField{Label: "Usage", Value: key.KeyUsage()},
		render.SummaryField{Label: "Manager", Value: key.KeyManager()},
	)

	if created := key.CreationDate(); created != "" {
		fields = append(fields, render.SummaryField{Label: "Created", Value: created})
	}

	return fields
}

// Navigations returns navigation shortcuts
func (r *KeyRenderer) Navigations(resource dao.Resource) []render.Navigation {
	// No navigations for now
	return nil
}
