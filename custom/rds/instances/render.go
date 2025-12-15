package instances

import (
	"fmt"
	"time"

	appaws "github.com/clawscli/claws/internal/aws"
	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/render"
)

// Ensure InstanceRenderer implements render.Navigator
var _ render.Navigator = (*InstanceRenderer)(nil)

// InstanceRenderer renders RDS instances with custom columns
type InstanceRenderer struct {
	render.BaseRenderer
}

// NewInstanceRenderer creates a new InstanceRenderer
func NewInstanceRenderer() render.Renderer {
	return &InstanceRenderer{
		BaseRenderer: render.BaseRenderer{
			Service:  "rds",
			Resource: "instances",
			Cols: []render.Column{
				{
					Name:  "IDENTIFIER",
					Width: 28,
					Getter: func(r dao.Resource) string {
						return r.GetID()
					},
					Priority: 0,
				},
				{
					Name:  "STATUS",
					Width: 12,
					Getter: func(r dao.Resource) string {
						if ir, ok := r.(*InstanceResource); ok {
							return ir.State()
						}
						return ""
					},
					Priority: 1,
				},
				{
					Name:  "ENGINE",
					Width: 18,
					Getter: func(r dao.Resource) string {
						if ir, ok := r.(*InstanceResource); ok {
							return ir.Engine()
						}
						return ""
					},
					Priority: 2,
				},
				{
					Name:  "CLASS",
					Width: 16,
					Getter: func(r dao.Resource) string {
						if ir, ok := r.(*InstanceResource); ok {
							return ir.InstanceClass()
						}
						return ""
					},
					Priority: 3,
				},
				{
					Name:  "AZ",
					Width: 14,
					Getter: func(r dao.Resource) string {
						if ir, ok := r.(*InstanceResource); ok {
							return ir.AZ()
						}
						return ""
					},
					Priority: 4,
				},
				{
					Name:  "STORAGE",
					Width: 10,
					Getter: func(r dao.Resource) string {
						if ir, ok := r.(*InstanceResource); ok {
							return fmt.Sprintf("%dGB", ir.AllocatedStorage())
						}
						return ""
					},
					Priority: 5,
				},
				{
					Name:  "MULTI-AZ",
					Width: 9,
					Getter: func(r dao.Resource) string {
						if ir, ok := r.(*InstanceResource); ok {
							if ir.MultiAZ() {
								return "Yes"
							}
							return "No"
						}
						return ""
					},
					Priority: 6,
				},
				{
					Name:  "AGE",
					Width: 8,
					Getter: func(r dao.Resource) string {
						if ir, ok := r.(*InstanceResource); ok {
							if ir.Item.InstanceCreateTime != nil {
								return render.FormatAge(*ir.Item.InstanceCreateTime)
							}
						}
						return ""
					},
					Priority: 7,
				},
			},
		},
	}
}

// RenderDetail renders detailed instance information
func (r *InstanceRenderer) RenderDetail(resource dao.Resource) string {
	ir, ok := resource.(*InstanceResource)
	if !ok {
		return ""
	}

	d := render.NewDetailBuilder()
	styles := d.Styles()

	d.Title("RDS Instance", ir.GetID())

	// Basic Info
	d.Section("Basic Information")
	d.Field("Identifier", ir.GetID())
	d.FieldStyled("Status", ir.State(), render.StateColorer()(ir.State()))
	d.Field("Engine", ir.Engine())
	d.Field("Engine Version", ir.EngineVersion())
	d.Field("Instance Class", ir.InstanceClass())
	d.FieldIf("License Model", ir.Item.LicenseModel)
	if ir.Item.InstanceCreateTime != nil {
		d.Field("Created", ir.Item.InstanceCreateTime.Format(time.RFC3339))
		d.Field("Age", render.FormatAge(*ir.Item.InstanceCreateTime))
	}

	// Connectivity
	d.Section("Connectivity")
	if ir.Endpoint() != "" {
		d.Field("Endpoint", ir.Endpoint())
		d.Field("Port", fmt.Sprintf("%d", ir.Port()))
	}
	d.FieldIf("VPC ID", ir.Item.DBSubnetGroup.VpcId)
	if ir.Item.DBSubnetGroup != nil && ir.Item.DBSubnetGroup.DBSubnetGroupName != nil {
		d.Field("Subnet Group", *ir.Item.DBSubnetGroup.DBSubnetGroupName)
	}
	d.Field("Availability Zone", ir.AZ())
	if ir.MultiAZ() {
		d.Field("Multi-AZ", "Yes")
		d.FieldIf("Secondary AZ", ir.Item.SecondaryAvailabilityZone)
	} else {
		d.Field("Multi-AZ", "No")
	}
	d.Field("Publicly Accessible", fmt.Sprintf("%v", ir.Item.PubliclyAccessible))

	// Security Groups
	if len(ir.Item.VpcSecurityGroups) > 0 {
		d.Section("VPC Security Groups")
		for _, sg := range ir.Item.VpcSecurityGroups {
			status := appaws.Str(sg.Status)
			id := appaws.Str(sg.VpcSecurityGroupId)
			d.Line("  " + styles.Value.Render(id) + styles.Dim.Render(" ("+status+")"))
		}
	}

	// Storage
	d.Section("Storage")
	d.Field("Storage Type", ir.StorageType())
	d.Field("Allocated Storage", fmt.Sprintf("%d GB", ir.AllocatedStorage()))
	if ir.Item.MaxAllocatedStorage != nil {
		d.Field("Max Allocated Storage", fmt.Sprintf("%d GB", *ir.Item.MaxAllocatedStorage))
	}
	if ir.Item.Iops != nil {
		d.Field("IOPS", fmt.Sprintf("%d", *ir.Item.Iops))
	}
	d.Field("Storage Encrypted", fmt.Sprintf("%v", ir.Item.StorageEncrypted))
	d.FieldIf("KMS Key ID", ir.Item.KmsKeyId)

	// Database
	d.Section("Database")
	d.FieldIf("DB Name", ir.Item.DBName)
	d.FieldIf("Master Username", ir.Item.MasterUsername)
	d.FieldIf("Character Set", ir.Item.CharacterSetName)

	// Backup
	d.Section("Backup")
	d.Field("Backup Retention Period", fmt.Sprintf("%d days", ir.Item.BackupRetentionPeriod))
	d.FieldIf("Preferred Backup Window", ir.Item.PreferredBackupWindow)
	d.FieldIf("Preferred Maintenance Window", ir.Item.PreferredMaintenanceWindow)
	if ir.Item.LatestRestorableTime != nil {
		d.Field("Latest Restorable Time", ir.Item.LatestRestorableTime.Format(time.RFC3339))
	}

	// Monitoring
	d.Section("Monitoring")
	d.Field("Enhanced Monitoring", fmt.Sprintf("%d sec", ir.Item.MonitoringInterval))
	d.FieldIf("Monitoring Role ARN", ir.Item.MonitoringRoleArn)
	d.Field("Performance Insights", fmt.Sprintf("%v", ir.Item.PerformanceInsightsEnabled))

	// Cluster info (for Aurora)
	if ir.Item.DBClusterIdentifier != nil {
		d.Section("Cluster")
		d.Field("Cluster Identifier", *ir.Item.DBClusterIdentifier)
	}

	// Tags
	d.Tags(rdsTagsToMap(ir.Item.TagList))

	return d.String()
}

