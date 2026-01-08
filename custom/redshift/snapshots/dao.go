package snapshots

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/redshift"
	"github.com/aws/aws-sdk-go-v2/service/redshift/types"

	appaws "github.com/clawscli/claws/internal/aws"
	"github.com/clawscli/claws/internal/dao"
	apperrors "github.com/clawscli/claws/internal/errors"
)

// SnapshotDAO provides data access for Redshift snapshots.
type SnapshotDAO struct {
	dao.BaseDAO
	client *redshift.Client
}

// NewSnapshotDAO creates a new SnapshotDAO.
func NewSnapshotDAO(ctx context.Context) (dao.DAO, error) {
	cfg, err := appaws.NewConfig(ctx)
	if err != nil {
		return nil, apperrors.Wrap(err, "new "+ServiceResourcePath+" dao")
	}
	return &SnapshotDAO{
		BaseDAO: dao.NewBaseDAO("redshift", "snapshots"),
		client:  redshift.NewFromConfig(cfg),
	}, nil
}

// List returns snapshots for the specified cluster.
func (d *SnapshotDAO) List(ctx context.Context) ([]dao.Resource, error) {
	clusterIdentifier := dao.GetFilterFromContext(ctx, "ClusterIdentifier")

	var input *redshift.DescribeClusterSnapshotsInput
	if clusterIdentifier != "" {
		input = &redshift.DescribeClusterSnapshotsInput{
			ClusterIdentifier: &clusterIdentifier,
		}
	} else {
		input = &redshift.DescribeClusterSnapshotsInput{}
	}

	snapshots, err := appaws.Paginate(ctx, func(token *string) ([]types.Snapshot, *string, error) {
		input.Marker = token
		output, err := d.client.DescribeClusterSnapshots(ctx, input)
		if err != nil {
			return nil, nil, apperrors.Wrap(err, "describe redshift snapshots")
		}
		return output.Snapshots, output.Marker, nil
	})
	if err != nil {
		return nil, err
	}

	resources := make([]dao.Resource, len(snapshots))
	for i, snapshot := range snapshots {
		resources[i] = NewSnapshotResource(snapshot)
	}
	return resources, nil
}

// Get returns a specific snapshot.
func (d *SnapshotDAO) Get(ctx context.Context, id string) (dao.Resource, error) {
	output, err := d.client.DescribeClusterSnapshots(ctx, &redshift.DescribeClusterSnapshotsInput{
		SnapshotIdentifier: &id,
	})
	if err != nil {
		return nil, apperrors.Wrap(err, "describe redshift snapshot")
	}
	if len(output.Snapshots) == 0 {
		return nil, fmt.Errorf("snapshot not found: %s", id)
	}
	return NewSnapshotResource(output.Snapshots[0]), nil
}

// Delete deletes a snapshot.
func (d *SnapshotDAO) Delete(ctx context.Context, id string) error {
	_, err := d.client.DeleteClusterSnapshot(ctx, &redshift.DeleteClusterSnapshotInput{
		SnapshotIdentifier: &id,
	})
	if err != nil {
		return apperrors.Wrap(err, "delete redshift snapshot")
	}
	return nil
}

// SnapshotResource wraps a Redshift snapshot.
type SnapshotResource struct {
	dao.BaseResource
	Snapshot *types.Snapshot
}

// NewSnapshotResource creates a new SnapshotResource.
func NewSnapshotResource(snapshot types.Snapshot) *SnapshotResource {
	return &SnapshotResource{
		BaseResource: dao.BaseResource{
			ID:   appaws.Str(snapshot.SnapshotIdentifier),
			ARN:  "",
			Data: snapshot,
		},
		Snapshot: &snapshot,
	}
}

// ClusterIdentifier returns the cluster identifier.
func (r *SnapshotResource) ClusterIdentifier() string {
	if r.Snapshot != nil && r.Snapshot.ClusterIdentifier != nil {
		return *r.Snapshot.ClusterIdentifier
	}
	return ""
}

// Status returns the snapshot status.
func (r *SnapshotResource) Status() string {
	if r.Snapshot != nil && r.Snapshot.Status != nil {
		return *r.Snapshot.Status
	}
	return ""
}

// SnapshotType returns the snapshot type.
func (r *SnapshotResource) SnapshotType() string {
	if r.Snapshot != nil && r.Snapshot.SnapshotType != nil {
		return *r.Snapshot.SnapshotType
	}
	return ""
}

// NodeType returns the node type.
func (r *SnapshotResource) NodeType() string {
	if r.Snapshot != nil && r.Snapshot.NodeType != nil {
		return *r.Snapshot.NodeType
	}
	return ""
}

// NumberOfNodes returns the number of nodes.
func (r *SnapshotResource) NumberOfNodes() int32 {
	if r.Snapshot != nil && r.Snapshot.NumberOfNodes != nil {
		return *r.Snapshot.NumberOfNodes
	}
	return 0
}

// TotalBackupSize returns the total backup size in megabytes.
func (r *SnapshotResource) TotalBackupSize() float64 {
	if r.Snapshot != nil && r.Snapshot.TotalBackupSizeInMegaBytes != nil {
		return *r.Snapshot.TotalBackupSizeInMegaBytes
	}
	return 0
}

// CreatedAt returns when the snapshot was created.
func (r *SnapshotResource) CreatedAt() *time.Time {
	if r.Snapshot != nil {
		return r.Snapshot.SnapshotCreateTime
	}
	return nil
}
