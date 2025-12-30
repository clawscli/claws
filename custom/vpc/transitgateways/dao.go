package transitgateways

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"

	appaws "github.com/clawscli/claws/internal/aws"
	"github.com/clawscli/claws/internal/dao"
	apperrors "github.com/clawscli/claws/internal/errors"
)

// TransitGatewayDAO provides data access for Transit Gateways.
type TransitGatewayDAO struct {
	dao.BaseDAO
	client *ec2.Client
}

// NewTransitGatewayDAO creates a new TransitGatewayDAO.
func NewTransitGatewayDAO(ctx context.Context) (dao.DAO, error) {
	cfg, err := appaws.NewConfig(ctx)
	if err != nil {
		return nil, apperrors.Wrap(err, "new vpc/transitgateways dao")
	}
	return &TransitGatewayDAO{
		BaseDAO: dao.NewBaseDAO("ec2", "transit-gateways"),
		client:  ec2.NewFromConfig(cfg),
	}, nil
}

// List returns all Transit Gateways.
func (d *TransitGatewayDAO) List(ctx context.Context) ([]dao.Resource, error) {
	tgws, err := appaws.Paginate(ctx, func(token *string) ([]types.TransitGateway, *string, error) {
		output, err := d.client.DescribeTransitGateways(ctx, &ec2.DescribeTransitGatewaysInput{
			NextToken: token,
		})
		if err != nil {
			return nil, nil, apperrors.Wrap(err, "describe transit gateways")
		}
		return output.TransitGateways, output.NextToken, nil
	})
	if err != nil {
		return nil, err
	}

	resources := make([]dao.Resource, len(tgws))
	for i, tgw := range tgws {
		resources[i] = NewTransitGatewayResource(tgw)
	}
	return resources, nil
}

// Get returns a specific Transit Gateway by ID.
func (d *TransitGatewayDAO) Get(ctx context.Context, id string) (dao.Resource, error) {
	output, err := d.client.DescribeTransitGateways(ctx, &ec2.DescribeTransitGatewaysInput{
		TransitGatewayIds: []string{id},
	})
	if err != nil {
		return nil, apperrors.Wrapf(err, "describe transit gateway %s", id)
	}
	if len(output.TransitGateways) == 0 {
		return nil, fmt.Errorf("transit gateway not found: %s", id)
	}
	return NewTransitGatewayResource(output.TransitGateways[0]), nil
}

// Delete deletes a Transit Gateway by ID.
func (d *TransitGatewayDAO) Delete(ctx context.Context, id string) error {
	_, err := d.client.DeleteTransitGateway(ctx, &ec2.DeleteTransitGatewayInput{
		TransitGatewayId: &id,
	})
	if err != nil {
		return apperrors.Wrapf(err, "delete transit gateway %s", id)
	}
	return nil
}

// TransitGatewayResource wraps a Transit Gateway.
type TransitGatewayResource struct {
	dao.BaseResource
	Item types.TransitGateway
}

// NewTransitGatewayResource creates a new TransitGatewayResource.
func NewTransitGatewayResource(tgw types.TransitGateway) *TransitGatewayResource {
	return &TransitGatewayResource{
		BaseResource: dao.BaseResource{
			ID:  appaws.Str(tgw.TransitGatewayId),
			ARN: appaws.Str(tgw.TransitGatewayArn),
		},
		Item: tgw,
	}
}

// State returns the transit gateway state.
func (r *TransitGatewayResource) State() string {
	return string(r.Item.State)
}

// OwnerId returns the owner account ID.
func (r *TransitGatewayResource) OwnerId() string {
	return appaws.Str(r.Item.OwnerId)
}

// Description returns the description.
func (r *TransitGatewayResource) Description() string {
	return appaws.Str(r.Item.Description)
}

// CreationTime returns when the TGW was created.
func (r *TransitGatewayResource) CreationTime() *time.Time {
	return r.Item.CreationTime
}

// DefaultRouteTableId returns the default route table ID.
func (r *TransitGatewayResource) DefaultRouteTableId() string {
	if r.Item.Options != nil {
		return appaws.Str(r.Item.Options.AssociationDefaultRouteTableId)
	}
	return ""
}

// AmazonSideAsn returns the Amazon side ASN.
func (r *TransitGatewayResource) AmazonSideAsn() int64 {
	if r.Item.Options != nil {
		return appaws.Int64(r.Item.Options.AmazonSideAsn)
	}
	return 0
}

// Name returns the Name tag value.
func (r *TransitGatewayResource) Name() string {
	for _, tag := range r.Item.Tags {
		if appaws.Str(tag.Key) == "Name" {
			return appaws.Str(tag.Value)
		}
	}
	return ""
}

// PropagationDefaultRouteTableId returns the propagation default route table ID.
func (r *TransitGatewayResource) PropagationDefaultRouteTableId() string {
	if r.Item.Options != nil {
		return appaws.Str(r.Item.Options.PropagationDefaultRouteTableId)
	}
	return ""
}

// AutoAcceptSharedAttachments returns if auto accept is enabled.
func (r *TransitGatewayResource) AutoAcceptSharedAttachments() string {
	if r.Item.Options != nil {
		return string(r.Item.Options.AutoAcceptSharedAttachments)
	}
	return ""
}

// DefaultRouteTableAssociation returns if default association is enabled.
func (r *TransitGatewayResource) DefaultRouteTableAssociation() string {
	if r.Item.Options != nil {
		return string(r.Item.Options.DefaultRouteTableAssociation)
	}
	return ""
}

// DefaultRouteTablePropagation returns if default propagation is enabled.
func (r *TransitGatewayResource) DefaultRouteTablePropagation() string {
	if r.Item.Options != nil {
		return string(r.Item.Options.DefaultRouteTablePropagation)
	}
	return ""
}

// VpnEcmpSupport returns if VPN ECMP support is enabled.
func (r *TransitGatewayResource) VpnEcmpSupport() string {
	if r.Item.Options != nil {
		return string(r.Item.Options.VpnEcmpSupport)
	}
	return ""
}

// DnsSupport returns if DNS support is enabled.
func (r *TransitGatewayResource) DnsSupport() string {
	if r.Item.Options != nil {
		return string(r.Item.Options.DnsSupport)
	}
	return ""
}

// MulticastSupport returns if multicast support is enabled.
func (r *TransitGatewayResource) MulticastSupport() string {
	if r.Item.Options != nil {
		return string(r.Item.Options.MulticastSupport)
	}
	return ""
}

// Tags returns all tags.
func (r *TransitGatewayResource) Tags() map[string]string {
	tags := make(map[string]string)
	for _, tag := range r.Item.Tags {
		tags[appaws.Str(tag.Key)] = appaws.Str(tag.Value)
	}
	return tags
}
