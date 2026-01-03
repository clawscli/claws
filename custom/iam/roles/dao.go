package roles

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/iam/types"

	appaws "github.com/clawscli/claws/internal/aws"
	"github.com/clawscli/claws/internal/dao"
	apperrors "github.com/clawscli/claws/internal/errors"
)

// RoleDAO provides data access for IAM Roles
type RoleDAO struct {
	dao.BaseDAO
	client *iam.Client
}

// NewRoleDAO creates a new RoleDAO
func NewRoleDAO(ctx context.Context) (dao.DAO, error) {
	cfg, err := appaws.NewConfig(ctx)
	if err != nil {
		return nil, apperrors.Wrap(err, "new "+ServiceResourcePath+" dao")
	}
	return &RoleDAO{
		BaseDAO: dao.NewBaseDAO("iam", "roles"),
		client:  iam.NewFromConfig(cfg),
	}, nil
}

// List returns roles (first page only for backwards compatibility).
// For paginated access, use ListPage instead.
func (d *RoleDAO) List(ctx context.Context) ([]dao.Resource, error) {
	resources, _, err := d.ListPage(ctx, 100, "")
	return resources, err
}

// ListPage returns a page of IAM roles.
// Implements dao.PaginatedDAO interface.
func (d *RoleDAO) ListPage(ctx context.Context, pageSize int, pageToken string) ([]dao.Resource, string, error) {
	maxItems := int32(pageSize)
	if maxItems > 1000 {
		maxItems = 1000 // AWS API max
	}

	input := &iam.ListRolesInput{
		MaxItems: &maxItems,
	}
	if pageToken != "" {
		input.Marker = &pageToken
	}

	output, err := d.client.ListRoles(ctx, input)
	if err != nil {
		return nil, "", apperrors.Wrap(err, "list roles")
	}

	resources := make([]dao.Resource, len(output.Roles))
	for i, role := range output.Roles {
		resources[i] = NewRoleResource(role)
	}

	nextToken := ""
	if output.Marker != nil {
		nextToken = *output.Marker
	}

	return resources, nextToken, nil
}

func (d *RoleDAO) Get(ctx context.Context, id string) (dao.Resource, error) {
	output, err := d.client.GetRole(ctx, &iam.GetRoleInput{
		RoleName: &id,
	})
	if err != nil {
		return nil, apperrors.Wrapf(err, "get role %s", id)
	}

	res := NewRoleResource(*output.Role)

	// Fetch attached policies
	if policies, err := d.client.ListAttachedRolePolicies(ctx, &iam.ListAttachedRolePoliciesInput{RoleName: &id}); err == nil {
		res.AttachedPolicies = policies.AttachedPolicies
	}

	// Fetch inline policy names
	if inline, err := d.client.ListRolePolicies(ctx, &iam.ListRolePoliciesInput{RoleName: &id}); err == nil {
		res.InlinePolicies = inline.PolicyNames
	}

	return res, nil
}

func (d *RoleDAO) Delete(ctx context.Context, id string) error {
	_, err := d.client.DeleteRole(ctx, &iam.DeleteRoleInput{
		RoleName: &id,
	})
	if err != nil {
		if apperrors.IsNotFound(err) {
			return nil // Already deleted
		}
		if apperrors.IsResourceInUse(err) {
			return apperrors.Wrapf(err, "role %s is in use (has attached policies or is referenced)", id)
		}
		return apperrors.Wrapf(err, "delete role %s", id)
	}
	return nil
}

// RoleResource wraps an IAM Role
type RoleResource struct {
	dao.BaseResource
	Item             types.Role
	AttachedPolicies []types.AttachedPolicy
	InlinePolicies   []string
}

// NewRoleResource creates a new RoleResource
func NewRoleResource(role types.Role) *RoleResource {
	name := appaws.Str(role.RoleName)

	return &RoleResource{
		BaseResource: dao.BaseResource{
			ID:   name,
			Name: name,
			ARN:  appaws.Str(role.Arn),
			Tags: appaws.TagsToMap(role.Tags),
			Data: role,
		},
		Item: role,
	}
}

// Path returns the role path
func (r *RoleResource) Path() string {
	if r.Item.Path != nil {
		return *r.Item.Path
	}
	return ""
}

// Arn returns the role ARN
func (r *RoleResource) Arn() string {
	if r.Item.Arn != nil {
		return *r.Item.Arn
	}
	return ""
}

// MaxSessionDuration returns the max session duration in seconds
func (r *RoleResource) MaxSessionDuration() int32 {
	if r.Item.MaxSessionDuration != nil {
		return *r.Item.MaxSessionDuration
	}
	return 0
}

// Description returns the role description
func (r *RoleResource) Description() string {
	if r.Item.Description != nil {
		return *r.Item.Description
	}
	return ""
}
