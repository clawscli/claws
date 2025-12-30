package internetgateways

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"

	appaws "github.com/clawscli/claws/internal/aws"
	"github.com/clawscli/claws/internal/dao"
	apperrors "github.com/clawscli/claws/internal/errors"
)

// InternetGatewayDAO provides data access for Internet Gateways
type InternetGatewayDAO struct {
	dao.BaseDAO
	client *ec2.Client
}

// NewInternetGatewayDAO creates a new InternetGatewayDAO
func NewInternetGatewayDAO(ctx context.Context) (dao.DAO, error) {
	cfg, err := appaws.NewConfig(ctx)
	if err != nil {
		return nil, apperrors.Wrap(err, "new vpc/internetgateways dao")
	}
	return &InternetGatewayDAO{
		BaseDAO: dao.NewBaseDAO("vpc", "internet-gateways"),
		client:  ec2.NewFromConfig(cfg),
	}, nil
}

func (d *InternetGatewayDAO) List(ctx context.Context) ([]dao.Resource, error) {
	input := &ec2.DescribeInternetGatewaysInput{}

	// Filter by VPC ID if provided
	if vpcID := dao.GetFilterFromContext(ctx, "VpcId"); vpcID != "" {
		input.Filters = []types.Filter{
			{
				Name:   appaws.StringPtr("attachment.vpc-id"),
				Values: []string{vpcID},
			},
		}
	}

	output, err := d.client.DescribeInternetGateways(ctx, input)
	if err != nil {
		return nil, apperrors.Wrap(err, "describe internet gateways")
	}

	var resources []dao.Resource
	for _, igw := range output.InternetGateways {
		resources = append(resources, NewInternetGatewayResource(igw))
	}

	return resources, nil
}

func (d *InternetGatewayDAO) Get(ctx context.Context, id string) (dao.Resource, error) {
	output, err := d.client.DescribeInternetGateways(ctx, &ec2.DescribeInternetGatewaysInput{
		InternetGatewayIds: []string{id},
	})
	if err != nil {
		return nil, apperrors.Wrapf(err, "describe internet gateway %s", id)
	}

	if len(output.InternetGateways) == 0 {
		return nil, fmt.Errorf("internet gateway not found: %s", id)
	}

	return NewInternetGatewayResource(output.InternetGateways[0]), nil
}

func (d *InternetGatewayDAO) Delete(ctx context.Context, id string) error {
	_, err := d.client.DeleteInternetGateway(ctx, &ec2.DeleteInternetGatewayInput{
		InternetGatewayId: &id,
	})
	if err != nil {
		if apperrors.IsNotFound(err) {
			return nil // Already deleted
		}
		if apperrors.IsResourceInUse(err) {
			return apperrors.Wrapf(err, "internet gateway %s is in use (must be detached first)", id)
		}
		return apperrors.Wrapf(err, "delete internet gateway %s", id)
	}
	return nil
}

// InternetGatewayResource wraps an Internet Gateway
type InternetGatewayResource struct {
	dao.BaseResource
	Item types.InternetGateway
}

// NewInternetGatewayResource creates a new InternetGatewayResource
func NewInternetGatewayResource(igw types.InternetGateway) *InternetGatewayResource {
	return &InternetGatewayResource{
		BaseResource: dao.BaseResource{
			ID:   appaws.Str(igw.InternetGatewayId),
			Name: appaws.EC2NameTag(igw.Tags),
			Tags: appaws.TagsToMap(igw.Tags),
			Data: igw,
		},
		Item: igw,
	}
}

// AttachedVpcId returns the attached VPC ID
func (r *InternetGatewayResource) AttachedVpcId() string {
	for _, attach := range r.Item.Attachments {
		if attach.VpcId != nil {
			return *attach.VpcId
		}
	}
	return ""
}

// AttachmentState returns the attachment state
func (r *InternetGatewayResource) AttachmentState() string {
	for _, attach := range r.Item.Attachments {
		return string(attach.State)
	}
	return "detached"
}

// OwnerId returns the owner ID
func (r *InternetGatewayResource) OwnerId() string {
	if r.Item.OwnerId != nil {
		return *r.Item.OwnerId
	}
	return ""
}
