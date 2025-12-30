package volumes

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"

	appaws "github.com/clawscli/claws/internal/aws"
	"github.com/clawscli/claws/internal/dao"
)

// VolumeDAO provides data access for EBS volumes
type VolumeDAO struct {
	dao.BaseDAO
	client *ec2.Client
}

// NewVolumeDAO creates a new VolumeDAO
func NewVolumeDAO(ctx context.Context) (dao.DAO, error) {
	cfg, err := appaws.NewConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("new ec2/volumes dao: %w", err)
	}
	return &VolumeDAO{
		BaseDAO: dao.NewBaseDAO("ec2", "volumes"),
		client:  ec2.NewFromConfig(cfg),
	}, nil
}

func (d *VolumeDAO) List(ctx context.Context) ([]dao.Resource, error) {
	input := &ec2.DescribeVolumesInput{}
	paginator := ec2.NewDescribeVolumesPaginator(d.client, input)

	var resources []dao.Resource
	for paginator.HasMorePages() {
		output, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("describe volumes: %w", err)
		}

		for _, vol := range output.Volumes {
			resources = append(resources, NewVolumeResource(vol))
		}
	}

	return resources, nil
}

func (d *VolumeDAO) Get(ctx context.Context, id string) (dao.Resource, error) {
	input := &ec2.DescribeVolumesInput{
		VolumeIds: []string{id},
	}

	output, err := d.client.DescribeVolumes(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("describe volume %s: %w", id, err)
	}

	if len(output.Volumes) == 0 {
		return nil, fmt.Errorf("volume not found: %s", id)
	}

	return NewVolumeResource(output.Volumes[0]), nil
}

func (d *VolumeDAO) Delete(ctx context.Context, id string) error {
	input := &ec2.DeleteVolumeInput{
		VolumeId: &id,
	}

	_, err := d.client.DeleteVolume(ctx, input)
	if err != nil {
		if appaws.IsNotFound(err) {
			return nil // Already deleted
		}
		if appaws.IsResourceInUse(err) {
			return fmt.Errorf("volume %s is attached to an instance", id)
		}
		return fmt.Errorf("delete volume %s: %w", id, err)
	}

	return nil
}

// VolumeResource wraps an EBS volume
type VolumeResource struct {
	dao.BaseResource
	Item types.Volume
}

// NewVolumeResource creates a new VolumeResource
func NewVolumeResource(vol types.Volume) *VolumeResource {
	return &VolumeResource{
		BaseResource: dao.BaseResource{
			ID:   appaws.Str(vol.VolumeId),
			Name: appaws.EC2NameTag(vol.Tags),
			Tags: appaws.TagsToMap(vol.Tags),
			Data: vol,
		},
		Item: vol,
	}
}

func (r *VolumeResource) State() string {
	return string(r.Item.State)
}

func (r *VolumeResource) Size() int32 {
	if r.Item.Size != nil {
		return *r.Item.Size
	}
	return 0
}

func (r *VolumeResource) VolumeType() string {
	return string(r.Item.VolumeType)
}

func (r *VolumeResource) AZ() string {
	if r.Item.AvailabilityZone != nil {
		return *r.Item.AvailabilityZone
	}
	return ""
}

func (r *VolumeResource) AttachedInstance() string {
	if len(r.Item.Attachments) > 0 && r.Item.Attachments[0].InstanceId != nil {
		return *r.Item.Attachments[0].InstanceId
	}
	return ""
}

func (r *VolumeResource) Encrypted() bool {
	if r.Item.Encrypted != nil {
		return *r.Item.Encrypted
	}
	return false
}

func (r *VolumeResource) IOPS() int32 {
	if r.Item.Iops != nil {
		return *r.Item.Iops
	}
	return 0
}
