package nodegroups

import (
	"encoding/json"
	"fmt"
	"strings"

	appaws "github.com/clawscli/claws/internal/aws"
	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/render"
)

// NodeGroupRenderer renders EKS node group resources
type NodeGroupRenderer struct {
	render.BaseRenderer
}

// NewNodeGroupRenderer creates a new NodeGroupRenderer
func NewNodeGroupRenderer() render.Renderer {
	return &NodeGroupRenderer{
		BaseRenderer: render.BaseRenderer{
			Service:  "eks",
			Resource: "node-groups",
			Cols: []render.Column{
				{
					Name:     "NAME",
					Width:    30,
					Priority: 0,
					Getter: func(r dao.Resource) string {
						return r.GetName()
					},
				},
				{
					Name:     "STATUS",
					Width:    12,
					Priority: 1,
					Getter: func(r dao.Resource) string {
						if ngr, ok := r.(*NodeGroupResource); ok {
							return ngr.Status()
						}
						return ""
					},
				},
				{
					Name:     "VERSION",
					Width:    10,
					Priority: 2,
					Getter: func(r dao.Resource) string {
						if ngr, ok := r.(*NodeGroupResource); ok {
							return ngr.Version()
						}
						return ""
					},
				},
				{
					Name:     "INSTANCE TYPE",
					Width:    15,
					Priority: 3,
					Getter: func(r dao.Resource) string {
						if ngr, ok := r.(*NodeGroupResource); ok {
							return ngr.InstanceTypes()
						}
						return ""
					},
				},
				{
					Name:     "DESIRED/MIN/MAX",
					Width:    15,
					Priority: 4,
					Getter: func(r dao.Resource) string {
						if ngr, ok := r.(*NodeGroupResource); ok {
							return fmt.Sprintf("%d/%d/%d", ngr.DesiredSize(), ngr.MinSize(), ngr.MaxSize())
						}
						return ""
					},
				},
				{
					Name:     "AGE",
					Width:    10,
					Priority: 5,
					Getter: func(r dao.Resource) string {
						if ngr, ok := r.(*NodeGroupResource); ok {
							return ngr.CreatedAge()
						}
						return ""
					},
				},
			},
		},
	}
}

