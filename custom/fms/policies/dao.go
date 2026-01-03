package policies

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/fms"
	"github.com/aws/aws-sdk-go-v2/service/fms/types"

	appaws "github.com/clawscli/claws/internal/aws"
	"github.com/clawscli/claws/internal/dao"
	apperrors "github.com/clawscli/claws/internal/errors"
)

// PolicyDAO provides data access for Firewall Manager policies.
type PolicyDAO struct {
	dao.BaseDAO
	client *fms.Client
}

// NewPolicyDAO creates a new PolicyDAO.
func NewPolicyDAO(ctx context.Context) (dao.DAO, error) {
	cfg, err := appaws.NewConfig(ctx)
	if err != nil {
		return nil, apperrors.Wrap(err, "new "+ServiceResourcePath+" dao")
	}
	return &PolicyDAO{
		BaseDAO: dao.NewBaseDAO("fms", "policies"),
		client:  fms.NewFromConfig(cfg),
	}, nil
}

// List returns all FMS policies.
func (d *PolicyDAO) List(ctx context.Context) ([]dao.Resource, error) {
	policies, err := appaws.Paginate(ctx, func(token *string) ([]types.PolicySummary, *string, error) {
		output, err := d.client.ListPolicies(ctx, &fms.ListPoliciesInput{
			NextToken: token,
		})
		if err != nil {
			return nil, nil, apperrors.Wrap(err, "list fms policies")
		}
		return output.PolicyList, output.NextToken, nil
	})
	if err != nil {
		return nil, err
	}

	resources := make([]dao.Resource, len(policies))
	for i, policy := range policies {
		resources[i] = NewPolicyResource(policy)
	}
	return resources, nil
}

// Get returns a specific policy by ID.
func (d *PolicyDAO) Get(ctx context.Context, id string) (dao.Resource, error) {
	output, err := d.client.GetPolicy(ctx, &fms.GetPolicyInput{
		PolicyId: &id,
	})
	if err != nil {
		return nil, apperrors.Wrapf(err, "get fms policy %s", id)
	}
	return NewPolicyResourceFromDetail(*output.Policy, output.PolicyArn), nil
}

// Delete deletes a policy by ID.
func (d *PolicyDAO) Delete(ctx context.Context, id string) error {
	_, err := d.client.DeletePolicy(ctx, &fms.DeletePolicyInput{
		PolicyId: &id,
	})
	if err != nil {
		return apperrors.Wrapf(err, "delete fms policy %s", id)
	}
	return nil
}

// PolicyResource wraps an FMS policy.
type PolicyResource struct {
	dao.BaseResource
	Summary *types.PolicySummary
	Detail  *types.Policy
}

// NewPolicyResource creates a new PolicyResource from summary.
func NewPolicyResource(policy types.PolicySummary) *PolicyResource {
	return &PolicyResource{
		BaseResource: dao.BaseResource{
			ID:  appaws.Str(policy.PolicyId),
			ARN: appaws.Str(policy.PolicyArn),
		},
		Summary: &policy,
	}
}

// NewPolicyResourceFromDetail creates a new PolicyResource from detail.
func NewPolicyResourceFromDetail(policy types.Policy, arn *string) *PolicyResource {
	return &PolicyResource{
		BaseResource: dao.BaseResource{
			ID:  appaws.Str(policy.PolicyId),
			ARN: appaws.Str(arn),
		},
		Detail: &policy,
	}
}

// PolicyName returns the policy name.
func (r *PolicyResource) PolicyName() string {
	if r.Summary != nil {
		return appaws.Str(r.Summary.PolicyName)
	}
	if r.Detail != nil {
		return appaws.Str(r.Detail.PolicyName)
	}
	return ""
}

// PolicyId returns the policy ID.
func (r *PolicyResource) PolicyId() string {
	return r.ID
}

// SecurityServiceType returns the security service type.
func (r *PolicyResource) SecurityServiceType() string {
	if r.Summary != nil {
		return string(r.Summary.SecurityServiceType)
	}
	if r.Detail != nil && r.Detail.SecurityServicePolicyData != nil {
		return string(r.Detail.SecurityServicePolicyData.Type)
	}
	return ""
}

// ResourceType returns the resource type.
func (r *PolicyResource) ResourceType() string {
	if r.Summary != nil {
		return appaws.Str(r.Summary.ResourceType)
	}
	if r.Detail != nil {
		return appaws.Str(r.Detail.ResourceType)
	}
	return ""
}

// RemediationEnabled returns whether remediation is enabled.
func (r *PolicyResource) RemediationEnabled() bool {
	if r.Summary != nil {
		return r.Summary.RemediationEnabled
	}
	if r.Detail != nil {
		return r.Detail.RemediationEnabled
	}
	return false
}

// DeleteUnusedFMManagedResources returns whether to delete unused resources.
func (r *PolicyResource) DeleteUnusedFMManagedResources() bool {
	if r.Summary != nil {
		return r.Summary.DeleteUnusedFMManagedResources
	}
	if r.Detail != nil {
		return r.Detail.DeleteUnusedFMManagedResources
	}
	return false
}

// ExcludeResourceTags returns whether resources with specific tags are excluded.
func (r *PolicyResource) ExcludeResourceTags() bool {
	if r.Detail != nil {
		return r.Detail.ExcludeResourceTags
	}
	return false
}

// ResourceTypeList returns the list of resource types protected.
func (r *PolicyResource) ResourceTypeList() []string {
	if r.Detail != nil {
		return r.Detail.ResourceTypeList
	}
	return nil
}

// PolicyUpdateToken returns the update token.
func (r *PolicyResource) PolicyUpdateToken() string {
	if r.Detail != nil {
		return appaws.Str(r.Detail.PolicyUpdateToken)
	}
	return ""
}

// ResourceTags returns the resource tags for the policy.
func (r *PolicyResource) ResourceTags() map[string]string {
	if r.Detail != nil && len(r.Detail.ResourceTags) > 0 {
		tags := make(map[string]string)
		for _, t := range r.Detail.ResourceTags {
			tags[appaws.Str(t.Key)] = appaws.Str(t.Value)
		}
		return tags
	}
	return nil
}

// IncludeMap returns the include map (account IDs or OU IDs).
func (r *PolicyResource) IncludeMap() map[string][]string {
	if r.Detail != nil && r.Detail.IncludeMap != nil {
		return r.Detail.IncludeMap
	}
	return nil
}

// ExcludeMap returns the exclude map (account IDs or OU IDs).
func (r *PolicyResource) ExcludeMap() map[string][]string {
	if r.Detail != nil && r.Detail.ExcludeMap != nil {
		return r.Detail.ExcludeMap
	}
	return nil
}
