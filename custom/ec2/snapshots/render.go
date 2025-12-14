package snapshots

import (
	"fmt"

	appaws "github.com/clawscli/claws/internal/aws"
	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/render"
)

// SnapshotRenderer renders EBS snapshots
type SnapshotRenderer struct {
	render.BaseRenderer
}

// NewSnapshotRenderer creates a new SnapshotRenderer
func NewSnapshotRenderer() render.Renderer {
	return &SnapshotRenderer{
		BaseRenderer: render.BaseRenderer{
			Service:  "ec2",
			Resource: "snapshots",
			Cols: []render.Column{
				{
					Name:  "NAME",
					Width: 25,
					Getter: func(r dao.Resource) string {
						return r.GetName()
					},
					Priority: 0,
				},
				{
					Name:  "SNAPSHOT ID",
					Width: 24,
					Getter: func(r dao.Resource) string {
						return r.GetID()
					},
					Priority: 1,
				},
				{
					Name:  "STATE",
					Width: 12,
					Getter: func(r dao.Resource) string {
						if v, ok := r.(*SnapshotResource); ok {
							return v.State()
						}
						return ""
					},
					Priority: 2,
				},
				{
					Name:  "PROGRESS",
					Width: 10,
					Getter: func(r dao.Resource) string {
						if v, ok := r.(*SnapshotResource); ok {
							return v.Progress()
						}
						return ""
					},
					Priority: 3,
				},
				{
					Name:  "SIZE",
					Width: 8,
					Getter: func(r dao.Resource) string {
						if v, ok := r.(*SnapshotResource); ok {
							return fmt.Sprintf("%dGiB", v.VolumeSize())
						}
						return ""
					},
					Priority: 4,
				},
				{
					Name:  "VOLUME ID",
					Width: 22,
					Getter: func(r dao.Resource) string {
						if v, ok := r.(*SnapshotResource); ok {
							return v.VolumeId()
						}
						return ""
					},
					Priority: 5,
				},
				{
					Name:  "ENC",
					Width: 4,
					Getter: func(r dao.Resource) string {
						if v, ok := r.(*SnapshotResource); ok {
							if v.Encrypted() {
								return "Yes"
							}
							return "No"
						}
						return ""
					},
					Priority: 6,
				},
				{
					Name:  "STARTED",
					Width: 12,
					Getter: func(r dao.Resource) string {
						if v, ok := r.(*SnapshotResource); ok {
							if v.Item.StartTime != nil {
								return render.FormatAge(*v.Item.StartTime)
							}
						}
						return ""
					},
					Priority: 7,
				},
				render.TagsColumn(25, 8),
			},
		},
	}
}

// RenderDetail renders detailed snapshot information
func (r *SnapshotRenderer) RenderDetail(resource dao.Resource) string {
	v, ok := resource.(*SnapshotResource)
	if !ok {
		return ""
	}

	d := render.NewDetailBuilder()

	d.Title("EBS Snapshot", v.GetName())

	// Basic Info
	d.Section("Basic Information")
	d.Field("Snapshot ID", v.GetID())
	d.FieldStyled("State", v.State(), render.StateColorer()(v.State()))
	d.Field("Progress", v.Progress())
	d.Field("Size", fmt.Sprintf("%d GiB", v.VolumeSize()))

	// Source
	d.Section("Source")
	d.Field("Volume ID", v.VolumeId())

	// Encryption
	d.Section("Encryption")
	if v.Encrypted() {
		d.Field("Encrypted", "Yes")
		d.FieldIf("KMS Key ID", v.Item.KmsKeyId)
	} else {
		d.Field("Encrypted", "No")
	}

	// Ownership
	d.Section("Ownership")
	d.Field("Owner ID", v.OwnerId())

	// Description
	if desc := v.Description(); desc != "" {
		d.Section("Description")
		d.DimIndent(desc)
	}

	// Timestamps
	d.Section("Timestamps")
	if v.Item.StartTime != nil {
		d.Field("Started", v.Item.StartTime.Format("2006-01-02 15:04:05"))
	}

	// Tags
	d.Tags(appaws.TagsToMap(v.Item.Tags))

	return d.String()
}

// RenderSummary returns summary fields for the header panel
func (r *SnapshotRenderer) RenderSummary(resource dao.Resource) []render.SummaryField {
	v, ok := resource.(*SnapshotResource)
	if !ok {
		return nil
	}

	stateStyle := render.StateColorer()(v.State())

	fields := []render.SummaryField{
		{Label: "Snapshot ID", Value: v.GetID()},
		{Label: "Name", Value: v.GetName()},
		{Label: "State", Value: v.State(), Style: stateStyle},
	}

	fields = append(fields, render.SummaryField{Label: "Progress", Value: v.Progress()})
	fields = append(fields, render.SummaryField{Label: "Size", Value: fmt.Sprintf("%d GiB", v.VolumeSize())})
	fields = append(fields, render.SummaryField{Label: "Volume ID", Value: v.VolumeId()})

	encValue := "No"
	if v.Encrypted() {
		encValue = "Yes"
	}
	fields = append(fields, render.SummaryField{Label: "Encrypted", Value: encValue})

	fields = append(fields, render.SummaryField{Label: "Owner", Value: v.OwnerId()})

	if v.Item.StartTime != nil {
		fields = append(fields, render.SummaryField{
			Label: "Started",
			Value: v.Item.StartTime.Format("2006-01-02 15:04"),
		})
	}

	return fields
}