func (rnd *NodeGroupRenderer) RenderDetail(resource dao.Resource) string {
	ngr, ok := resource.(*NodeGroupResource)
	if !ok {
		return ""
	}

	d := render.NewDetailBuilder()
	d.Title("EKS Node Group", ngr.GetName())

	// Basic Information
	d.Section("Basic Information")
	d.Field("Name", ngr.GetName())
	d.Field("ARN", ngr.GetARN())
	d.Field("Cluster", appaws.Str(ngr.NodeGroup.ClusterName))
	d.Field("Status", ngr.Status())
	d.Field("Version", ngr.Version())
	d.Field("Release Version", appaws.Str(ngr.NodeGroup.ReleaseVersion))
	d.Field("Created", ngr.CreatedAge())
	if modified := appaws.Time(ngr.NodeGroup.ModifiedAt); !modified.IsZero() {
		d.Field("Modified", render.FormatAge(modified))
	}

	// Instance Configuration
	d.Section("Instance Configuration")
	d.Field("AMI Type", string(ngr.NodeGroup.AmiType))
	d.Field("Capacity Type", string(ngr.NodeGroup.CapacityType))
	if len(ngr.NodeGroup.InstanceTypes) > 0 {
		d.Field("Instance Types", strings.Join(ngr.NodeGroup.InstanceTypes, ", "))
	} else {
		d.Field("Instance Types", render.Empty)
	}
	if diskSize := appaws.Int32(ngr.NodeGroup.DiskSize); diskSize > 0 {
		d.Field("Disk Size (GB)", fmt.Sprintf("%d", diskSize))
	}
	d.Field("Node Role", appaws.Str(ngr.NodeGroup.NodeRole))

	// Scaling Configuration
	d.Section("Scaling Configuration")
	if sc := ngr.NodeGroup.ScalingConfig; sc != nil {
		d.Field("Desired Size", fmt.Sprintf("%d", appaws.Int32(sc.DesiredSize)))
		d.Field("Min Size", fmt.Sprintf("%d", appaws.Int32(sc.MinSize)))
		d.Field("Max Size", fmt.Sprintf("%d", appaws.Int32(sc.MaxSize)))
	}

	// Update Configuration
	if uc := ngr.NodeGroup.UpdateConfig; uc != nil {
		d.Section("Update Configuration")
		if uc.UpdateStrategy != "" {
			d.Field("Update Strategy", string(uc.UpdateStrategy))
		}
		if maxUnavail := appaws.Int32(uc.MaxUnavailable); maxUnavail > 0 {
			d.Field("Max Unavailable", fmt.Sprintf("%d", maxUnavail))
		}
		if maxUnavailPct := appaws.Int32(uc.MaxUnavailablePercentage); maxUnavailPct > 0 {
			d.Field("Max Unavailable %", fmt.Sprintf("%d%%", maxUnavailPct))
		}
	}

	// Launch Template
	if lt := ngr.NodeGroup.LaunchTemplate; lt != nil {
		d.Section("Launch Template")
		d.Field("Name", appaws.Str(lt.Name))
		d.Field("ID", appaws.Str(lt.Id))
		d.Field("Version", appaws.Str(lt.Version))
	}

	// Remote Access
	if ra := ngr.NodeGroup.RemoteAccess; ra != nil {
		d.Section("Remote Access")
		d.Field("EC2 SSH Key", appaws.Str(ra.Ec2SshKey))
		if len(ra.SourceSecurityGroups) > 0 {
			d.Field("Source Security Groups", strings.Join(ra.SourceSecurityGroups, ", "))
		}
	}

	// Network
	d.Section("Network")
	if len(ngr.NodeGroup.Subnets) > 0 {
		d.Field("Subnets", strings.Join(ngr.NodeGroup.Subnets, ", "))
	}

	// Resources
	if res := ngr.NodeGroup.Resources; res != nil {
		d.Section("Resources")
		if len(res.AutoScalingGroups) > 0 {
			for i, asg := range res.AutoScalingGroups {
				d.Field(fmt.Sprintf("Auto Scaling Group #%d", i+1), appaws.Str(asg.Name))
			}
		}
		if rsg := appaws.Str(res.RemoteAccessSecurityGroup); rsg != "" {
			d.Field("Remote Access Security Group", rsg)
		}
	}

	// Labels
	if len(ngr.NodeGroup.Labels) > 0 {
		d.Section("Labels")
		d.Tags(ngr.NodeGroup.Labels)
	}

	// Taints
	if len(ngr.NodeGroup.Taints) > 0 {
		d.Section("Taints")
		for i, taint := range ngr.NodeGroup.Taints {
			d.Field(fmt.Sprintf("Taint #%d", i+1), fmt.Sprintf("%s=%s:%s",
				appaws.Str(taint.Key),
				appaws.Str(taint.Value),
				string(taint.Effect)))
		}
	}

	// Health Issues
	if health := ngr.NodeGroup.Health; health != nil && len(health.Issues) > 0 {
		d.Section("Health Issues")
		for i, issue := range health.Issues {
			d.Field(fmt.Sprintf("Issue #%d Code", i+1), string(issue.Code))
			if msg := appaws.Str(issue.Message); msg != "" {
				d.Field(fmt.Sprintf("Issue #%d Message", i+1), msg)
			}
			if len(issue.ResourceIds) > 0 {
				d.Field(fmt.Sprintf("Issue #%d Resources", i+1), strings.Join(issue.ResourceIds, ", "))
			}
		}
	}

	// Tags
	if len(ngr.NodeGroup.Tags) > 0 {
		d.Section("Tags")
		d.Tags(ngr.NodeGroup.Tags)
	}

	// Full Details
	d.Section("Full Details")
	if jsonBytes, err := json.MarshalIndent(ngr.NodeGroup, "", "  "); err == nil {
		d.Line(string(jsonBytes))
	}

	return d.String()
}

func (rnd *NodeGroupRenderer) RenderSummary(resource dao.Resource) []render.SummaryField {
	ngr, ok := resource.(*NodeGroupResource)
	if !ok {
		return nil
	}

	return []render.SummaryField{
		{Label: "Name", Value: ngr.GetName()},
		{Label: "Status", Value: ngr.Status()},
		{Label: "Version", Value: ngr.Version()},
		{Label: "Desired", Value: fmt.Sprintf("%d", ngr.DesiredSize())},
	}
}
