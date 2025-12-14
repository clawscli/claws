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

// InstanceRenderer renders EC2 instances with custom columns
type InstanceRenderer struct {
	render.BaseRenderer
}

// NewInstanceRenderer creates a new InstanceRenderer
func NewInstanceRenderer() render.Renderer {
	return &InstanceRenderer{
		BaseRenderer: render.BaseRenderer{
			Service:  "ec2",
			Resource: "instances",
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
					Width: 21,
					Getter: func(r dao.Resource) string {
						return r.GetID()
					},
					Priority: 1,
				},
				{
					Name:  "STATE",
					Width: 11,
					Getter: func(r dao.Resource) string {
						if ir, ok := r.(*InstanceResource); ok {
							return ir.State()
						}
						return ""
					},
					Priority: 2,
				},
				{
					Name:  "TYPE",
					Width: 13,
					Getter: func(r dao.Resource) string {
						if ir, ok := r.(*InstanceResource); ok {
							return ir.InstanceType()
						}
						return ""
					},
					Priority: 3,
				},
				{
					Name:  "PRIVATE IP",
					Width: 16,
					Getter: func(r dao.Resource) string {
						if ir, ok := r.(*InstanceResource); ok {
							return ir.PrivateIP()
						}
						return ""
					},
					Priority: 4,
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
					Priority: 5,
				},
				{
					Name:  "AGE",
					Width: 8,
					Getter: func(r dao.Resource) string {
						if ir, ok := r.(*InstanceResource); ok {
							if ir.Item.LaunchTime != nil {
								return render.FormatAge(*ir.Item.LaunchTime)
							}
						}
						return ""
					},
					Priority: 6,
				},
				render.TagsColumn(30, 7),
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

	d.Title("EC2 Instance", ir.GetName())

	// Basic Info
	d.Section("Basic Information")
	d.Field("Instance ID", ir.GetID())
	d.FieldStyled("State", ir.State(), render.StateColorer()(ir.State()))
	d.Field("Type", ir.InstanceType())
	d.Field("AZ", ir.AZ())

	if ir.Item.Platform != "" {
		d.Field("Platform", string(ir.Item.Platform))
	}
	if ir.Item.Architecture != "" {
		d.Field("Architecture", string(ir.Item.Architecture))
	}
	d.FieldIf("AMI ID", ir.Item.ImageId)
	d.FieldIf("Key Name", ir.Item.KeyName)

	// Instance lifecycle (spot vs on-demand)
	if lifecycle := ir.InstanceLifecycle(); lifecycle != "" {
		d.Field("Lifecycle", lifecycle)
	} else {
		d.Field("Lifecycle", "on-demand")
	}

	if ir.Item.LaunchTime != nil {
		d.Field("Launch Time", ir.Item.LaunchTime.Format(time.RFC3339))
		d.Field("Age", render.FormatAge(*ir.Item.LaunchTime))
	}

	// State reason (useful for stopped/terminated instances)
	if reason := ir.StateReason(); reason != "" {
		d.Section("State Information")
		d.Field("State Reason", reason)
		if code := ir.StateReasonCode(); code != "" {
			d.Field("Reason Code", code)
		}
	}

	// Compute Configuration
	d.Section("Compute Configuration")
	if cores := ir.CpuCoreCount(); cores > 0 {
		threads := ir.CpuThreadsPerCore()
		if threads > 0 {
			d.Field("CPU", fmt.Sprintf("%d cores × %d threads = %d vCPUs", cores, threads, cores*threads))
		} else {
			d.Field("CPU Cores", fmt.Sprintf("%d", cores))
		}
	}
	d.Field("Virtualization", ir.VirtualizationType())
	if hypervisor := ir.Hypervisor(); hypervisor != "" {
		d.Field("Hypervisor", hypervisor)
	}
	if tenancy := ir.Tenancy(); tenancy != "" && tenancy != "default" {
		d.Field("Tenancy", tenancy)
	}
	if ir.EbsOptimized() {
		d.Field("EBS Optimized", "Enabled")
	}
	if ir.HibernationEnabled() {
		d.Field("Hibernation", "Enabled")
	}
	if ir.EnclaveEnabled() {
		d.Field("Nitro Enclave", "Enabled")
	}

	// Monitoring
	if monitoring := ir.MonitoringState(); monitoring != "" {
		d.Field("Monitoring", monitoring)
	}

	// Network
	d.Section("Network")
	d.Field("Private IP", ir.PrivateIP())
	d.Field("Public IP", ir.PublicIP())
	d.FieldIf("Private DNS", ir.Item.PrivateDnsName)
	d.FieldIf("Public DNS", ir.Item.PublicDnsName)
	d.FieldIf("VPC ID", ir.Item.VpcId)
	d.FieldIf("Subnet ID", ir.Item.SubnetId)
	// Source/Dest check (important for NAT instances)
	if !ir.SourceDestCheck() {
		d.Field("Source/Dest Check", "Disabled")
	}

	// Instance Metadata Service (IMDS)
	if httpTokens := ir.MetadataHttpTokens(); httpTokens != "" {
		d.Section("Instance Metadata Service")
		if httpTokens == "required" {
			d.Field("IMDSv2", "Required (secure)")
		} else {
			d.Field("IMDSv2", "Optional")
		}
		if endpoint := ir.MetadataHttpEndpoint(); endpoint != "" {
			d.Field("IMDS Endpoint", endpoint)
		}
	}

	// Security Groups
	if len(ir.Item.SecurityGroups) > 0 {
		d.Section("Security Groups")
		for _, sg := range ir.Item.SecurityGroups {
			name := appaws.Str(sg.GroupName)
			id := appaws.Str(sg.GroupId)
			d.Line("  " + styles.Value.Render(name) + styles.Dim.Render(" ("+id+")"))
		}
	}

	// Storage
	d.Section("Storage")
	d.Field("Root Device Type", ir.RootDeviceType())
	if rootName := ir.RootDeviceName(); rootName != "" {
		d.Field("Root Device Name", rootName)
	}

	// Block Devices
	if len(ir.Item.BlockDeviceMappings) > 0 {
		d.Section("Block Devices")
		for _, bd := range ir.Item.BlockDeviceMappings {
			device := appaws.Str(bd.DeviceName)
			volId := ""
			status := ""
			if bd.Ebs != nil {
				volId = appaws.Str(bd.Ebs.VolumeId)
				status = string(bd.Ebs.Status)
			}
			info := volId
			if status != "" {
				info = volId + " (" + status + ")"
			}
			d.Line("  " + styles.Value.Render(device) + styles.Dim.Render(" → "+info))
		}
	}

	// Network Interfaces
	if len(ir.Item.NetworkInterfaces) > 0 {
		d.Section("Network Interfaces")
		for _, ni := range ir.Item.NetworkInterfaces {
			eniId := appaws.Str(ni.NetworkInterfaceId)
			status := string(ni.Status)
			privateIp := appaws.Str(ni.PrivateIpAddress)
			d.Field(eniId, fmt.Sprintf("%s (%s)", privateIp, status))
		}
	}

	// IAM Role
	if ir.Item.IamInstanceProfile != nil && ir.Item.IamInstanceProfile.Arn != nil {
		d.Section("IAM Instance Profile")
		d.Field("ARN", *ir.Item.IamInstanceProfile.Arn)
		if ir.RoleName != "" {
			d.Field("Role Name", ir.RoleName)
		}
	}

	// Tags
	d.Tags(appaws.TagsToMap(ir.Item.Tags))

	return d.String()
}

// RenderSummary returns summary fields for the header panel
func (r *InstanceRenderer) RenderSummary(resource dao.Resource) []render.SummaryField {
	ir, ok := resource.(*InstanceResource)
	if !ok {
		return nil
	}

	stateStyle := render.StateColorer()(ir.State())

	// Row 1: ID, Name, State
	fields := []render.SummaryField{
		{Label: "ID", Value: ir.GetID()},
		{Label: "Name", Value: ir.GetName()},
		{Label: "State", Value: ir.State(), Style: stateStyle},
	}

	// Row 2: Type, AZ, Platform
	fields = append(fields, render.SummaryField{Label: "Type", Value: ir.InstanceType()})
	fields = append(fields, render.SummaryField{Label: "AZ", Value: ir.AZ()})
	if ir.Item.Architecture != "" {
		fields = append(fields, render.SummaryField{Label: "Arch", Value: string(ir.Item.Architecture)})
	}

	// Row 3: Network
	fields = append(fields, render.SummaryField{Label: "Private IP", Value: ir.PrivateIP()})
	fields = append(fields, render.SummaryField{Label: "Public IP", Value: ir.PublicIP()})
	if ir.Item.VpcId != nil {
		fields = append(fields, render.SummaryField{Label: "VPC", Value: *ir.Item.VpcId})
	}

	// Row 4: Additional info
	if ir.Item.SubnetId != nil {
		fields = append(fields, render.SummaryField{Label: "Subnet", Value: *ir.Item.SubnetId})
	}
	if ir.Item.ImageId != nil {
		fields = append(fields, render.SummaryField{Label: "AMI", Value: *ir.Item.ImageId})
	}
	if ir.Item.KeyName != nil {
		fields = append(fields, render.SummaryField{Label: "Key", Value: *ir.Item.KeyName})
	}

	// Row 5: Launch time
	if ir.Item.LaunchTime != nil {
		fields = append(fields, render.SummaryField{
			Label: "Launched",
			Value: ir.Item.LaunchTime.Format("2006-01-02 15:04") + " (" + render.FormatAge(*ir.Item.LaunchTime) + ")",
		})
	}

	return fields
}

// Navigations returns navigation shortcuts for EC2 instances
func (r *InstanceRenderer) Navigations(resource dao.Resource) []render.Navigation {
	ir, ok := resource.(*InstanceResource)
	if !ok {
		return nil
	}

	var navs []render.Navigation

	// VPC navigation
	if ir.Item.VpcId != nil {
		navs = append(navs, render.Navigation{
			Key: "v", Label: "VPC", Service: "vpc", Resource: "vpcs",
			FilterField: "VpcId", FilterValue: *ir.Item.VpcId,
		})
	}

	// Subnet navigation
	if ir.Item.SubnetId != nil {
		navs = append(navs, render.Navigation{
			Key: "u", Label: "Subnet", Service: "vpc", Resource: "subnets",
			FilterField: "SubnetId", FilterValue: *ir.Item.SubnetId,
		})
	}

	// Security Groups - navigate to SGs in same VPC
	if ir.Item.VpcId != nil {
		navs = append(navs, render.Navigation{
			Key: "g", Label: "Security Groups", Service: "ec2", Resource: "security-groups",
			FilterField: "VpcId", FilterValue: *ir.Item.VpcId,
		})
	}

	// IAM Role navigation
	if ir.RoleName != "" {
		navs = append(navs, render.Navigation{
			Key: "r", Label: "Role", Service: "iam", Resource: "roles",
			FilterField: "RoleName", FilterValue: ir.RoleName,
		})
	}

	return navs
}
