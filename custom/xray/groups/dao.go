package groups

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/xray"
	"github.com/aws/aws-sdk-go-v2/service/xray/types"

	appaws "github.com/clawscli/claws/internal/aws"
	"github.com/clawscli/claws/internal/dao"
)

// GroupDAO provides data access for X-Ray groups.
type GroupDAO struct {
	dao.BaseDAO
	client *xray.Client
}

// NewGroupDAO creates a new GroupDAO.
func NewGroupDAO(ctx context.Context) (dao.DAO, error) {
	cfg, err := appaws.NewConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("new xray/groups dao: %w", err)
	}
	return &GroupDAO{
		BaseDAO: dao.NewBaseDAO("xray", "groups"),
		client:  xray.NewFromConfig(cfg),
	}, nil
}

// List returns all X-Ray groups.
func (d *GroupDAO) List(ctx context.Context) ([]dao.Resource, error) {
	groups, err := appaws.Paginate(ctx, func(token *string) ([]types.GroupSummary, *string, error) {
		output, err := d.client.GetGroups(ctx, &xray.GetGroupsInput{
			NextToken: token,
		})
		if err != nil {
			return nil, nil, fmt.Errorf("get xray groups: %w", err)
		}
		return output.Groups, output.NextToken, nil
	})
	if err != nil {
		return nil, err
	}

	resources := make([]dao.Resource, len(groups))
	for i, group := range groups {
		resources[i] = NewGroupResource(group)
	}
	return resources, nil
}

// Get returns a specific X-Ray group by name.
func (d *GroupDAO) Get(ctx context.Context, id string) (dao.Resource, error) {
	output, err := d.client.GetGroup(ctx, &xray.GetGroupInput{
		GroupName: &id,
	})
	if err != nil {
		return nil, fmt.Errorf("get xray group %s: %w", id, err)
	}
	return NewGroupResourceFromDetail(*output.Group), nil
}

// Delete deletes an X-Ray group by name.
func (d *GroupDAO) Delete(ctx context.Context, id string) error {
	_, err := d.client.DeleteGroup(ctx, &xray.DeleteGroupInput{
		GroupName: &id,
	})
	if err != nil {
		return fmt.Errorf("delete xray group %s: %w", id, err)
	}
	return nil
}

// GroupResource wraps an X-Ray group.
type GroupResource struct {
	dao.BaseResource
	Item             *types.GroupSummary
	InsightsEnabled  bool
	NotificationsArn string
}

// NewGroupResource creates a new GroupResource from GroupSummary.
func NewGroupResource(group types.GroupSummary) *GroupResource {
	insightsEnabled := false
	if group.InsightsConfiguration != nil {
		insightsEnabled = appaws.Bool(group.InsightsConfiguration.InsightsEnabled)
	}
	return &GroupResource{
		BaseResource: dao.BaseResource{
			ID:  appaws.Str(group.GroupName),
			ARN: appaws.Str(group.GroupARN),
		},
		Item:            &group,
		InsightsEnabled: insightsEnabled,
	}
}

// NewGroupResourceFromDetail creates a new GroupResource from Group (detail).
func NewGroupResourceFromDetail(group types.Group) *GroupResource {
	insightsEnabled := false
	notificationsEnabled := false
	if group.InsightsConfiguration != nil {
		insightsEnabled = appaws.Bool(group.InsightsConfiguration.InsightsEnabled)
		notificationsEnabled = appaws.Bool(group.InsightsConfiguration.NotificationsEnabled)
	}
	notificationsArn := ""
	if notificationsEnabled {
		notificationsArn = "Enabled"
	}
	return &GroupResource{
		BaseResource: dao.BaseResource{
			ID:  appaws.Str(group.GroupName),
			ARN: appaws.Str(group.GroupARN),
		},
		InsightsEnabled:  insightsEnabled,
		NotificationsArn: notificationsArn,
	}
}

// Name returns the group name.
func (r *GroupResource) Name() string {
	return r.ID
}

// FilterExpression returns the filter expression.
func (r *GroupResource) FilterExpression() string {
	if r.Item != nil {
		return appaws.Str(r.Item.FilterExpression)
	}
	return ""
}
