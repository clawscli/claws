package images

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"

	appaws "github.com/clawscli/claws/internal/aws"
	"github.com/clawscli/claws/internal/dao"
)

// ImageDAO provides data access for EC2 AMIs
type ImageDAO struct {
	dao.BaseDAO
	client *ec2.Client
}

// NewImageDAO creates a new ImageDAO
func NewImageDAO(ctx context.Context) (dao.DAO, error) {
	cfg, err := appaws.NewConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("new ec2/images dao: %w", err)
	}
	return &ImageDAO{
		BaseDAO: dao.NewBaseDAO("ec2", "images"),
		client:  ec2.NewFromConfig(cfg),
	}, nil
}

func (d *ImageDAO) List(ctx context.Context) ([]dao.Resource, error) {
	// By default, only show owned images
	self := "self"
	input := &ec2.DescribeImagesInput{
		Owners: []string{self},
	}

	output, err := d.client.DescribeImages(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("describe images: %w", err)
	}

	var resources []dao.Resource
	for _, img := range output.Images {
		resources = append(resources, NewImageResource(img))
	}

	return resources, nil
}

func (d *ImageDAO) Get(ctx context.Context, id string) (dao.Resource, error) {
	input := &ec2.DescribeImagesInput{
		ImageIds: []string{id},
	}

	output, err := d.client.DescribeImages(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("describe image %s: %w", id, err)
	}

	if len(output.Images) == 0 {
		return nil, fmt.Errorf("image not found: %s", id)
	}

	return NewImageResource(output.Images[0]), nil
}

func (d *ImageDAO) Delete(ctx context.Context, id string) error {
	input := &ec2.DeregisterImageInput{
		ImageId: &id,
	}

	_, err := d.client.DeregisterImage(ctx, input)
	if err != nil {
		return fmt.Errorf("deregister image %s: %w", id, err)
	}

	return nil
}

// ImageResource wraps an EC2 AMI
type ImageResource struct {
	dao.BaseResource
	Item types.Image
}

// NewImageResource creates a new ImageResource
func NewImageResource(img types.Image) *ImageResource {
	// Try Name field first, then fall back to tags
	name := appaws.Str(img.Name)
	if name == "" {
		name = appaws.EC2NameTag(img.Tags)
	}

	return &ImageResource{
		BaseResource: dao.BaseResource{
			ID:   appaws.Str(img.ImageId),
			Name: name,
			Tags: appaws.TagsToMap(img.Tags),
			Data: img,
		},
		Item: img,
	}
}

func (r *ImageResource) State() string {
	return string(r.Item.State)
}

func (r *ImageResource) ImageType() string {
	return string(r.Item.ImageType)
}

func (r *ImageResource) Architecture() string {
	return string(r.Item.Architecture)
}

func (r *ImageResource) Platform() string {
	if r.Item.PlatformDetails != nil {
		return *r.Item.PlatformDetails
	}
	return ""
}

func (r *ImageResource) RootDeviceType() string {
	return string(r.Item.RootDeviceType)
}

func (r *ImageResource) CreationDate() string {
	if r.Item.CreationDate != nil {
		return *r.Item.CreationDate
	}
	return ""
}

func (r *ImageResource) Description() string {
	if r.Item.Description != nil {
		return *r.Item.Description
	}
	return ""
}

func (r *ImageResource) Public() bool {
	if r.Item.Public != nil {
		return *r.Item.Public
	}
	return false
}

func (r *ImageResource) OwnerId() string {
	if r.Item.OwnerId != nil {
		return *r.Item.OwnerId
	}
	return ""
}

func (r *ImageResource) RootDeviceName() string {
	if r.Item.RootDeviceName != nil {
		return *r.Item.RootDeviceName
	}
	return ""
}

func (r *ImageResource) VirtualizationType() string {
	return string(r.Item.VirtualizationType)
}
