package snapshots

import (
	"fmt"

	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/render"
)

// SnapshotRenderer renders Redshift snapshots.
type SnapshotRenderer struct {
	render.BaseRenderer
}

// NewSnapshotRenderer creates a new SnapshotRenderer.
func NewSnapshotRenderer() render.Renderer {
	return &SnapshotRenderer{
		BaseRenderer: render.BaseRenderer{
			Service:  "redshift",
			Resource: "snapshots",
			Cols: []render.Column{
				{Name: "SNAPSHOT ID", Width: 40, Getter: func(r dao.Resource) string { return r.GetID() }},
				{Name: "CLUSTER", Width: 25, Getter: getCluster},
				{Name: "STATUS", Width: 12, Getter: getStatus},
				{Name: "TYPE", Width: 12, Getter: getType},
				{Name: "SIZE (MB)", Width: 12, Getter: getSize},
			},
		},
	}
}

func getCluster(r dao.Resource) string {
	snapshot, ok := r.(*SnapshotResource)
	if !ok {
		return ""
	}
	return snapshot.ClusterIdentifier()
}

func getStatus(r dao.Resource) string {
	snapshot, ok := r.(*SnapshotResource)
	if !ok {
		return ""
	}
	return snapshot.Status()
}

func getType(r dao.Resource) string {
	snapshot, ok := r.(*SnapshotResource)
	if !ok {
		return ""
	}
	return snapshot.SnapshotType()
}

func getSize(r dao.Resource) string {
	snapshot, ok := r.(*SnapshotResource)
	if !ok {
		return ""
	}
	size := snapshot.TotalBackupSize()
	if size == 0 {
		return ""
	}
	return fmt.Sprintf("%.2f", size)
}

// RenderDetail renders the detail view for a snapshot.
func (r *SnapshotRenderer) RenderDetail(resource dao.Resource) string {
	snapshot, ok := resource.(*SnapshotResource)
	if !ok {
		return ""
	}

	d := render.NewDetailBuilder()
	s := snapshot.Snapshot

	d.Title("Redshift Snapshot", snapshot.GetID())

	// Basic Info
	d.Section("Basic Information")
	d.Field("Snapshot ID", snapshot.GetID())
	d.Field("Cluster ID", snapshot.ClusterIdentifier())
	d.Field("Status", snapshot.Status())
	d.Field("Type", snapshot.SnapshotType())
	if s.DBName != nil {
		d.Field("Database", *s.DBName)
	}
	if s.AvailabilityZone != nil {
		d.Field("Availability Zone", *s.AvailabilityZone)
	}

	// Size
	d.Section("Size & Progress")
	d.Field("Total Backup Size", fmt.Sprintf("%.2f MB", snapshot.TotalBackupSize()))
	if s.ActualIncrementalBackupSizeInMegaBytes != nil && *s.ActualIncrementalBackupSizeInMegaBytes > 0 {
		d.Field("Incremental Size", fmt.Sprintf("%.2f MB", *s.ActualIncrementalBackupSizeInMegaBytes))
	}
	if s.BackupProgressInMegaBytes != nil && *s.BackupProgressInMegaBytes > 0 {
		d.Field("Backup Progress", fmt.Sprintf("%.2f MB", *s.BackupProgressInMegaBytes))
	}
	if s.ElapsedTimeInSeconds != nil && *s.ElapsedTimeInSeconds > 0 {
		d.Field("Elapsed Time", fmt.Sprintf("%ds", *s.ElapsedTimeInSeconds))
	}

	// Cluster Info
	d.Section("Cluster Info")
	d.Field("Node Type", snapshot.NodeType())
	d.Field("Number of Nodes", fmt.Sprintf("%d", snapshot.NumberOfNodes()))
	if s.ClusterVersion != nil {
		d.Field("Cluster Version", *s.ClusterVersion)
	}
	if s.EngineFullVersion != nil {
		d.Field("Engine Version", *s.EngineFullVersion)
	}

	// Encryption
	if s.Encrypted != nil && *s.Encrypted {
		d.Section("Encryption")
		d.Field("Encrypted", "Yes")
		if s.EncryptedWithHSM != nil && *s.EncryptedWithHSM {
			d.Field("HSM Encrypted", "Yes")
		}
		if s.KmsKeyId != nil {
			d.Field("KMS Key", *s.KmsKeyId)
		}
	}

	// Retention
	if s.ManualSnapshotRetentionPeriod != nil {
		d.Section("Retention")
		if *s.ManualSnapshotRetentionPeriod == -1 {
			d.Field("Retention", "Indefinite")
		} else {
			d.Field("Retention", fmt.Sprintf("%d days", *s.ManualSnapshotRetentionPeriod))
		}
		if s.ManualSnapshotRemainingDays != nil {
			d.Field("Days Remaining", fmt.Sprintf("%d", *s.ManualSnapshotRemainingDays))
		}
	}

	// VPC
	if s.VpcId != nil {
		d.Section("Network")
		d.Field("VPC ID", *s.VpcId)
		if s.EnhancedVpcRouting != nil && *s.EnhancedVpcRouting {
			d.Field("Enhanced VPC Routing", "Enabled")
		}
	}

	// Timestamps
	d.Section("Timestamps")
	if t := snapshot.CreatedAt(); t != nil {
		d.Field("Snapshot Created", t.Format("2006-01-02 15:04:05"))
	}
	if s.ClusterCreateTime != nil {
		d.Field("Cluster Created", s.ClusterCreateTime.Format("2006-01-02 15:04:05"))
	}

	return d.String()
}

// RenderSummary renders summary fields for a snapshot.
func (r *SnapshotRenderer) RenderSummary(resource dao.Resource) []render.SummaryField {
	snapshot, ok := resource.(*SnapshotResource)
	if !ok {
		return r.BaseRenderer.RenderSummary(resource)
	}

	return []render.SummaryField{
		{Label: "Snapshot ID", Value: snapshot.GetID()},
		{Label: "Status", Value: snapshot.Status()},
		{Label: "Type", Value: snapshot.SnapshotType()},
	}
}
