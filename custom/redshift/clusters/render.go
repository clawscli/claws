package clusters

import (
	"fmt"

	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/render"
)

// ClusterRenderer renders Redshift clusters.
// Ensure ClusterRenderer implements render.Navigator
var _ render.Navigator = (*ClusterRenderer)(nil)

type ClusterRenderer struct {
	render.BaseRenderer
}

// NewClusterRenderer creates a new ClusterRenderer.
func NewClusterRenderer() render.Renderer {
	return &ClusterRenderer{
		BaseRenderer: render.BaseRenderer{
			Service:  "redshift",
			Resource: "clusters",
			Cols: []render.Column{
				{Name: "CLUSTER ID", Width: 30, Getter: func(r dao.Resource) string { return r.GetID() }},
				{Name: "STATUS", Width: 15, Getter: getStatus},
				{Name: "NODE TYPE", Width: 15, Getter: getNodeType},
				{Name: "NODES", Width: 8, Getter: getNodes},
				{Name: "ENDPOINT", Width: 35, Getter: getEndpoint},
			},
		},
	}
}

func getStatus(r dao.Resource) string {
	cluster, ok := r.(*ClusterResource)
	if !ok {
		return ""
	}
	return cluster.Status()
}

func getNodeType(r dao.Resource) string {
	cluster, ok := r.(*ClusterResource)
	if !ok {
		return ""
	}
	return cluster.NodeType()
}

func getNodes(r dao.Resource) string {
	cluster, ok := r.(*ClusterResource)
	if !ok {
		return ""
	}
	return fmt.Sprintf("%d", cluster.NumberOfNodes())
}

func getEndpoint(r dao.Resource) string {
	cluster, ok := r.(*ClusterResource)
	if !ok {
		return ""
	}
	return cluster.Endpoint()
}

// RenderDetail renders the detail view for a cluster.
func (r *ClusterRenderer) RenderDetail(resource dao.Resource) string {
	cluster, ok := resource.(*ClusterResource)
	if !ok {
		return ""
	}

	d := render.NewDetailBuilder()
	c := cluster.Cluster

	d.Title("Redshift Cluster", cluster.GetID())

	// Basic Info
	d.Section("Basic Information")
	d.Field("Cluster ID", cluster.GetID())
	d.Field("Status", cluster.Status())
	if c.ClusterAvailabilityStatus != nil {
		d.Field("Availability Status", *c.ClusterAvailabilityStatus)
	}
	d.Field("Node Type", cluster.NodeType())
	d.Field("Number of Nodes", fmt.Sprintf("%d", cluster.NumberOfNodes()))
	if c.ClusterVersion != nil {
		d.Field("Cluster Version", *c.ClusterVersion)
	}
	if c.ClusterRevisionNumber != nil {
		d.Field("Revision Number", *c.ClusterRevisionNumber)
	}

	// Database
	d.Section("Database")
	d.Field("Database Name", cluster.DBName())
	d.Field("Master Username", cluster.MasterUsername())
	if c.AutomatedSnapshotRetentionPeriod != nil {
		d.Field("Snapshot Retention", fmt.Sprintf("%d days", *c.AutomatedSnapshotRetentionPeriod))
	}

	// Connectivity
	d.Section("Connectivity")
	if cluster.Endpoint() != "" {
		d.Field("Endpoint", cluster.Endpoint())
	}
	if c.PubliclyAccessible != nil {
		d.Field("Publicly Accessible", fmt.Sprintf("%v", *c.PubliclyAccessible))
	}
	d.Field("VPC ID", cluster.VpcId())
	if c.AvailabilityZone != nil {
		d.Field("Availability Zone", *c.AvailabilityZone)
	}
	if c.ClusterSubnetGroupName != nil {
		d.Field("Subnet Group", *c.ClusterSubnetGroupName)
	}
	if len(c.VpcSecurityGroups) > 0 {
		for i, sg := range c.VpcSecurityGroups {
			if sg.VpcSecurityGroupId != nil {
				label := "Security Group"
				if i > 0 {
					label = fmt.Sprintf("Security Group %d", i+1)
				}
				status := ""
				if sg.Status != nil {
					status = fmt.Sprintf(" (%s)", *sg.Status)
				}
				d.Field(label, *sg.VpcSecurityGroupId+status)
			}
		}
	}
	if c.EnhancedVpcRouting != nil && *c.EnhancedVpcRouting {
		d.Field("Enhanced VPC Routing", "Enabled")
	}

	// Encryption
	if c.Encrypted != nil && *c.Encrypted {
		d.Section("Encryption")
		d.Field("Encrypted", "Yes")
		if c.KmsKeyId != nil {
			d.Field("KMS Key", *c.KmsKeyId)
		}
	}

	// Maintenance
	if c.PreferredMaintenanceWindow != nil || (c.AllowVersionUpgrade != nil && *c.AllowVersionUpgrade) {
		d.Section("Maintenance")
		if c.PreferredMaintenanceWindow != nil {
			d.Field("Maintenance Window", *c.PreferredMaintenanceWindow)
		}
		if c.AllowVersionUpgrade != nil {
			d.Field("Auto Upgrade", fmt.Sprintf("%v", *c.AllowVersionUpgrade))
		}
	}

	// Timestamps
	d.Section("Timestamps")
	if t := cluster.CreatedAt(); t != nil {
		d.Field("Created", t.Format("2006-01-02 15:04:05"))
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
		{Label: "Status", Value: cluster.Status()},
		{Label: "Node Type", Value: cluster.NodeType()},
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
			Label:       "Snapshots",
			Service:     "redshift",
			Resource:    "snapshots",
			FilterField: "ClusterIdentifier",
			FilterValue: cluster.GetID(),
		},
	}
}
