package rulegroups

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/networkfirewall"
	"github.com/aws/aws-sdk-go-v2/service/networkfirewall/types"

	appaws "github.com/clawscli/claws/internal/aws"
	"github.com/clawscli/claws/internal/dao"
	apperrors "github.com/clawscli/claws/internal/errors"
)

// RuleGroupDAO provides data access for Network Firewall rule groups.
type RuleGroupDAO struct {
	dao.BaseDAO
	client *networkfirewall.Client
}

// NewRuleGroupDAO creates a new RuleGroupDAO.
func NewRuleGroupDAO(ctx context.Context) (dao.DAO, error) {
	cfg, err := appaws.NewConfig(ctx)
	if err != nil {
		return nil, apperrors.Wrap(err, "new networkfirewall/rulegroups dao")
	}
	return &RuleGroupDAO{
		BaseDAO: dao.NewBaseDAO("network-firewall", "rule-groups"),
		client:  networkfirewall.NewFromConfig(cfg),
	}, nil
}

// List returns all Network Firewall rule groups.
func (d *RuleGroupDAO) List(ctx context.Context) ([]dao.Resource, error) {
	ruleGroups, err := appaws.Paginate(ctx, func(token *string) ([]types.RuleGroupMetadata, *string, error) {
		output, err := d.client.ListRuleGroups(ctx, &networkfirewall.ListRuleGroupsInput{
			NextToken: token,
		})
		if err != nil {
			return nil, nil, apperrors.Wrap(err, "list network firewall rule groups")
		}
		return output.RuleGroups, output.NextToken, nil
	})
	if err != nil {
		return nil, err
	}

	resources := make([]dao.Resource, len(ruleGroups))
	for i, rg := range ruleGroups {
		resources[i] = NewRuleGroupResource(rg)
	}
	return resources, nil
}

// Get returns a specific rule group by name.
func (d *RuleGroupDAO) Get(ctx context.Context, id string) (dao.Resource, error) {
	output, err := d.client.DescribeRuleGroup(ctx, &networkfirewall.DescribeRuleGroupInput{
		RuleGroupName: &id,
	})
	if err != nil {
		return nil, apperrors.Wrapf(err, "describe network firewall rule group %s", id)
	}
	return NewRuleGroupResourceFromDetail(output.RuleGroupResponse, output.RuleGroup), nil
}

// Delete deletes a rule group by name.
func (d *RuleGroupDAO) Delete(ctx context.Context, id string) error {
	_, err := d.client.DeleteRuleGroup(ctx, &networkfirewall.DeleteRuleGroupInput{
		RuleGroupName: &id,
	})
	if err != nil {
		return apperrors.Wrapf(err, "delete network firewall rule group %s", id)
	}
	return nil
}

// RuleGroupResource wraps a Network Firewall rule group.
type RuleGroupResource struct {
	dao.BaseResource
	Metadata *types.RuleGroupMetadata
	Response *types.RuleGroupResponse
	Detail   *types.RuleGroup
}

// NewRuleGroupResource creates a new RuleGroupResource from metadata.
func NewRuleGroupResource(rg types.RuleGroupMetadata) *RuleGroupResource {
	return &RuleGroupResource{
		BaseResource: dao.BaseResource{
			ID:  appaws.Str(rg.Name),
			ARN: appaws.Str(rg.Arn),
		},
		Metadata: &rg,
	}
}

// NewRuleGroupResourceFromDetail creates a new RuleGroupResource from detail.
func NewRuleGroupResourceFromDetail(resp *types.RuleGroupResponse, rg *types.RuleGroup) *RuleGroupResource {
	return &RuleGroupResource{
		BaseResource: dao.BaseResource{
			ID:  appaws.Str(resp.RuleGroupName),
			ARN: appaws.Str(resp.RuleGroupArn),
		},
		Response: resp,
		Detail:   rg,
	}
}

// RuleGroupName returns the rule group name.
func (r *RuleGroupResource) RuleGroupName() string {
	return r.ID
}

// Type returns the rule group type (STATELESS or STATEFUL).
func (r *RuleGroupResource) Type() string {
	if r.Response != nil {
		return string(r.Response.Type)
	}
	return ""
}

// Status returns the rule group status.
func (r *RuleGroupResource) Status() string {
	if r.Response != nil {
		return string(r.Response.RuleGroupStatus)
	}
	return ""
}

// Capacity returns the rule group capacity.
func (r *RuleGroupResource) Capacity() int32 {
	if r.Response != nil {
		return appaws.Int32(r.Response.Capacity)
	}
	return 0
}

// Description returns the rule group description.
func (r *RuleGroupResource) Description() string {
	if r.Response != nil {
		return appaws.Str(r.Response.Description)
	}
	return ""
}

// NumberOfAssociations returns the number of firewall policies using this rule group.
func (r *RuleGroupResource) NumberOfAssociations() int32 {
	if r.Response != nil {
		return appaws.Int32(r.Response.NumberOfAssociations)
	}
	return 0
}

// ConsumedCapacity returns the consumed capacity.
func (r *RuleGroupResource) ConsumedCapacity() int32 {
	if r.Response != nil {
		return appaws.Int32(r.Response.ConsumedCapacity)
	}
	return 0
}
