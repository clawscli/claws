package natgateways

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"

	appaws "github.com/clawscli/claws/internal/aws"
	"github.com/clawscli/claws/internal/dao"
	apperrors "github.com/clawscli/claws/internal/errors"
)

// NatGatewayDAO provides data access for NAT Gateways
type NatGatewayDAO struct {
	dao.BaseDAO
	client *ec2.Client
}

// NewNatGatewayDAO creates a new NatGatewayDAO
func NewNatGatewayDAO(ctx context.Context) (dao.DAO, error) {
	cfg, err := appaws.NewConfig(ctx)
	if err != nil {
		return nil, apperrors.Wrap(err, "new vpc/natgateways dao")
	}
	return &NatGatewayDAO{
		BaseDAO: dao.NewBaseDAO("vpc", "nat-gateways"),
		client:  ec2.NewFromConfig(cfg),
	}, nil
}

func (d *NatGatewayDAO) List(ctx context.Context) ([]dao.Resource, error) {
	paginator := ec2.NewDescribeNatGatewaysPaginator(d.client, &ec2.DescribeNatGatewaysInput{})

	var resources []dao.Resource
	for paginator.HasMorePages() {
		output, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, apperrors.Wrap(err, "describe nat gateways")
		}

		for _, ngw := range output.NatGateways {
			resources = append(resources, NewNatGatewayResource(ngw))
		}
	}

	return resources, nil
}

func (d *NatGatewayDAO) Get(ctx context.Context, id string) (dao.Resource, error) {
	output, err := d.client.DescribeNatGateways(ctx, &ec2.DescribeNatGatewaysInput{
		NatGatewayIds: []string{id},
	})
	if err != nil {
		return nil, apperrors.Wrapf(err, "describe nat gateway %s", id)
	}

	if len(output.NatGateways) == 0 {
		return nil, fmt.Errorf("nat gateway not found: %s", id)
	}

	return NewNatGatewayResource(output.NatGateways[0]), nil
}

func (d *NatGatewayDAO) Delete(ctx context.Context, id string) error {
	_, err := d.client.DeleteNatGateway(ctx, &ec2.DeleteNatGatewayInput{
		NatGatewayId: &id,
	})
	if err != nil {
		if apperrors.IsNotFound(err) {
			return nil // Already deleted
		}
		if apperrors.IsResourceInUse(err) {
			return apperrors.Wrapf(err, "nat gateway %s is in use", id)
		}
		return apperrors.Wrapf(err, "delete nat gateway %s", id)
	}
	return nil
}

// NatGatewayResource wraps a NAT Gateway
type NatGatewayResource struct {
	dao.BaseResource
	Item types.NatGateway
}

// NewNatGatewayResource creates a new NatGatewayResource
func NewNatGatewayResource(ngw types.NatGateway) *NatGatewayResource {
	return &NatGatewayResource{
		BaseResource: dao.BaseResource{
			ID:   appaws.Str(ngw.NatGatewayId),
			Name: appaws.EC2NameTag(ngw.Tags),
			Tags: appaws.TagsToMap(ngw.Tags),
			Data: ngw,
		},
		Item: ngw,
	}
}

// State returns the NAT gateway state
func (r *NatGatewayResource) State() string {
	return string(r.Item.State)
}

// ConnectivityType returns the connectivity type (public/private)
func (r *NatGatewayResource) ConnectivityType() string {
	return string(r.Item.ConnectivityType)
}

// VpcId returns the VPC ID
func (r *NatGatewayResource) VpcId() string {
	if r.Item.VpcId != nil {
		return *r.Item.VpcId
	}
	return ""
}

// SubnetId returns the subnet ID
func (r *NatGatewayResource) SubnetId() string {
	if r.Item.SubnetId != nil {
		return *r.Item.SubnetId
	}
	return ""
}

// PublicIp returns the public IP address
func (r *NatGatewayResource) PublicIp() string {
	for _, addr := range r.Item.NatGatewayAddresses {
		if addr.PublicIp != nil {
			return *addr.PublicIp
		}
	}
	return ""
}

// PrivateIp returns the private IP address
func (r *NatGatewayResource) PrivateIp() string {
	for _, addr := range r.Item.NatGatewayAddresses {
		if addr.PrivateIp != nil {
			return *addr.PrivateIp
		}
	}
	return ""
}
