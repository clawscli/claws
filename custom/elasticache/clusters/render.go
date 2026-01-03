package clusters

import (
	"fmt"
	"strings"

	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/render"
	"github.com/clawscli/claws/internal/ui"
)

// ClusterRenderer renders ElastiCache clusters
// Ensure ClusterRenderer implements render.Navigator
var _ render.Navigator = (*ClusterRenderer)(nil)

type ClusterRenderer struct {
	render.BaseRenderer
}

// NewClusterRenderer creates a new ClusterRenderer
func NewClusterRenderer() *ClusterRenderer {
	return &ClusterRenderer{
		BaseRenderer: render.BaseRenderer{
			Service:  "elasticache",
			Resource: "clusters",
			Cols: []render.Column{
				{Name: "CLUSTER ID", Width: 28, Getter: getClusterId},
				{Name: "ENGINE", Width: 10, Getter: getEngine},
				{Name: "VERSION", Width: 8, Getter: getVersion},
				{Name: "TYPE", Width: 18, Getter: getNodeType},
				{Name: "NODES", Width: 6, Getter: getNumNodes},
				{Name: "STATUS", Width: 12, Getter: getStatus},
				{Name: "AZ", Width: 14, Getter: getAZ},
			},
		},
	}
}

func getClusterId(r dao.Resource) string {
	if cluster, ok := r.(*ClusterResource); ok {
		return cluster.ClusterId()
	}
	return ""
}

func getEngine(r dao.Resource) string {
	if cluster, ok := r.(*ClusterResource); ok {
		return cluster.Engine()
	}
	return ""
}

func getVersion(r dao.Resource) string {
	if cluster, ok := r.(*ClusterResource); ok {
		return cluster.EngineVersion()
	}
	return ""
}

func getNodeType(r dao.Resource) string {
	if cluster, ok := r.(*ClusterResource); ok {
		return cluster.NodeType()
	}
	return ""
}

func getNumNodes(r dao.Resource) string {
	if cluster, ok := r.(*ClusterResource); ok {
		return fmt.Sprintf("%d", cluster.NumNodes())
	}
	return ""
}

func getStatus(r dao.Resource) string {
	if cluster, ok := r.(*ClusterResource); ok {
		return cluster.Status()
	}
	return ""
}

func getAZ(r dao.Resource) string {
	if cluster, ok := r.(*ClusterResource); ok {
		return cluster.AvailabilityZone()
	}
	return ""
}

