package groups

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/iam/types"

	appaws "github.com/clawscli/claws/internal/aws"
	"github.com/clawscli/claws/internal/dao"
)

// GroupDAO provides data access for IAM Groups
type GroupDAO struct {
	dao.BaseDAO
	client *iam.Client
}

// NewGroupDAO creates a new GroupDAO
func NewGroupDAO(ctx context.Context) (dao.DAO, error) {
	cfg, err := appaws.NewConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("new iam/groups dao: %w", err)
	}
	return &GroupDAO{
		BaseDAO: dao.NewBaseDAO("iam", "groups"),
		client:  iam.NewFromConfig(cfg),
	}, nil
}

func (d *GroupDAO) List(ctx context.Context) ([]dao.Resource, error) {
	paginator := iam.NewListGroupsPaginator(d.client, &iam.ListGroupsInput{})

	var resources []dao.Resource
	for paginator.HasMorePages() {
		output, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("list groups: %w", err)
		}

		for _, group := range output.Groups {
			resources = append(resources, NewGroupResource(group))
		}
	}

	return resources, nil
}

func (d *GroupDAO) Get(ctx context.Context, id string) (dao.Resource, error) {
	output, err := d.client.GetGroup(ctx, &iam.GetGroupInput{
		GroupName: &id,
	})
	if err != nil {
		return nil, fmt.Errorf("get group %s: %w", id, err)
	}

	return NewGroupResource(*output.Group), nil
}

func (d *GroupDAO) Delete(ctx context.Context, id string) error {
	_, err := d.client.DeleteGroup(ctx, &iam.DeleteGroupInput{
		GroupName: &id,
	})
	if err != nil {
		return fmt.Errorf("delete group %s: %w", id, err)
	}
	return nil
}

// GroupResource wraps an IAM Group
type GroupResource struct {
	dao.BaseResource
	Item types.Group
}

// NewGroupResource creates a new GroupResource
func NewGroupResource(group types.Group) *GroupResource {
	name := appaws.Str(group.GroupName)

	return &GroupResource{
		BaseResource: dao.BaseResource{
			ID:   name,
			Name: name,
			ARN:  appaws.Str(group.Arn),
			Tags: nil, // Groups don't have tags in ListGroups response
			Data: group,
		},
		Item: group,
	}
}

// Path returns the group path
func (r *GroupResource) Path() string {
	if r.Item.Path != nil {
		return *r.Item.Path
	}
	return ""
}

// Arn returns the group ARN
func (r *GroupResource) Arn() string {
	if r.Item.Arn != nil {
		return *r.Item.Arn
	}
	return ""
}

// GroupId returns the group ID
func (r *GroupResource) GroupId() string {
	if r.Item.GroupId != nil {
		return *r.Item.GroupId
	}
	return ""
}
