package firewallpolicies

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/networkfirewall"
	"github.com/aws/aws-sdk-go-v2/service/networkfirewall/types"

	appaws "github.com/clawscli/claws/internal/aws"
	"github.com/clawscli/claws/internal/dao"
	apperrors "github.com/clawscli/claws/internal/errors"
)

// FirewallPolicyDAO provides data access for Network Firewall policies.
type FirewallPolicyDAO struct {
	dao.BaseDAO
	client *networkfirewall.Client
}

// NewFirewallPolicyDAO creates a new FirewallPolicyDAO.
func NewFirewallPolicyDAO(ctx context.Context) (dao.DAO, error) {
	cfg, err := appaws.NewConfig(ctx)
	if err != nil {
		return nil, apperrors.Wrap(err, "new networkfirewall/firewallpolicies dao")
	}
	return &FirewallPolicyDAO{
		BaseDAO: dao.NewBaseDAO("network-firewall", "firewall-policies"),
		client:  networkfirewall.NewFromConfig(cfg),
	}, nil
}

// List returns all Network Firewall policies.
func (d *FirewallPolicyDAO) List(ctx context.Context) ([]dao.Resource, error) {
	policies, err := appaws.Paginate(ctx, func(token *string) ([]types.FirewallPolicyMetadata, *string, error) {
		output, err := d.client.ListFirewallPolicies(ctx, &networkfirewall.ListFirewallPoliciesInput{
			NextToken: token,
		})
		if err != nil {
			return nil, nil, apperrors.Wrap(err, "list network firewall policies")
		}
		return output.FirewallPolicies, output.NextToken, nil
	})
	if err != nil {
		return nil, err
	}

	resources := make([]dao.Resource, len(policies))
	for i, p := range policies {
		resources[i] = NewFirewallPolicyResource(p)
	}
	return resources, nil
}

// Get returns a specific firewall policy by name.
func (d *FirewallPolicyDAO) Get(ctx context.Context, id string) (dao.Resource, error) {
	output, err := d.client.DescribeFirewallPolicy(ctx, &networkfirewall.DescribeFirewallPolicyInput{
		FirewallPolicyName: &id,
	})
	if err != nil {
		return nil, apperrors.Wrapf(err, "describe network firewall policy %s", id)
	}
	return NewFirewallPolicyResourceFromDetail(output.FirewallPolicyResponse, output.FirewallPolicy), nil
}

// Delete deletes a firewall policy by name.
func (d *FirewallPolicyDAO) Delete(ctx context.Context, id string) error {
	_, err := d.client.DeleteFirewallPolicy(ctx, &networkfirewall.DeleteFirewallPolicyInput{
		FirewallPolicyName: &id,
	})
	if err != nil {
		return apperrors.Wrapf(err, "delete network firewall policy %s", id)
	}
	return nil
}

// FirewallPolicyResource wraps a Network Firewall policy.
type FirewallPolicyResource struct {
	dao.BaseResource
	Metadata *types.FirewallPolicyMetadata
	Response *types.FirewallPolicyResponse
	Detail   *types.FirewallPolicy
}

// NewFirewallPolicyResource creates a new FirewallPolicyResource from metadata.
func NewFirewallPolicyResource(p types.FirewallPolicyMetadata) *FirewallPolicyResource {
	return &FirewallPolicyResource{
		BaseResource: dao.BaseResource{
			ID:  appaws.Str(p.Name),
			ARN: appaws.Str(p.Arn),
		},
		Metadata: &p,
	}
}

// NewFirewallPolicyResourceFromDetail creates a new FirewallPolicyResource from detail.
func NewFirewallPolicyResourceFromDetail(resp *types.FirewallPolicyResponse, p *types.FirewallPolicy) *FirewallPolicyResource {
	return &FirewallPolicyResource{
		BaseResource: dao.BaseResource{
			ID:  appaws.Str(resp.FirewallPolicyName),
			ARN: appaws.Str(resp.FirewallPolicyArn),
		},
		Response: resp,
		Detail:   p,
	}
}

// FirewallPolicyName returns the policy name.
func (r *FirewallPolicyResource) FirewallPolicyName() string {
	return r.ID
}

// Status returns the policy status.
func (r *FirewallPolicyResource) Status() string {
	if r.Response != nil {
		return string(r.Response.FirewallPolicyStatus)
	}
	return ""
}

// Description returns the policy description.
func (r *FirewallPolicyResource) Description() string {
	if r.Response != nil {
		return appaws.Str(r.Response.Description)
	}
	return ""
}

// NumberOfAssociations returns the number of firewalls using this policy.
func (r *FirewallPolicyResource) NumberOfAssociations() int32 {
	if r.Response != nil {
		return appaws.Int32(r.Response.NumberOfAssociations)
	}
	return 0
}

// ConsumedStatelessRuleCapacity returns consumed stateless rule capacity.
func (r *FirewallPolicyResource) ConsumedStatelessRuleCapacity() int32 {
	if r.Response != nil {
		return appaws.Int32(r.Response.ConsumedStatelessRuleCapacity)
	}
	return 0
}

// ConsumedStatefulRuleCapacity returns consumed stateful rule capacity.
func (r *FirewallPolicyResource) ConsumedStatefulRuleCapacity() int32 {
	if r.Response != nil {
		return appaws.Int32(r.Response.ConsumedStatefulRuleCapacity)
	}
	return 0
}

// StatelessRuleGroupCount returns the number of stateless rule groups.
func (r *FirewallPolicyResource) StatelessRuleGroupCount() int {
	if r.Detail != nil {
		return len(r.Detail.StatelessRuleGroupReferences)
	}
	return 0
}

// StatefulRuleGroupCount returns the number of stateful rule groups.
func (r *FirewallPolicyResource) StatefulRuleGroupCount() int {
	if r.Detail != nil {
		return len(r.Detail.StatefulRuleGroupReferences)
	}
	return 0
}
