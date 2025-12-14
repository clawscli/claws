package volumes

import (
	"fmt"

	appaws "github.com/clawscli/claws/internal/aws"
	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/render"
)

// VolumeRenderer renders EBS volumes
type VolumeRenderer struct {
	render.BaseRenderer
}

// NewVolumeRenderer creates a new VolumeRenderer
func NewVolumeRenderer() render.Renderer {
	return &VolumeRenderer{
		BaseRenderer: render.BaseRenderer{
			Service:  "ec2",
			Resource: "volumes",
			Cols: []render.Column{
				{
					Name:  "NAME",
					Width: 20,
					Getter: func(r dao.Resource) string {
						return r.GetName()
					},
					Priority: 0,
				},
				{
					Name:  "ID",
					Width: 22,
					Getter: func(r dao.Resource) string {
						return r.GetID()
					},
					Priority: 1,
				},
				{
					Name:  "STATE",
					Width: 12,
					Getter: func(r dao.Resource) string {
						if v, ok := r.(*VolumeResource); ok {
							return v.State()
						}
						return ""
					},
					Priority: 2,
				},
				{
					Name:  "SIZE",
					Width: 8,
					Getter: func(r dao.Resource) string {
						if v, ok := r.(*VolumeResource); ok {
							return fmt.Sprintf("%dGiB", v.Size())
						}
						return ""
					},
					Priority: 3,
				},
				{
					Name:  "TYPE",
					Width: 8,
					Getter: func(r dao.Resource) string {
						if v, ok := r.(*VolumeResource); ok {
							return v.VolumeType()
						}
						return ""
					},
					Priority: 4,
				},
				{
					Name:  "IOPS",
					Width: 6,
					Getter: func(r dao.Resource) string {
						if v, ok := r.(*VolumeResource); ok {
							iops := v.IOPS()
							if iops > 0 {
								return fmt.Sprintf("%d", iops)
							}
						}
						return "-"
					},
					Priority: 5,
				},
				{
					Name:  "AZ",
					Width: 15,
					Getter: func(r dao.Resource) string {
						if v, ok := r.(*VolumeResource); ok {
							return v.AZ()
						}
						return ""
					},
					Priority: 6,
				},
				{
					Name:  "ATTACHED TO",
					Width: 20,
					Getter: func(r dao.Resource) string {
						if v, ok := r.(*VolumeResource); ok {
							return v.AttachedInstance()
						}
						return ""
					},
					Priority: 7,
				},
				{
					Name:  "ENC",
					Width: 4,
					Getter: func(r dao.Resource) string {
						if v, ok := r.(*VolumeResource); ok {
							if v.Encrypted() {
								return "Yes"
							}
							return "No"
						}
						return ""
					},
					Priority: 8,
				},
				render.TagsColumn(25, 9),
			},
		},
	}
}

// RenderDetail renders detailed volume information
func (r *VolumeRenderer) RenderDetail(resource dao.Resource) string {
	v, ok := resource.(*VolumeResource)
	if !ok {
		return ""
	}

	d := render.NewDetailBuilder()

	d.Title("EBS Volume", v.GetName())

	// Basic Info
	d.Section("Basic Information")
	d.Field("Volume ID", v.GetID())
	d.FieldStyled("State", v.State(), render.StateColorer()(v.State()))
	d.Field("Size", fmt.Sprintf("%d GiB", v.Size()))
	d.Field("Volume Type", v.VolumeType())
	d.Field("Availability Zone", v.AZ())

	if iops := v.IOPS(); iops > 0 {
		d.Field("IOPS", fmt.Sprintf("%d", iops))
	}
	if v.Item.Throughput != nil {
		d.Field("Throughput", fmt.Sprintf("%d MiB/s", *v.Item.Throughput))
	}

	// Encryption
	d.Section("Encryption")
	if v.Encrypted() {
		d.Field("Encrypted", "Yes")
		d.FieldIf("KMS Key ID", v.Item.KmsKeyId)
	} else {
		d.Field("Encrypted", "No")
	}

	// Attachments
	d.Section("Attachments")
	if len(v.Item.Attachments) > 0 {
		for _, att := range v.Item.Attachments {
			instanceID := appaws.Str(att.InstanceId)
			if instanceID == "" {
				instanceID = "-"
			}
			device := appaws.Str(att.Device)
			if device == "" {
				device = "-"
			}
			deleteOnTerm := "No"
			if att.DeleteOnTermination != nil && *att.DeleteOnTermination {
				deleteOnTerm = "Yes"
			}

			d.Field("Instance", instanceID)
			d.Field("Device", device)
			d.Field("State", string(att.State))
			d.Field("Delete on Term", deleteOnTerm)
			if att.AttachTime != nil {
				d.Field("Attach Time", att.AttachTime.Format("2006-01-02 15:04:05"))
			}
		}
	} else {
		d.DimIndent("(not attached)")
	}

	// Snapshot info
	if v.Item.SnapshotId != nil && *v.Item.SnapshotId != "" {
		d.Section("Source")
		d.Field("Snapshot ID", *v.Item.SnapshotId)
	}

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
func (r *VolumeRenderer) RenderSummary(resource dao.Resource) []render.SummaryField {
	v, ok := resource.(*VolumeResource)
	if !ok {
		return nil
	}

	stateStyle := render.StateColorer()(v.State())

	// Row 1: ID, Name, State
	fields := []render.SummaryField{
		{Label: "ID", Value: v.GetID()},
		{Label: "Name", Value: v.GetName()},
		{Label: "State", Value: v.State(), Style: stateStyle},
	}

	// Row 2: Size, Type, IOPS
	fields = append(fields, render.SummaryField{Label: "Size", Value: fmt.Sprintf("%d GiB", v.Size())})
	fields = append(fields, render.SummaryField{Label: "Type", Value: v.VolumeType()})
	if iops := v.IOPS(); iops > 0 {
		fields = append(fields, render.SummaryField{Label: "IOPS", Value: fmt.Sprintf("%d", iops)})
	}

	// Row 3: AZ, Encrypted
	fields = append(fields, render.SummaryField{Label: "AZ", Value: v.AZ()})
	encValue := "No"
	if v.Encrypted() {
		encValue = "Yes"
	}
	fields = append(fields, render.SummaryField{Label: "Encrypted", Value: encValue})

	// Row 4: Attachment info
	if attached := v.AttachedInstance(); attached != "" {
		fields = append(fields, render.SummaryField{Label: "Attached To", Value: attached})
		// Add device name if available
		if len(v.Item.Attachments) > 0 && v.Item.Attachments[0].Device != nil {
			fields = append(fields, render.SummaryField{Label: "Device", Value: *v.Item.Attachments[0].Device})
		}
	}

	// Row 5: Creation time
	if v.Item.CreateTime != nil {
		fields = append(fields, render.SummaryField{
			Label: "Created",
			Value: v.Item.CreateTime.Format("2006-01-02 15:04"),
		})
	}

	return fields
}
