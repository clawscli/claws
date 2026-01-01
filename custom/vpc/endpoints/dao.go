package vpcendpoints

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

// VpcEndpointDAO provides data access for VPC Endpoints.
type VpcEndpointDAO struct {
	dao.BaseDAO
	client *ec2.Client
}

// NewVpcEndpointDAO creates a new VpcEndpointDAO.
func NewVpcEndpointDAO(ctx context.Context) (dao.DAO, error) {
	cfg, err := appaws.NewConfig(ctx)
	if err != nil {
		return nil, apperrors.Wrap(err, "new vpc/endpoints dao")
	}
	return &VpcEndpointDAO{
		BaseDAO: dao.NewBaseDAO("vpc", "endpoints"),
		client:  ec2.NewFromConfig(cfg),
	}, nil
}

// List returns all VPC endpoints.
func (d *VpcEndpointDAO) List(ctx context.Context) ([]dao.Resource, error) {
	endpoints, err := appaws.Paginate(ctx, func(token *string) ([]types.VpcEndpoint, *string, error) {
		output, err := d.client.DescribeVpcEndpoints(ctx, &ec2.DescribeVpcEndpointsInput{
			NextToken: token,
		})
		if err != nil {
			return nil, nil, apperrors.Wrap(err, "describe vpc endpoints")
		}
		return output.VpcEndpoints, output.NextToken, nil
	})
	if err != nil {
		return nil, err
	}

	resources := make([]dao.Resource, len(endpoints))
	for i, endpoint := range endpoints {
		resources[i] = NewVpcEndpointResource(endpoint)
	}
	return resources, nil
}

// Get returns a specific VPC endpoint by ID.
func (d *VpcEndpointDAO) Get(ctx context.Context, id string) (dao.Resource, error) {
	output, err := d.client.DescribeVpcEndpoints(ctx, &ec2.DescribeVpcEndpointsInput{
		VpcEndpointIds: []string{id},
	})
	if err != nil {
		return nil, apperrors.Wrapf(err, "describe vpc endpoint %s", id)
	}
	if len(output.VpcEndpoints) == 0 {
		return nil, fmt.Errorf("vpc endpoint not found: %s", id)
	}
	return NewVpcEndpointResource(output.VpcEndpoints[0]), nil
}

// Delete deletes a VPC endpoint by ID.
func (d *VpcEndpointDAO) Delete(ctx context.Context, id string) error {
	_, err := d.client.DeleteVpcEndpoints(ctx, &ec2.DeleteVpcEndpointsInput{
		VpcEndpointIds: []string{id},
	})
	if err != nil {
		return apperrors.Wrapf(err, "delete vpc endpoint %s", id)
	}
	return nil
}

// VpcEndpointResource wraps a VPC endpoint.
type VpcEndpointResource struct {
	dao.BaseResource
	Item types.VpcEndpoint
}

// NewVpcEndpointResource creates a new VpcEndpointResource.
func NewVpcEndpointResource(endpoint types.VpcEndpoint) *VpcEndpointResource {
	return &VpcEndpointResource{
		BaseResource: dao.BaseResource{
			ID:  appaws.Str(endpoint.VpcEndpointId),
			ARN: "",
		},
		Item: endpoint,
	}
}

// VpcId returns the VPC ID.
func (r *VpcEndpointResource) VpcId() string {
	return appaws.Str(r.Item.VpcId)
}

// ServiceName returns the service name.
func (r *VpcEndpointResource) ServiceName() string {
	return appaws.Str(r.Item.ServiceName)
}

// VpcEndpointType returns the endpoint type.
func (r *VpcEndpointResource) VpcEndpointType() string {
	return string(r.Item.VpcEndpointType)
}

// State returns the endpoint state.
func (r *VpcEndpointResource) State() string {
	return string(r.Item.State)
}

// PrivateDnsEnabled returns whether private DNS is enabled.
func (r *VpcEndpointResource) PrivateDnsEnabled() bool {
	return appaws.Bool(r.Item.PrivateDnsEnabled)
}

// CreationTimestamp returns when the endpoint was created.
func (r *VpcEndpointResource) CreationTimestamp() *time.Time {
	return r.Item.CreationTimestamp
}

// SubnetIds returns the subnet IDs.
func (r *VpcEndpointResource) SubnetIds() []string {
	return r.Item.SubnetIds
}

// SecurityGroupIds returns the security group IDs.
func (r *VpcEndpointResource) SecurityGroupIds() []string {
	var ids []string
	for _, group := range r.Item.Groups {
		ids = append(ids, appaws.Str(group.GroupId))
	}
	return ids
}

// Name returns the Name tag value.
func (r *VpcEndpointResource) Name() string {
	for _, tag := range r.Item.Tags {
		if appaws.Str(tag.Key) == "Name" {
			return appaws.Str(tag.Value)
		}
	}
	return ""
}

// OwnerId returns the owner account ID.
func (r *VpcEndpointResource) OwnerId() string {
	return appaws.Str(r.Item.OwnerId)
}

// RouteTableIds returns the route table IDs (for Gateway endpoints).
func (r *VpcEndpointResource) RouteTableIds() []string {
	return r.Item.RouteTableIds
}

// NetworkInterfaceIds returns the network interface IDs.
func (r *VpcEndpointResource) NetworkInterfaceIds() []string {
	return r.Item.NetworkInterfaceIds
}

// DnsEntries returns the DNS entries.
func (r *VpcEndpointResource) DnsEntries() []string {
	var entries []string
	for _, entry := range r.Item.DnsEntries {
		if entry.DnsName != nil {
			entries = append(entries, *entry.DnsName)
		}
	}
	return entries
}

// PolicyDocument returns the policy document.
func (r *VpcEndpointResource) PolicyDocument() string {
	return appaws.Str(r.Item.PolicyDocument)
}

// RequesterManaged returns if the endpoint is requester managed.
func (r *VpcEndpointResource) RequesterManaged() bool {
	return appaws.Bool(r.Item.RequesterManaged)
}

// IpAddressType returns the IP address type.
func (r *VpcEndpointResource) IpAddressType() string {
	return string(r.Item.IpAddressType)
}

// Tags returns all tags.
func (r *VpcEndpointResource) Tags() map[string]string {
	tags := make(map[string]string)
	for _, tag := range r.Item.Tags {
		tags[appaws.Str(tag.Key)] = appaws.Str(tag.Value)
	}
	return tags
}