// RenderDetail renders detailed cluster information
func (r *ClusterRenderer) RenderDetail(resource dao.Resource) string {
	cluster, ok := resource.(*ClusterResource)
	if !ok {
		return ""
	}

	d := render.NewDetailBuilder()

	d.Title("ElastiCache Cluster", cluster.ClusterId())

	// Basic Info
	d.Section("Basic Information")
	d.Field("Cluster ID", cluster.ClusterId())
	d.Field("ARN", cluster.GetARN())
	d.Field("Status", cluster.Status())

	// Engine Configuration
	d.Section("Engine Configuration")
	d.Field("Engine", cluster.Engine())
	d.Field("Engine Version", cluster.EngineVersion())
	d.Field("Node Type", cluster.NodeType())
	d.Field("Number of Nodes", fmt.Sprintf("%d", cluster.NumNodes()))

	// Networking
	d.Section("Networking")
	if endpoint := cluster.Endpoint(); endpoint != "" {
		d.Field("Endpoint", endpoint)
	}
	d.Field("Availability Zone", cluster.AvailabilityZone())
	if subnetGroup := cluster.SubnetGroupName(); subnetGroup != "" {
		d.Field("Subnet Group", subnetGroup)
	}
	if sgs := cluster.SecurityGroups(); len(sgs) > 0 {
		d.Field("Security Groups", strings.Join(sgs, ", "))
	}

	// Replication (Redis)
	if replGroup := cluster.ReplicationGroupId(); replGroup != "" {
		d.Section("Replication")
		d.Field("Replication Group", replGroup)
	}

	// Configuration
	d.Section("Configuration")
	if paramGroup := cluster.ParameterGroupName(); paramGroup != "" {
		d.Field("Parameter Group", paramGroup)
	}
	d.Field("Maintenance Window", cluster.MaintenanceWindow())
	if snapshotWindow := cluster.SnapshotWindow(); snapshotWindow != "" {
		d.Field("Snapshot Window", snapshotWindow)
		d.Field("Snapshot Retention", fmt.Sprintf("%d days", cluster.SnapshotRetentionLimit()))
	}

	// Cache Nodes
	if len(cluster.Item.CacheNodes) > 0 {
		d.Section("Cache Nodes")
		for i, node := range cluster.Item.CacheNodes {
			status := ""
			if node.CacheNodeStatus != nil {
				status = *node.CacheNodeStatus
			}
			nodeId := fmt.Sprintf("Node %d", i+1)
			if node.CacheNodeId != nil {
				nodeId = *node.CacheNodeId
			}
			if status == "available" {
				d.FieldStyled(nodeId, status, ui.SuccessStyle())
			} else {
				d.Field(nodeId, status)
			}
		}
	}

	// Security
	d.Section("Security")
	if cluster.TransitEncryptionEnabled() {
		d.FieldStyled("Transit Encryption", "Enabled", ui.SuccessStyle())
	} else {
		d.Field("Transit Encryption", "Disabled")
	}
	if cluster.AtRestEncryptionEnabled() {
		d.FieldStyled("At-Rest Encryption", "Enabled", ui.SuccessStyle())
	} else {
		d.Field("At-Rest Encryption", "Disabled")
	}
	if cluster.Item.AuthTokenEnabled != nil && *cluster.Item.AuthTokenEnabled {
		d.FieldStyled("AUTH Token", "Enabled", ui.SuccessStyle())
	}
	d.Field("Auto Minor Version Upgrade", formatBool(cluster.AutoMinorVersionUpgrade()))

	// Notification
	if cluster.Item.NotificationConfiguration != nil && cluster.Item.NotificationConfiguration.TopicArn != nil {
		d.Section("Notifications")
		d.Field("SNS Topic", *cluster.Item.NotificationConfiguration.TopicArn)
		if cluster.Item.NotificationConfiguration.TopicStatus != nil {
			d.Field("Status", *cluster.Item.NotificationConfiguration.TopicStatus)
		}
	}

	// Timestamps
	d.Section("Timestamps")
	if created := cluster.CreatedAt(); created != "" {
		d.Field("Created", created)
	}

	return d.String()
}

func formatBool(b bool) string {
	if b {
		return "Enabled"
	}
	return "Disabled"
}

// RenderSummary returns summary fields for the header panel
func (r *ClusterRenderer) RenderSummary(resource dao.Resource) []render.SummaryField {
	cluster, ok := resource.(*ClusterResource)
	if !ok {
		return r.BaseRenderer.RenderSummary(resource)
	}

	fields := []render.SummaryField{
		{Label: "Cluster ID", Value: cluster.ClusterId()},
		{Label: "ARN", Value: cluster.GetARN()},
		{Label: "Status", Value: cluster.Status()},
		{Label: "Engine", Value: fmt.Sprintf("%s %s", cluster.Engine(), cluster.EngineVersion())},
		{Label: "Node Type", Value: cluster.NodeType()},
		{Label: "Nodes", Value: fmt.Sprintf("%d", cluster.NumNodes())},
	}

	if endpoint := cluster.Endpoint(); endpoint != "" {
		fields = append(fields, render.SummaryField{Label: "Endpoint", Value: endpoint})
	}

	if created := cluster.CreatedAt(); created != "" {
		fields = append(fields, render.SummaryField{Label: "Created", Value: created})
	}

	return fields
}

// Navigations returns navigation shortcuts
func (r *ClusterRenderer) Navigations(resource dao.Resource) []render.Navigation {
	// No navigations for now
	return nil
}
