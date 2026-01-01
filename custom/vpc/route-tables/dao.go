package routetables

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"

	appaws "github.com/clawscli/claws/internal/aws"
	"github.com/clawscli/claws/internal/dao"
	apperrors "github.com/clawscli/claws/internal/errors"
)

// RouteTableDAO provides data access for Route Tables
type RouteTableDAO struct {
	dao.BaseDAO
	client *ec2.Client
}

// NewRouteTableDAO creates a new RouteTableDAO
func NewRouteTableDAO(ctx context.Context) (dao.DAO, error) {
	cfg, err := appaws.NewConfig(ctx)
	if err != nil {
		return nil, apperrors.Wrap(err, "new vpc/routetables dao")
	}
	return &RouteTableDAO{
		BaseDAO: dao.NewBaseDAO("vpc", "route-tables"),
		client:  ec2.NewFromConfig(cfg),
	}, nil
}

func (d *RouteTableDAO) List(ctx context.Context) ([]dao.Resource, error) {
	output, err := d.client.DescribeRouteTables(ctx, &ec2.DescribeRouteTablesInput{})
	if err != nil {
		return nil, apperrors.Wrap(err, "describe route tables")
	}

	var resources []dao.Resource
	for _, rt := range output.RouteTables {
		resources = append(resources, NewRouteTableResource(rt))
	}

	return resources, nil
}

func (d *RouteTableDAO) Get(ctx context.Context, id string) (dao.Resource, error) {
	output, err := d.client.DescribeRouteTables(ctx, &ec2.DescribeRouteTablesInput{
		RouteTableIds: []string{id},
	})
	if err != nil {
		return nil, apperrors.Wrapf(err, "describe route table %s", id)
	}

	if len(output.RouteTables) == 0 {
		return nil, fmt.Errorf("route table not found: %s", id)
	}

	return NewRouteTableResource(output.RouteTables[0]), nil
}

func (d *RouteTableDAO) Delete(ctx context.Context, id string) error {
	_, err := d.client.DeleteRouteTable(ctx, &ec2.DeleteRouteTableInput{
		RouteTableId: &id,
	})
	if err != nil {
		return apperrors.Wrapf(err, "delete route table %s", id)
	}
	return nil
}

// RouteTableResource wraps a Route Table
type RouteTableResource struct {
	dao.BaseResource
	Item types.RouteTable
}

// NewRouteTableResource creates a new RouteTableResource
func NewRouteTableResource(rt types.RouteTable) *RouteTableResource {
	return &RouteTableResource{
		BaseResource: dao.BaseResource{
			ID:   appaws.Str(rt.RouteTableId),
			Name: appaws.EC2NameTag(rt.Tags),
			Tags: appaws.TagsToMap(rt.Tags),
			Data: rt,
		},
		Item: rt,
	}
}

// VpcId returns the VPC ID
func (r *RouteTableResource) VpcId() string {
	if r.Item.VpcId != nil {
		return *r.Item.VpcId
	}
	return ""
}

// IsMain returns whether this is the main route table
func (r *RouteTableResource) IsMain() bool {
	for _, assoc := range r.Item.Associations {
		if assoc.Main != nil && *assoc.Main {
			return true
		}
	}
	return false
}

// RouteCount returns the number of routes
func (r *RouteTableResource) RouteCount() int {
	return len(r.Item.Routes)
}

// SubnetAssociationCount returns the number of subnet associations (excluding main)
func (r *RouteTableResource) SubnetAssociationCount() int {
	count := 0
	for _, assoc := range r.Item.Associations {
		if assoc.SubnetId != nil {
			count++
		}
	}
	return count
}
