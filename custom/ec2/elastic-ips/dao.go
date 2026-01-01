package elasticips

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"

	appaws "github.com/clawscli/claws/internal/aws"
	"github.com/clawscli/claws/internal/dao"
	apperrors "github.com/clawscli/claws/internal/errors"
)

// ElasticIPDAO provides data access for Elastic IPs
type ElasticIPDAO struct {
	dao.BaseDAO
	client *ec2.Client
}

// NewElasticIPDAO creates a new ElasticIPDAO
func NewElasticIPDAO(ctx context.Context) (dao.DAO, error) {
	cfg, err := appaws.NewConfig(ctx)
	if err != nil {
		return nil, apperrors.Wrap(err, "new ec2/elasticips dao")
	}
	return &ElasticIPDAO{
		BaseDAO: dao.NewBaseDAO("ec2", "elastic-ips"),
		client:  ec2.NewFromConfig(cfg),
	}, nil
}

func (d *ElasticIPDAO) List(ctx context.Context) ([]dao.Resource, error) {
	output, err := d.client.DescribeAddresses(ctx, &ec2.DescribeAddressesInput{})
	if err != nil {
		return nil, apperrors.Wrap(err, "describe addresses")
	}

	var resources []dao.Resource
	for _, addr := range output.Addresses {
		resources = append(resources, NewElasticIPResource(addr))
	}

	return resources, nil
}

func (d *ElasticIPDAO) Get(ctx context.Context, id string) (dao.Resource, error) {
	output, err := d.client.DescribeAddresses(ctx, &ec2.DescribeAddressesInput{
		AllocationIds: []string{id},
	})
	if err != nil {
		return nil, apperrors.Wrapf(err, "describe address %s", id)
	}

	if len(output.Addresses) == 0 {
		return nil, fmt.Errorf("elastic ip not found: %s", id)
	}

	return NewElasticIPResource(output.Addresses[0]), nil
}

func (d *ElasticIPDAO) Delete(ctx context.Context, id string) error {
	_, err := d.client.ReleaseAddress(ctx, &ec2.ReleaseAddressInput{
		AllocationId: &id,
	})
	if err != nil {
		if apperrors.IsNotFound(err) {
			return nil // Already deleted
		}
		if apperrors.IsResourceInUse(err) {
			return apperrors.Wrapf(err, "elastic IP %s is associated with an instance or network interface", id)
		}
		return apperrors.Wrapf(err, "release address %s", id)
	}

	return nil
}

// ElasticIPResource wraps an Elastic IP address
type ElasticIPResource struct {
	dao.BaseResource
	Item types.Address
}

// NewElasticIPResource creates a new ElasticIPResource
func NewElasticIPResource(addr types.Address) *ElasticIPResource {
	return &ElasticIPResource{
		BaseResource: dao.BaseResource{
			ID:   appaws.Str(addr.AllocationId),
			Name: appaws.EC2NameTag(addr.Tags),
			Tags: appaws.TagsToMap(addr.Tags),
			Data: addr,
		},
		Item: addr,
	}
}

func (r *ElasticIPResource) PublicIP() string {
	if r.Item.PublicIp != nil {
		return *r.Item.PublicIp
	}
	return ""
}

func (r *ElasticIPResource) PrivateIP() string {
	if r.Item.PrivateIpAddress != nil {
		return *r.Item.PrivateIpAddress
	}
	return ""
}

func (r *ElasticIPResource) InstanceId() string {
	if r.Item.InstanceId != nil {
		return *r.Item.InstanceId
	}
	return ""
}

func (r *ElasticIPResource) AssociationId() string {
	if r.Item.AssociationId != nil {
		return *r.Item.AssociationId
	}
	return ""
}

func (r *ElasticIPResource) NetworkInterfaceId() string {
	if r.Item.NetworkInterfaceId != nil {
		return *r.Item.NetworkInterfaceId
	}
	return ""
}

func (r *ElasticIPResource) Domain() string {
	return string(r.Item.Domain)
}

func (r *ElasticIPResource) NetworkInterfaceOwnerId() string {
	if r.Item.NetworkInterfaceOwnerId != nil {
		return *r.Item.NetworkInterfaceOwnerId
	}
	return ""
}
