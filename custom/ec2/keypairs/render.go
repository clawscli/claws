package keypairs

import (
	appaws "github.com/clawscli/claws/internal/aws"
	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/render"
)

// KeyPairRenderer renders EC2 Key Pairs
type KeyPairRenderer struct {
	render.BaseRenderer
}

// NewKeyPairRenderer creates a new KeyPairRenderer
func NewKeyPairRenderer() render.Renderer {
	return &KeyPairRenderer{
		BaseRenderer: render.BaseRenderer{
			Service:  "ec2",
			Resource: "key-pairs",
			Cols: []render.Column{
				{
					Name:  "NAME",
					Width: 30,
					Getter: func(r dao.Resource) string {
						return r.GetName()
					},
					Priority: 0,
				},
				{
					Name:  "KEY PAIR ID",
					Width: 24,
					Getter: func(r dao.Resource) string {
						return r.GetID()
					},
					Priority: 1,
				},
				{
					Name:  "TYPE",
					Width: 10,
					Getter: func(r dao.Resource) string {
						if v, ok := r.(*KeyPairResource); ok {
							return v.KeyType()
						}
						return ""
					},
					Priority: 2,
				},
				{
					Name:  "FINGERPRINT",
					Width: 50,
					Getter: func(r dao.Resource) string {
						if v, ok := r.(*KeyPairResource); ok {
							return v.Fingerprint()
						}
						return ""
					},
					Priority: 3,
				},
				{
					Name:  "CREATED",
					Width: 12,
					Getter: func(r dao.Resource) string {
						if v, ok := r.(*KeyPairResource); ok {
							if v.Item.CreateTime != nil {
								return render.FormatAge(*v.Item.CreateTime)
							}
						}
						return ""
					},
					Priority: 4,
				},
				render.TagsColumn(25, 5),
			},
		},
	}
}

// RenderDetail renders detailed Key Pair information
func (r *KeyPairRenderer) RenderDetail(resource dao.Resource) string {
	v, ok := resource.(*KeyPairResource)
	if !ok {
		return ""
	}

	d := render.NewDetailBuilder()

	d.Title("EC2 Key Pair", v.GetName())

	// Basic Info
	d.Section("Basic Information")
	d.Field("Key Name", v.KeyName())
	d.Field("Key Pair ID", v.GetID())
	d.Field("Key Type", v.KeyType())
	d.Field("Fingerprint", v.Fingerprint())

	// Timestamps
	d.Section("Timestamps")
	if v.Item.CreateTime != nil {
		d.Field("Created", v.Item.CreateTime.Format("2006-01-02 15:04:05"))
	}

	// Tags
	d.Tags(appaws.TagsToMap(v.Item.Tags))

	return d.String()
}

// RenderSummary returns summary fields for the header panel
func (r *KeyPairRenderer) RenderSummary(resource dao.Resource) []render.SummaryField {
	v, ok := resource.(*KeyPairResource)
	if !ok {
		return nil
	}

	fields := []render.SummaryField{
		{Label: "Key Name", Value: v.KeyName()},
		{Label: "Key Pair ID", Value: v.GetID()},
		{Label: "Key Type", Value: v.KeyType()},
	}

	fields = append(fields, render.SummaryField{Label: "Fingerprint", Value: v.Fingerprint()})

	if v.Item.CreateTime != nil {
		fields = append(fields, render.SummaryField{
			Label: "Created",
			Value: v.Item.CreateTime.Format("2006-01-02 15:04"),
		})
	}

	return fields
}
