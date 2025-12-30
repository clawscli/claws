package securitygroups

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"

	appaws "github.com/clawscli/claws/internal/aws"
	"github.com/clawscli/claws/internal/dao"
)

// SecurityGroupDAO provides data access for EC2 security groups
type SecurityGroupDAO struct {
	dao.BaseDAO
	client *ec2.Client
}

// NewSecurityGroupDAO creates a new SecurityGroupDAO
func NewSecurityGroupDAO(ctx context.Context) (dao.DAO, error) {
	cfg, err := appaws.NewConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("new ec2/securitygroups dao: %w", err)
	}
	return &SecurityGroupDAO{
		BaseDAO: dao.NewBaseDAO("ec2", "security-groups"),
		client:  ec2.NewFromConfig(cfg),
	}, nil
}

func (d *SecurityGroupDAO) List(ctx context.Context) ([]dao.Resource, error) {
	input := &ec2.DescribeSecurityGroupsInput{}
	paginator := ec2.NewDescribeSecurityGroupsPaginator(d.client, input)

	var resources []dao.Resource
	for paginator.HasMorePages() {
		output, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("describe security groups: %w", err)
		}

		for _, sg := range output.SecurityGroups {
			resources = append(resources, NewSecurityGroupResource(sg))
		}
	}

	return resources, nil
}

func (d *SecurityGroupDAO) Get(ctx context.Context, id string) (dao.Resource, error) {
	input := &ec2.DescribeSecurityGroupsInput{
		GroupIds: []string{id},
	}

	output, err := d.client.DescribeSecurityGroups(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("describe security group %s: %w", id, err)
	}

	if len(output.SecurityGroups) == 0 {
		return nil, fmt.Errorf("security group not found: %s", id)
	}

	return NewSecurityGroupResource(output.SecurityGroups[0]), nil
}

func (d *SecurityGroupDAO) Delete(ctx context.Context, id string) error {
	_, err := d.client.DeleteSecurityGroup(ctx, &ec2.DeleteSecurityGroupInput{
		GroupId: &id,
	})
	if err != nil {
		if appaws.IsNotFound(err) {
			return nil // Already deleted
		}
		if appaws.IsResourceInUse(err) {
			return fmt.Errorf("security group %s is in use by other resources", id)
		}
		return fmt.Errorf("delete security group %s: %w", id, err)
	}
	return nil
}

// SecurityGroupResource wraps an EC2 security group
type SecurityGroupResource struct {
	dao.BaseResource
	Item types.SecurityGroup
}

// NewSecurityGroupResource creates a new SecurityGroupResource
func NewSecurityGroupResource(sg types.SecurityGroup) *SecurityGroupResource {
	return &SecurityGroupResource{
		BaseResource: dao.BaseResource{
			ID:   appaws.Str(sg.GroupId),
			Name: appaws.Str(sg.GroupName),
			Tags: appaws.TagsToMap(sg.Tags),
			Data: sg,
		},
		Item: sg,
	}
}

func (r *SecurityGroupResource) Description() string {
	if r.Item.Description != nil {
		return *r.Item.Description
	}
	return ""
}

func (r *SecurityGroupResource) VpcID() string {
	if r.Item.VpcId != nil {
		return *r.Item.VpcId
	}
	return ""
}

func (r *SecurityGroupResource) InboundRuleCount() int {
	return len(r.Item.IpPermissions)
}

func (r *SecurityGroupResource) OutboundRuleCount() int {
	return len(r.Item.IpPermissionsEgress)
}
