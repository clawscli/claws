package snapshots

import (
	"fmt"
	"time"

	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/render"
)

// Ensure SnapshotRenderer implements render.Navigator
var _ render.Navigator = (*SnapshotRenderer)(nil)

// SnapshotRenderer renders RDS snapshots with custom columns
type SnapshotRenderer struct {
	render.BaseRenderer
}

// NewSnapshotRenderer creates a new SnapshotRenderer
func NewSnapshotRenderer() render.Renderer {
	return &SnapshotRenderer{
		BaseRenderer: render.BaseRenderer{
			Service:  "rds",
			Resource: "snapshots",
			Cols: []render.Column{
				{
					Name:  "IDENTIFIER",
					Width: 40,
					Getter: func(r dao.Resource) string {
						return r.GetID()
					},
					Priority: 0,
				},
				{
					Name:  "STATUS",
					Width: 12,
					Getter: func(r dao.Resource) string {
						if sr, ok := r.(*SnapshotResource); ok {
							return sr.State()
						}
						return ""
					},
					Priority: 1,
				},
				{
					Name:  "INSTANCE",
					Width: 25,
					Getter: func(r dao.Resource) string {
						if sr, ok := r.(*SnapshotResource); ok {
							return sr.InstanceIdentifier()
						}
						return ""
					},
					Priority: 2,
				},
				{
					Name:  "TYPE",
					Width: 12,
					Getter: func(r dao.Resource) string {
						if sr, ok := r.(*SnapshotResource); ok {
							return sr.SnapshotType()
						}
						return ""
					},
					Priority: 3,
				},
				{
					Name:  "ENGINE",
					Width: 15,
					Getter: func(r dao.Resource) string {
						if sr, ok := r.(*SnapshotResource); ok {
							return sr.Engine()
						}
						return ""
					},
					Priority: 4,
				},
				{
					Name:  "SIZE",
					Width: 8,
					Getter: func(r dao.Resource) string {
						if sr, ok := r.(*SnapshotResource); ok {
							return fmt.Sprintf("%dGB", sr.AllocatedStorage())
						}
						return ""
					},
					Priority: 5,
				},
				{
					Name:  "AGE",
					Width: 8,
					Getter: func(r dao.Resource) string {
						if sr, ok := r.(*SnapshotResource); ok {
							if sr.Item.SnapshotCreateTime != nil {
								return render.FormatAge(*sr.Item.SnapshotCreateTime)
							}
						}
						return ""
					},
					Priority: 6,
				},
			},
		},
	}
}

// RenderDetail renders detailed snapshot information
func (r *SnapshotRenderer) RenderDetail(resource dao.Resource) string {
	sr, ok := resource.(*SnapshotResource)
	if !ok {
		return ""
	}

	d := render.NewDetailBuilder()

	d.Title("RDS Snapshot", sr.GetID())

	// Basic Info
	d.Section("Basic Information")
	d.Field("Identifier", sr.GetID())
	d.FieldStyled("Status", sr.State(), render.StateColorer()(sr.State()))
	d.Field("Snapshot Type", sr.SnapshotType())
	d.FieldIf("Source DB Instance", sr.Item.DBInstanceIdentifier)
	if sr.Item.SnapshotCreateTime != nil {
		d.Field("Created", sr.Item.SnapshotCreateTime.Format(time.RFC3339))
		d.Field("Age", render.FormatAge(*sr.Item.SnapshotCreateTime))
	}

	// Engine
	d.Section("Engine")
	d.Field("Engine", sr.Engine())
	d.Field("Engine Version", sr.EngineVersion())
	d.FieldIf("License Model", sr.Item.LicenseModel)

	// Storage
	d.Section("Storage")
	d.Field("Allocated Storage", fmt.Sprintf("%d GB", sr.AllocatedStorage()))
	d.FieldIf("Storage Type", sr.Item.StorageType)
	d.Field("Encrypted", fmt.Sprintf("%v", sr.Item.Encrypted))
	d.FieldIf("KMS Key ID", sr.Item.KmsKeyId)
	if sr.Item.Iops != nil {
		d.Field("IOPS", fmt.Sprintf("%d", *sr.Item.Iops))
	}

	// Network
	d.Section("Network")
	d.FieldIf("VPC ID", sr.Item.VpcId)
	d.FieldIf("Availability Zone", sr.Item.AvailabilityZone)
	if sr.Item.Port != nil && *sr.Item.Port != 0 {
		d.Field("Port", fmt.Sprintf("%d", *sr.Item.Port))
	}

	// Options
	d.Section("Configuration")
	d.FieldIf("Master Username", sr.Item.MasterUsername)
	d.FieldIf("Option Group", sr.Item.OptionGroupName)
	d.FieldIf("DB Parameter Group", sr.Item.DBInstanceIdentifier)

	// Tags
	d.Tags(rdsTagsToMap(sr.Item.TagList))

	return d.String()
}

// RenderSummary returns summary fields for the header panel
func (r *SnapshotRenderer) RenderSummary(resource dao.Resource) []render.SummaryField {
	sr, ok := resource.(*SnapshotResource)
	if !ok {
		return nil
	}

	stateStyle := render.StateColorer()(sr.State())

	fields := []render.SummaryField{
		{Label: "Identifier", Value: sr.GetID()},
		{Label: "Status", Value: sr.State(), Style: stateStyle},
		{Label: "Type", Value: sr.SnapshotType()},
	}

	if sr.InstanceIdentifier() != "" {
		fields = append(fields, render.SummaryField{Label: "Source Instance", Value: sr.InstanceIdentifier()})
	}

	fields = append(fields, render.SummaryField{Label: "Engine", Value: fmt.Sprintf("%s %s", sr.Engine(), sr.EngineVersion())})
	fields = append(fields, render.SummaryField{Label: "Storage", Value: fmt.Sprintf("%d GB", sr.AllocatedStorage())})

	if sr.Item.SnapshotCreateTime != nil {
		fields = append(fields, render.SummaryField{
			Label: "Created",
			Value: sr.Item.SnapshotCreateTime.Format("2006-01-02 15:04") + " (" + render.FormatAge(*sr.Item.SnapshotCreateTime) + ")",
		})
	}

	return fields
}

// Navigations returns navigation shortcuts for RDS snapshots
func (r *SnapshotRenderer) Navigations(resource dao.Resource) []render.Navigation {
	sr, ok := resource.(*SnapshotResource)
	if !ok {
		return nil
	}

	var navs []render.Navigation

	// Source instance navigation
	if sr.InstanceIdentifier() != "" {
		navs = append(navs, render.Navigation{
			Key: "i", Label: "Instance", Service: "rds", Resource: "instances",
			FilterField: "DBInstanceIdentifier", FilterValue: sr.InstanceIdentifier(),
		})
	}

	return navs
}
