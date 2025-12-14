package images

import (
	"fmt"
	"time"

	appaws "github.com/clawscli/claws/internal/aws"
	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/render"
)

// ImageRenderer renders EC2 AMIs
type ImageRenderer struct {
	render.BaseRenderer
}

// NewImageRenderer creates a new ImageRenderer
func NewImageRenderer() render.Renderer {
	return &ImageRenderer{
		BaseRenderer: render.BaseRenderer{
			Service:  "ec2",
			Resource: "images",
			Cols: []render.Column{
				{
					Name:  "NAME",
					Width: 35,
					Getter: func(r dao.Resource) string {
						return r.GetName()
					},
					Priority: 0,
				},
				{
					Name:  "IMAGE ID",
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
						if v, ok := r.(*ImageResource); ok {
							return v.State()
						}
						return ""
					},
					Priority: 2,
				},
				{
					Name:  "TYPE",
					Width: 10,
					Getter: func(r dao.Resource) string {
						if v, ok := r.(*ImageResource); ok {
							return v.ImageType()
						}
						return ""
					},
					Priority: 3,
				},
				{
					Name:  "ARCH",
					Width: 8,
					Getter: func(r dao.Resource) string {
						if v, ok := r.(*ImageResource); ok {
							return v.Architecture()
						}
						return ""
					},
					Priority: 4,
				},
				{
					Name:  "PLATFORM",
					Width: 20,
					Getter: func(r dao.Resource) string {
						if v, ok := r.(*ImageResource); ok {
							return v.Platform()
						}
						return ""
					},
					Priority: 5,
				},
				{
					Name:  "ROOT",
					Width: 6,
					Getter: func(r dao.Resource) string {
						if v, ok := r.(*ImageResource); ok {
							return v.RootDeviceType()
						}
						return ""
					},
					Priority: 6,
				},
				{
					Name:  "CREATED",
					Width: 12,
					Getter: func(r dao.Resource) string {
						if v, ok := r.(*ImageResource); ok {
							dateStr := v.CreationDate()
							if dateStr == "" {
								return ""
							}
							t, err := time.Parse(time.RFC3339, dateStr)
							if err != nil {
								return dateStr
							}
							return render.FormatAge(t)
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

// RenderDetail renders detailed AMI information
func (r *ImageRenderer) RenderDetail(resource dao.Resource) string {
	v, ok := resource.(*ImageResource)
	if !ok {
		return ""
	}

	d := render.NewDetailBuilder()

	d.Title("EC2 AMI", v.GetName())

	// Basic Info
	d.Section("Basic Information")
	d.Field("Image ID", v.GetID())
	d.Field("Name", v.GetName())
	d.FieldStyled("State", v.State(), render.StateColorer()(v.State()))
	d.Field("Image Type", v.ImageType())
	d.Field("Architecture", v.Architecture())
	d.Field("Virtualization", v.VirtualizationType())

	// Platform
	d.Section("Platform")
	d.Field("Platform Details", v.Platform())
	d.Field("Root Device Type", v.RootDeviceType())
	d.Field("Root Device Name", v.RootDeviceName())

	// Ownership
	d.Section("Ownership")
	d.Field("Owner ID", v.OwnerId())
	publicStr := "No"
	if v.Public() {
		publicStr = "Yes"
	}
	d.Field("Public", publicStr)

	// Description
	if desc := v.Description(); desc != "" {
		d.Section("Description")
		d.DimIndent(desc)
	}

	// Block Device Mappings
	d.Section("Block Device Mappings")
	if len(v.Item.BlockDeviceMappings) > 0 {
		for _, bdm := range v.Item.BlockDeviceMappings {
			deviceName := appaws.Str(bdm.DeviceName)
			if deviceName == "" {
				deviceName = "-"
			}
			d.Field("Device", deviceName)
			if bdm.Ebs != nil {
				if bdm.Ebs.SnapshotId != nil {
					d.Field("  Snapshot", *bdm.Ebs.SnapshotId)
				}
				if bdm.Ebs.VolumeSize != nil {
					d.Field("  Size", fmt.Sprintf("%d GiB", *bdm.Ebs.VolumeSize))
				}
				d.Field("  Volume Type", string(bdm.Ebs.VolumeType))
				if bdm.Ebs.Encrypted != nil {
					encStr := "No"
					if *bdm.Ebs.Encrypted {
						encStr = "Yes"
					}
					d.Field("  Encrypted", encStr)
				}
			}
		}
	} else {
		d.DimIndent("(none)")
	}

	// Timestamps
	d.Section("Timestamps")
	if dateStr := v.CreationDate(); dateStr != "" {
		d.Field("Created", dateStr)
	}

	// Tags
	d.Tags(appaws.TagsToMap(v.Item.Tags))

	return d.String()
}

// RenderSummary returns summary fields for the header panel
func (r *ImageRenderer) RenderSummary(resource dao.Resource) []render.SummaryField {
	v, ok := resource.(*ImageResource)
	if !ok {
		return nil
	}

	stateStyle := render.StateColorer()(v.State())

	fields := []render.SummaryField{
		{Label: "Image ID", Value: v.GetID()},
		{Label: "Name", Value: v.GetName()},
		{Label: "State", Value: v.State(), Style: stateStyle},
	}

	fields = append(fields, render.SummaryField{Label: "Type", Value: v.ImageType()})
	fields = append(fields, render.SummaryField{Label: "Arch", Value: v.Architecture()})
	fields = append(fields, render.SummaryField{Label: "Platform", Value: v.Platform()})

	fields = append(fields, render.SummaryField{Label: "Root Device", Value: v.RootDeviceType()})
	fields = append(fields, render.SummaryField{Label: "Virtualization", Value: v.VirtualizationType()})

	publicStr := "No"
	if v.Public() {
		publicStr = "Yes"
	}
	fields = append(fields, render.SummaryField{Label: "Public", Value: publicStr})

	if dateStr := v.CreationDate(); dateStr != "" {
		fields = append(fields, render.SummaryField{Label: "Created", Value: dateStr})
	}

	return fields
}