// RenderSummary returns summary fields for the header panel
func (r *InstanceRenderer) RenderSummary(resource dao.Resource) []render.SummaryField {
	ir, ok := resource.(*InstanceResource)
	if !ok {
		return nil
	}

	stateStyle := render.StateColorer()(ir.State())

	fields := []render.SummaryField{
		{Label: "Identifier", Value: ir.GetID()},
		{Label: "Status", Value: ir.State(), Style: stateStyle},
		{Label: "Engine", Value: fmt.Sprintf("%s %s", ir.Engine(), ir.EngineVersion())},
	}

	fields = append(fields, render.SummaryField{Label: "Class", Value: ir.InstanceClass()})
	fields = append(fields, render.SummaryField{Label: "AZ", Value: ir.AZ()})
	if ir.MultiAZ() {
		fields = append(fields, render.SummaryField{Label: "Multi-AZ", Value: "Yes"})
	}

	if ir.Endpoint() != "" {
		fields = append(fields, render.SummaryField{
			Label: "Endpoint",
			Value: fmt.Sprintf("%s:%d", ir.Endpoint(), ir.Port()),
		})
	}

	fields = append(fields, render.SummaryField{
		Label: "Storage",
		Value: fmt.Sprintf("%d GB (%s)", ir.AllocatedStorage(), ir.StorageType()),
	})

	if ir.Item.InstanceCreateTime != nil {
		fields = append(fields, render.SummaryField{
			Label: "Created",
			Value: ir.Item.InstanceCreateTime.Format("2006-01-02 15:04") + " (" + render.FormatAge(*ir.Item.InstanceCreateTime) + ")",
		})
	}

	return fields
}

// Navigations returns navigation shortcuts for RDS instances
func (r *InstanceRenderer) Navigations(resource dao.Resource) []render.Navigation {
	ir, ok := resource.(*InstanceResource)
	if !ok {
		return nil
	}

	var navs []render.Navigation

	// VPC navigation
	if ir.Item.DBSubnetGroup != nil && ir.Item.DBSubnetGroup.VpcId != nil {
		navs = append(navs, render.Navigation{
			Key: "v", Label: "VPC", Service: "vpc", Resource: "vpcs",
			FilterField: "VpcId", FilterValue: *ir.Item.DBSubnetGroup.VpcId,
		})
	}

	// Security Groups navigation
	if len(ir.Item.VpcSecurityGroups) > 0 && ir.Item.VpcSecurityGroups[0].VpcSecurityGroupId != nil {
		navs = append(navs, render.Navigation{
			Key: "g", Label: "Security Groups", Service: "ec2", Resource: "security-groups",
			FilterField: "GroupId", FilterValue: *ir.Item.VpcSecurityGroups[0].VpcSecurityGroupId,
		})
	}

	// Snapshots navigation
	navs = append(navs, render.Navigation{
		Key: "s", Label: "Snapshots", Service: "rds", Resource: "snapshots",
		FilterField: "DBInstanceIdentifier", FilterValue: ir.GetID(),
	})

	// Cluster navigation (for Aurora instances)
	if ir.Item.DBClusterIdentifier != nil {
		navs = append(navs, render.Navigation{
			Key: "c", Label: "Cluster", Service: "rds", Resource: "clusters",
			FilterField: "DBClusterIdentifier", FilterValue: *ir.Item.DBClusterIdentifier,
		})
	}

	return navs
}
