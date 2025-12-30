package policies

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/iam/types"

	appaws "github.com/clawscli/claws/internal/aws"
	"github.com/clawscli/claws/internal/dao"
)

// PolicyDAO provides data access for IAM Policies
type PolicyDAO struct {
	dao.BaseDAO
	client *iam.Client
}

// NewPolicyDAO creates a new PolicyDAO
func NewPolicyDAO(ctx context.Context) (dao.DAO, error) {
	cfg, err := appaws.NewConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("new iam/policies dao: %w", err)
	}
	return &PolicyDAO{
		BaseDAO: dao.NewBaseDAO("iam", "policies"),
		client:  iam.NewFromConfig(cfg),
	}, nil
}

// List returns policies (first page only for backwards compatibility).
// For paginated access, use ListPage instead.
func (d *PolicyDAO) List(ctx context.Context) ([]dao.Resource, error) {
	resources, _, err := d.ListPage(ctx, 100, "")
	return resources, err
}

// ListPage returns a page of IAM policies.
// Implements dao.PaginatedDAO interface.
func (d *PolicyDAO) ListPage(ctx context.Context, pageSize int, pageToken string) ([]dao.Resource, string, error) {
	maxItems := int32(pageSize)
	if maxItems > 1000 {
		maxItems = 1000 // AWS API max
	}

	input := &iam.ListPoliciesInput{
		Scope:    types.PolicyScopeTypeAll,
		MaxItems: &maxItems,
	}
	if pageToken != "" {
		input.Marker = &pageToken
	}

	output, err := d.client.ListPolicies(ctx, input)
	if err != nil {
		return nil, "", fmt.Errorf("list policies: %w", err)
	}

	resources := make([]dao.Resource, len(output.Policies))
	for i, policy := range output.Policies {
		resources[i] = NewPolicyResource(policy)
	}

	nextToken := ""
	if output.Marker != nil {
		nextToken = *output.Marker
	}

	return resources, nextToken, nil
}

func (d *PolicyDAO) Get(ctx context.Context, id string) (dao.Resource, error) {
	output, err := d.client.GetPolicy(ctx, &iam.GetPolicyInput{
		PolicyArn: &id,
	})
	if err != nil {
		return nil, fmt.Errorf("get policy %s: %w", id, err)
	}

	res := NewPolicyResource(*output.Policy)

	// Fetch the policy document for the default version
	if output.Policy.DefaultVersionId != nil {
		versionOutput, err := d.client.GetPolicyVersion(ctx, &iam.GetPolicyVersionInput{
			PolicyArn: &id,
			VersionId: output.Policy.DefaultVersionId,
		})
		if err == nil && versionOutput.PolicyVersion != nil && versionOutput.PolicyVersion.Document != nil {
			res.PolicyDocument = *versionOutput.PolicyVersion.Document
		}
	}

	// List entities attached to this policy
	if entities, err := d.client.ListEntitiesForPolicy(ctx, &iam.ListEntitiesForPolicyInput{PolicyArn: &id}); err == nil {
		res.AttachedUsers = entities.PolicyUsers
		res.AttachedRoles = entities.PolicyRoles
		res.AttachedGroups = entities.PolicyGroups
	}

	return res, nil
}

func (d *PolicyDAO) Delete(ctx context.Context, id string) error {
	_, err := d.client.DeletePolicy(ctx, &iam.DeletePolicyInput{
		PolicyArn: &id,
	})
	if err != nil {
		if appaws.IsNotFound(err) {
			return nil // Already deleted
		}
		if appaws.IsResourceInUse(err) {
			return fmt.Errorf("policy %s is in use (attached to roles/users/groups)", id)
		}
		return fmt.Errorf("delete policy %s: %w", id, err)
	}
	return nil
}

// PolicyResource wraps an IAM Policy
type PolicyResource struct {
	dao.BaseResource
	Item           types.Policy
	PolicyDocument string
	AttachedUsers  []types.PolicyUser
	AttachedRoles  []types.PolicyRole
	AttachedGroups []types.PolicyGroup
}

// NewPolicyResource creates a new PolicyResource
func NewPolicyResource(policy types.Policy) *PolicyResource {
	arn := appaws.Str(policy.Arn)
	return &PolicyResource{
		BaseResource: dao.BaseResource{
			ID:   arn,
			Name: appaws.Str(policy.PolicyName),
			ARN:  arn,
			Tags: appaws.TagsToMap(policy.Tags),
			Data: policy,
		},
		Item: policy,
	}
}

// Path returns the policy path
func (r *PolicyResource) Path() string {
	if r.Item.Path != nil {
		return *r.Item.Path
	}
	return ""
}

// Arn returns the policy ARN
func (r *PolicyResource) Arn() string {
	if r.Item.Arn != nil {
		return *r.Item.Arn
	}
	return ""
}

// PolicyId returns the policy ID
func (r *PolicyResource) PolicyId() string {
	if r.Item.PolicyId != nil {
		return *r.Item.PolicyId
	}
	return ""
}

// IsAttachable returns if the policy is attachable
func (r *PolicyResource) IsAttachable() bool {
	return r.Item.IsAttachable
}

// AttachmentCount returns the number of attachments
func (r *PolicyResource) AttachmentCount() int32 {
	if r.Item.AttachmentCount != nil {
		return *r.Item.AttachmentCount
	}
	return 0
}

// IsAWSManaged returns true if this is an AWS managed policy
func (r *PolicyResource) IsAWSManaged() bool {
	if r.Item.Arn != nil {
		// AWS managed policies have ARNs starting with arn:aws:iam::aws:
		return len(*r.Item.Arn) > 20 && (*r.Item.Arn)[13:16] == "aws"
	}
	return false
}

// Scope returns "AWS" for AWS managed policies, "Local" for customer managed
func (r *PolicyResource) Scope() string {
	if r.IsAWSManaged() {
		return "AWS"
	}
	return "Local"
}
