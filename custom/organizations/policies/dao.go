package policies

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/organizations"
	"github.com/aws/aws-sdk-go-v2/service/organizations/types"

	appaws "github.com/clawscli/claws/internal/aws"
	"github.com/clawscli/claws/internal/dao"
)

// PolicyDAO provides data access for Organizations policies.
type PolicyDAO struct {
	dao.BaseDAO
	client *organizations.Client
}

// NewPolicyDAO creates a new PolicyDAO.
func NewPolicyDAO(ctx context.Context) (dao.DAO, error) {
	cfg, err := appaws.NewConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("new organizations/policies dao: %w", err)
	}
	return &PolicyDAO{
		BaseDAO: dao.NewBaseDAO("organizations", "policies"),
		client:  organizations.NewFromConfig(cfg),
	}, nil
}

// List returns all policies (of all types).
func (d *PolicyDAO) List(ctx context.Context) ([]dao.Resource, error) {
	// List all policy types
	policyTypes := []types.PolicyType{
		types.PolicyTypeServiceControlPolicy,
		types.PolicyTypeTagPolicy,
		types.PolicyTypeBackupPolicy,
		types.PolicyTypeAiservicesOptOutPolicy,
	}

	var allPolicies []types.PolicySummary
	for _, policyType := range policyTypes {
		policies, err := appaws.Paginate(ctx, func(token *string) ([]types.PolicySummary, *string, error) {
			output, err := d.client.ListPolicies(ctx, &organizations.ListPoliciesInput{
				Filter:    policyType,
				NextToken: token,
			})
			if err != nil {
				// Skip if policy type is not enabled
				return nil, nil, nil
			}
			return output.Policies, output.NextToken, nil
		})
		if err != nil {
			return nil, err
		}
		allPolicies = append(allPolicies, policies...)
	}

	resources := make([]dao.Resource, len(allPolicies))
	for i, policy := range allPolicies {
		resources[i] = NewPolicyResource(policy)
	}
	return resources, nil
}

// Get returns a specific policy.
func (d *PolicyDAO) Get(ctx context.Context, id string) (dao.Resource, error) {
	output, err := d.client.DescribePolicy(ctx, &organizations.DescribePolicyInput{
		PolicyId: &id,
	})
	if err != nil {
		return nil, fmt.Errorf("describe organizations policy: %w", err)
	}
	return &PolicyResource{
		BaseResource: dao.BaseResource{
			ID:  appaws.Str(output.Policy.PolicySummary.Id),
			ARN: appaws.Str(output.Policy.PolicySummary.Arn),
		},
		Policy:  output.Policy.PolicySummary,
		Content: appaws.Str(output.Policy.Content),
	}, nil
}

// Delete deletes a policy.
func (d *PolicyDAO) Delete(ctx context.Context, id string) error {
	_, err := d.client.DeletePolicy(ctx, &organizations.DeletePolicyInput{
		PolicyId: &id,
	})
	if err != nil {
		return fmt.Errorf("delete organizations policy: %w", err)
	}
	return nil
}

// PolicyResource wraps an Organizations policy.
type PolicyResource struct {
	dao.BaseResource
	Policy  *types.PolicySummary
	Content string
}

// NewPolicyResource creates a new PolicyResource.
func NewPolicyResource(policy types.PolicySummary) *PolicyResource {
	return &PolicyResource{
		BaseResource: dao.BaseResource{
			ID:  appaws.Str(policy.Id),
			ARN: appaws.Str(policy.Arn),
		},
		Policy: &policy,
	}
}

// Name returns the policy name.
func (r *PolicyResource) Name() string {
	if r.Policy != nil && r.Policy.Name != nil {
		return *r.Policy.Name
	}
	return ""
}

// Description returns the policy description.
func (r *PolicyResource) Description() string {
	if r.Policy != nil && r.Policy.Description != nil {
		return *r.Policy.Description
	}
	return ""
}

// Type returns the policy type.
func (r *PolicyResource) Type() string {
	if r.Policy != nil {
		return string(r.Policy.Type)
	}
	return ""
}

// AwsManaged returns whether this is an AWS managed policy.
func (r *PolicyResource) AwsManaged() bool {
	if r.Policy != nil {
		return r.Policy.AwsManaged
	}
	return false
}
