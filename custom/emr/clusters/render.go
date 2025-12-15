package clusters

import (
	"fmt"
	"strings"

	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/render"
)

// ClusterRenderer renders EMR clusters.
// Ensure ClusterRenderer implements render.Navigator
var _ render.Navigator = (*ClusterRenderer)(nil)

type ClusterRenderer struct {
	render.BaseRenderer
}

// NewClusterRenderer creates a new ClusterRenderer.
func NewClusterRenderer() render.Renderer {
	return &ClusterRenderer{
		BaseRenderer: render.BaseRenderer{
			Service:  "emr",
			Resource: "clusters",
			Cols: []render.Column{
				{Name: "CLUSTER ID", Width: 20, Getter: func(r dao.Resource) string { return r.GetID() }},
				{Name: "NAME", Width: 35, Getter: getName},
				{Name: "STATE", Width: 15, Getter: getState},
				{Name: "HOURS", Width: 10, Getter: getHours},
			},
		},
	}
}

func getName(r dao.Resource) string {
	cluster, ok := r.(*ClusterResource)
	if !ok {
		return ""
	}
	return cluster.Name()
}

func getState(r dao.Resource) string {
	cluster, ok := r.(*ClusterResource)
	if !ok {
		return ""
	}
	return cluster.State()
}

func getHours(r dao.Resource) string {
	cluster, ok := r.(*ClusterResource)
	if !ok {
		return ""
	}
	hours := cluster.NormalizedInstanceHours()
	if hours == 0 {
		return ""
	}
	return fmt.Sprintf("%d", hours)
}

// RenderDetail renders the detail view for a cluster.
func (r *ClusterRenderer) RenderDetail(resource dao.Resource) string {
	cluster, ok := resource.(*ClusterResource)
	if !ok {
		return ""
	}

	d := render.NewDetailBuilder()

	d.Title("EMR Cluster", cluster.GetID())

	// Basic Info
	d.Section("Basic Information")
	d.Field("Cluster ID", cluster.GetID())
	d.Field("Name", cluster.Name())
	d.Field("ARN", cluster.GetARN())
	d.Field("State", cluster.State())

	if cluster.ReleaseLabel != "" {
		d.Field("Release Label", cluster.ReleaseLabel)
	}
	if apps := cluster.GetApplications(); len(apps) > 0 {
		d.Field("Applications", strings.Join(apps, ", "))
	}
	d.Field("Visible To All Users", fmt.Sprintf("%v", cluster.GetVisibleToAllUsers()))

	// State Change Reason
	if reason := cluster.GetStateChangeReason(); reason != nil {
		if reason.Code != "" || reason.Message != nil {
			d.Section("State Change Reason")
			if reason.Code != "" {
				d.Field("Code", string(reason.Code))
			}
			if reason.Message != nil {
				d.Field("Message", *reason.Message)
			}
		}
	}

	// Master Node
	if dns := cluster.GetMasterPublicDnsName(); dns != "" {
		d.Section("Master Node")
		d.Field("Public DNS", dns)
	}

	// EC2 Instance Attributes
	if ec2 := cluster.GetEc2InstanceAttrs(); ec2 != nil {
		d.Section("EC2 Configuration")
		if ec2.Ec2KeyName != nil {
			d.Field("Key Name", *ec2.Ec2KeyName)
		}
		if ec2.Ec2SubnetId != nil {
			d.Field("Subnet ID", *ec2.Ec2SubnetId)
		}
		if ec2.Ec2AvailabilityZone != nil {
			d.Field("Availability Zone", *ec2.Ec2AvailabilityZone)
		}
		if ec2.IamInstanceProfile != nil {
			d.Field("Instance Profile", *ec2.IamInstanceProfile)
		}
		if ec2.EmrManagedMasterSecurityGroup != nil {
			d.Field("Master SG", *ec2.EmrManagedMasterSecurityGroup)
		}
		if ec2.EmrManagedSlaveSecurityGroup != nil {
			d.Field("Slave SG", *ec2.EmrManagedSlaveSecurityGroup)
		}
		if ec2.ServiceAccessSecurityGroup != nil {
			d.Field("Service Access SG", *ec2.ServiceAccessSecurityGroup)
		}
	}

	// Roles
	if cluster.GetServiceRole() != "" || cluster.GetAutoScalingRole() != "" {
		d.Section("IAM Roles")
		if cluster.GetServiceRole() != "" {
			d.Field("Service Role", cluster.GetServiceRole())
		}
		if cluster.GetAutoScalingRole() != "" {
			d.Field("Auto Scaling Role", cluster.GetAutoScalingRole())
		}
	}

	// Configuration
	if cluster.GetLogUri() != "" || cluster.GetScaleDownBehavior() != "" {
		d.Section("Configuration")
		if cluster.GetLogUri() != "" {
			d.Field("Log URI", cluster.GetLogUri())
		}
		if cluster.GetScaleDownBehavior() != "" {
			d.Field("Scale Down Behavior", cluster.GetScaleDownBehavior())
		}
	}

	// Metrics
	if cluster.NormalizedInstanceHours() > 0 {
		d.Section("Metrics")
		d.Field("Normalized Instance Hours", fmt.Sprintf("%d", cluster.NormalizedInstanceHours()))
	}

	// Tags
	if tags := cluster.GetClusterTags(); len(tags) > 0 {
		d.Section("Tags")
		for _, tag := range tags {
			if tag.Key != nil && tag.Value != nil {
				d.Field(*tag.Key, *tag.Value)
			}
		}
	}

	return d.String()
}

// RenderSummary renders summary fields for a cluster.
func (r *ClusterRenderer) RenderSummary(resource dao.Resource) []render.SummaryField {
	cluster, ok := resource.(*ClusterResource)
	if !ok {
		return r.BaseRenderer.RenderSummary(resource)
	}

	return []render.SummaryField{
		{Label: "Cluster ID", Value: cluster.GetID()},
		{Label: "Name", Value: cluster.Name()},
		{Label: "State", Value: cluster.State()},
	}
}

// Navigations returns available navigations from a cluster.
func (r *ClusterRenderer) Navigations(resource dao.Resource) []render.Navigation {
	cluster, ok := resource.(*ClusterResource)
	if !ok {
		return nil
	}
	return []render.Navigation{
		{
			Key:         "s",
			Label:       "Steps",
			Service:     "emr",
			Resource:    "steps",
			FilterField: "ClusterId",
			FilterValue: cluster.GetID(),
		},
	}
}
