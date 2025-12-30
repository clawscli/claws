package snapshots

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/rds"
	"github.com/aws/aws-sdk-go-v2/service/rds/types"

	appaws "github.com/clawscli/claws/internal/aws"
	"github.com/clawscli/claws/internal/dao"
)

// SnapshotDAO provides data access for RDS snapshots
type SnapshotDAO struct {
	dao.BaseDAO
	client *rds.Client
}

// NewSnapshotDAO creates a new SnapshotDAO
func NewSnapshotDAO(ctx context.Context) (dao.DAO, error) {
	cfg, err := appaws.NewConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("new rds/snapshots dao: %w", err)
	}
	return &SnapshotDAO{
		BaseDAO: dao.NewBaseDAO("rds", "snapshots"),
		client:  rds.NewFromConfig(cfg),
	}, nil
}

// List returns snapshots (first page only for backwards compatibility).
// For paginated access, use ListPage instead.
func (d *SnapshotDAO) List(ctx context.Context) ([]dao.Resource, error) {
	resources, _, err := d.ListPage(ctx, 100, "")
	return resources, err
}

// ListPage returns a page of RDS snapshots.
// Implements dao.PaginatedDAO interface.
func (d *SnapshotDAO) ListPage(ctx context.Context, pageSize int, pageToken string) ([]dao.Resource, string, error) {
	maxRecords := int32(pageSize)
	if maxRecords > 100 {
		maxRecords = 100 // AWS API max
	}

	input := &rds.DescribeDBSnapshotsInput{
		MaxRecords: &maxRecords,
	}
	if pageToken != "" {
		input.Marker = &pageToken
	}

	output, err := d.client.DescribeDBSnapshots(ctx, input)
	if err != nil {
		return nil, "", fmt.Errorf("describe db snapshots: %w", err)
	}

	resources := make([]dao.Resource, len(output.DBSnapshots))
	for i, snapshot := range output.DBSnapshots {
		resources[i] = NewSnapshotResource(snapshot)
	}

	nextToken := ""
	if output.Marker != nil {
		nextToken = *output.Marker
	}

	return resources, nextToken, nil
}

func (d *SnapshotDAO) Get(ctx context.Context, id string) (dao.Resource, error) {
	input := &rds.DescribeDBSnapshotsInput{
		DBSnapshotIdentifier: &id,
	}

	output, err := d.client.DescribeDBSnapshots(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("describe db snapshot %s: %w", id, err)
	}

	if len(output.DBSnapshots) == 0 {
		return nil, fmt.Errorf("db snapshot not found: %s", id)
	}

	return NewSnapshotResource(output.DBSnapshots[0]), nil
}

func (d *SnapshotDAO) Delete(ctx context.Context, id string) error {
	input := &rds.DeleteDBSnapshotInput{
		DBSnapshotIdentifier: &id,
	}

	_, err := d.client.DeleteDBSnapshot(ctx, input)
	if err != nil {
		if appaws.IsNotFound(err) {
			return nil // Already deleted
		}
		if appaws.IsResourceInUse(err) {
			return fmt.Errorf("db snapshot %s is in use", id)
		}
		return fmt.Errorf("delete db snapshot %s: %w", id, err)
	}

	return nil
}

// SnapshotResource wraps an RDS snapshot
type SnapshotResource struct {
	dao.BaseResource
	Item types.DBSnapshot
}

// NewSnapshotResource creates a new SnapshotResource
func NewSnapshotResource(snapshot types.DBSnapshot) *SnapshotResource {
	return &SnapshotResource{
		BaseResource: dao.BaseResource{
			ID:   appaws.Str(snapshot.DBSnapshotIdentifier),
			Name: appaws.Str(snapshot.DBSnapshotIdentifier),
			ARN:  appaws.Str(snapshot.DBSnapshotArn),
			Tags: appaws.TagsToMap(snapshot.TagList),
			Data: snapshot,
		},
		Item: snapshot,
	}
}

// State returns the snapshot status
func (r *SnapshotResource) State() string {
	if r.Item.Status != nil {
		return *r.Item.Status
	}
	return "unknown"
}

// Engine returns the database engine
func (r *SnapshotResource) Engine() string {
	if r.Item.Engine != nil {
		return *r.Item.Engine
	}
	return ""
}

// EngineVersion returns the engine version
func (r *SnapshotResource) EngineVersion() string {
	if r.Item.EngineVersion != nil {
		return *r.Item.EngineVersion
	}
	return ""
}

// InstanceIdentifier returns the source DB instance identifier
func (r *SnapshotResource) InstanceIdentifier() string {
	if r.Item.DBInstanceIdentifier != nil {
		return *r.Item.DBInstanceIdentifier
	}
	return ""
}

// SnapshotType returns the snapshot type
func (r *SnapshotResource) SnapshotType() string {
	if r.Item.SnapshotType != nil {
		return *r.Item.SnapshotType
	}
	return ""
}

// AllocatedStorage returns the allocated storage in GB
func (r *SnapshotResource) AllocatedStorage() int32 {
	if r.Item.AllocatedStorage != nil {
		return *r.Item.AllocatedStorage
	}
	return 0
}
