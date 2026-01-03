package snapshots

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"

	appaws "github.com/clawscli/claws/internal/aws"
	"github.com/clawscli/claws/internal/dao"
	apperrors "github.com/clawscli/claws/internal/errors"
)

// SnapshotDAO provides data access for EBS snapshots
type SnapshotDAO struct {
	dao.BaseDAO
	client *ec2.Client
}

// NewSnapshotDAO creates a new SnapshotDAO
func NewSnapshotDAO(ctx context.Context) (dao.DAO, error) {
	cfg, err := appaws.NewConfig(ctx)
	if err != nil {
		return nil, apperrors.Wrap(err, "new "+ServiceResourcePath+" dao")
	}
	return &SnapshotDAO{
		BaseDAO: dao.NewBaseDAO("ec2", "snapshots"),
		client:  ec2.NewFromConfig(cfg),
	}, nil
}

// List returns snapshots (first page only for backwards compatibility).
// For paginated access, use ListPage instead.
func (d *SnapshotDAO) List(ctx context.Context) ([]dao.Resource, error) {
	resources, _, err := d.ListPage(ctx, 100, "")
	return resources, err
}

// ListPage returns a page of EBS snapshots.
// Implements dao.PaginatedDAO interface.
func (d *SnapshotDAO) ListPage(ctx context.Context, pageSize int, pageToken string) ([]dao.Resource, string, error) {
	// By default, only show owned snapshots
	self := "self"
	maxResults := int32(pageSize)
	if maxResults > 1000 {
		maxResults = 1000 // AWS API max
	}

	input := &ec2.DescribeSnapshotsInput{
		OwnerIds:   []string{self},
		MaxResults: &maxResults,
	}
	if pageToken != "" {
		input.NextToken = &pageToken
	}

	output, err := d.client.DescribeSnapshots(ctx, input)
	if err != nil {
		return nil, "", apperrors.Wrap(err, "describe snapshots")
	}

	resources := make([]dao.Resource, len(output.Snapshots))
	for i, snap := range output.Snapshots {
		resources[i] = NewSnapshotResource(snap)
	}

	nextToken := ""
	if output.NextToken != nil {
		nextToken = *output.NextToken
	}

	return resources, nextToken, nil
}

func (d *SnapshotDAO) Get(ctx context.Context, id string) (dao.Resource, error) {
	input := &ec2.DescribeSnapshotsInput{
		SnapshotIds: []string{id},
	}

	output, err := d.client.DescribeSnapshots(ctx, input)
	if err != nil {
		return nil, apperrors.Wrapf(err, "describe snapshot %s", id)
	}

	if len(output.Snapshots) == 0 {
		return nil, fmt.Errorf("snapshot not found: %s", id)
	}

	return NewSnapshotResource(output.Snapshots[0]), nil
}

func (d *SnapshotDAO) Delete(ctx context.Context, id string) error {
	input := &ec2.DeleteSnapshotInput{
		SnapshotId: &id,
	}

	_, err := d.client.DeleteSnapshot(ctx, input)
	if err != nil {
		if apperrors.IsNotFound(err) {
			return nil // Already deleted
		}
		if apperrors.IsResourceInUse(err) {
			return apperrors.Wrapf(err, "snapshot %s is in use", id)
		}
		return apperrors.Wrapf(err, "delete snapshot %s", id)
	}

	return nil
}

// SnapshotResource wraps an EBS snapshot
type SnapshotResource struct {
	dao.BaseResource
	Item types.Snapshot
}

// NewSnapshotResource creates a new SnapshotResource
func NewSnapshotResource(snap types.Snapshot) *SnapshotResource {
	return &SnapshotResource{
		BaseResource: dao.BaseResource{
			ID:   appaws.Str(snap.SnapshotId),
			Name: appaws.EC2NameTag(snap.Tags),
			Tags: appaws.TagsToMap(snap.Tags),
			Data: snap,
		},
		Item: snap,
	}
}

func (r *SnapshotResource) State() string {
	return string(r.Item.State)
}

func (r *SnapshotResource) Progress() string {
	if r.Item.Progress != nil {
		return *r.Item.Progress
	}
	return ""
}

func (r *SnapshotResource) VolumeSize() int32 {
	if r.Item.VolumeSize != nil {
		return *r.Item.VolumeSize
	}
	return 0
}

func (r *SnapshotResource) VolumeId() string {
	if r.Item.VolumeId != nil {
		return *r.Item.VolumeId
	}
	return ""
}

func (r *SnapshotResource) Encrypted() bool {
	if r.Item.Encrypted != nil {
		return *r.Item.Encrypted
	}
	return false
}

func (r *SnapshotResource) Description() string {
	if r.Item.Description != nil {
		return *r.Item.Description
	}
	return ""
}

func (r *SnapshotResource) OwnerId() string {
	if r.Item.OwnerId != nil {
		return *r.Item.OwnerId
	}
	return ""
}
