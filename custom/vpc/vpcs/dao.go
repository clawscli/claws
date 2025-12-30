package vpcs

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"

	appaws "github.com/clawscli/claws/internal/aws"
	"github.com/clawscli/claws/internal/dao"
)

// VPCDAO provides data access for VPCs
type VPCDAO struct {
	dao.BaseDAO
	client *ec2.Client
}

// NewVPCDAO creates a new VPCDAO
func NewVPCDAO(ctx context.Context) (dao.DAO, error) {
	cfg, err := appaws.NewConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("new vpc/vpcs dao: %w", err)
	}
	return &VPCDAO{
		BaseDAO: dao.NewBaseDAO("vpc", "vpcs"),
		client:  ec2.NewFromConfig(cfg),
	}, nil
}

func (d *VPCDAO) List(ctx context.Context) ([]dao.Resource, error) {
	output, err := d.client.DescribeVpcs(ctx, &ec2.DescribeVpcsInput{})
	if err != nil {
		return nil, fmt.Errorf("describe vpcs: %w", err)
	}

	var resources []dao.Resource
	for _, vpc := range output.Vpcs {
		resources = append(resources, NewVPCResource(vpc))
	}

	return resources, nil
}

func (d *VPCDAO) Get(ctx context.Context, id string) (dao.Resource, error) {
	output, err := d.client.DescribeVpcs(ctx, &ec2.DescribeVpcsInput{
		VpcIds: []string{id},
	})
	if err != nil {
		return nil, fmt.Errorf("describe vpc %s: %w", id, err)
	}

	if len(output.Vpcs) == 0 {
		return nil, fmt.Errorf("vpc not found: %s", id)
	}

	res := NewVPCResource(output.Vpcs[0])

	// Fetch DNS settings
	if dnsHostnames, err := d.client.DescribeVpcAttribute(ctx, &ec2.DescribeVpcAttributeInput{
		VpcId:     &id,
		Attribute: "enableDnsHostnames",
	}); err == nil && dnsHostnames.EnableDnsHostnames != nil {
		res.EnableDnsHostnames = dnsHostnames.EnableDnsHostnames.Value != nil && *dnsHostnames.EnableDnsHostnames.Value
	}

	if dnsSupport, err := d.client.DescribeVpcAttribute(ctx, &ec2.DescribeVpcAttributeInput{
		VpcId:     &id,
		Attribute: "enableDnsSupport",
	}); err == nil && dnsSupport.EnableDnsSupport != nil {
		res.EnableDnsSupport = dnsSupport.EnableDnsSupport.Value != nil && *dnsSupport.EnableDnsSupport.Value
	}

	return res, nil
}

func (d *VPCDAO) Delete(ctx context.Context, id string) error {
	_, err := d.client.DeleteVpc(ctx, &ec2.DeleteVpcInput{
		VpcId: &id,
	})
	if err != nil {
		if appaws.IsNotFound(err) {
			return nil // Already deleted
		}
		if appaws.IsResourceInUse(err) {
			return fmt.Errorf("vpc %s is in use", id)
		}
		return fmt.Errorf("delete vpc %s: %w", id, err)
	}
	return nil
}

// VPCResource wraps a VPC
type VPCResource struct {
	dao.BaseResource
	Item               types.Vpc
	EnableDnsHostnames bool
	EnableDnsSupport   bool
}

// NewVPCResource creates a new VPCResource
func NewVPCResource(vpc types.Vpc) *VPCResource {
	name := appaws.EC2NameTag(vpc.Tags)
	id := appaws.Str(vpc.VpcId)

	return &VPCResource{
		BaseResource: dao.BaseResource{
			ID:   id,
			Name: name,
			Tags: appaws.TagsToMap(vpc.Tags),
			Data: vpc,
		},
		Item: vpc,
	}
}

// State returns the VPC state
func (r *VPCResource) State() string {
	return string(r.Item.State)
}

// CidrBlock returns the CIDR block
func (r *VPCResource) CidrBlock() string {
	if r.Item.CidrBlock != nil {
		return *r.Item.CidrBlock
	}
	return ""
}

// IsDefault returns whether this is the default VPC
func (r *VPCResource) IsDefault() bool {
	if r.Item.IsDefault != nil {
		return *r.Item.IsDefault
	}
	return false
}

// Tenancy returns the instance tenancy
func (r *VPCResource) Tenancy() string {
	return string(r.Item.InstanceTenancy)
}

// OwnerId returns the owner ID
func (r *VPCResource) OwnerId() string {
	if r.Item.OwnerId != nil {
		return *r.Item.OwnerId
	}
	return ""
}
