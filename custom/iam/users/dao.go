package users

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/iam/types"

	appaws "github.com/clawscli/claws/internal/aws"
	"github.com/clawscli/claws/internal/dao"
)

// UserDetail contains extended user information from multiple API calls
type UserDetail struct {
	User             types.User
	AccessKeys       []types.AccessKeyMetadata
	MFADevices       []types.MFADevice
	Groups           []types.Group
	AttachedPolicies []types.AttachedPolicy
	InlinePolicies   []string
}

// UserDAO provides data access for IAM Users
type UserDAO struct {
	dao.BaseDAO
	client *iam.Client
}

// NewUserDAO creates a new UserDAO
func NewUserDAO(ctx context.Context) (dao.DAO, error) {
	cfg, err := appaws.NewConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("new iam/users dao: %w", err)
	}
	return &UserDAO{
		BaseDAO: dao.NewBaseDAO("iam", "users"),
		client:  iam.NewFromConfig(cfg),
	}, nil
}

func (d *UserDAO) List(ctx context.Context) ([]dao.Resource, error) {
	paginator := iam.NewListUsersPaginator(d.client, &iam.ListUsersInput{})

	var resources []dao.Resource
	for paginator.HasMorePages() {
		output, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("list users: %w", err)
		}

		for _, user := range output.Users {
			resources = append(resources, NewUserResource(user))
		}
	}

	return resources, nil
}

func (d *UserDAO) Get(ctx context.Context, id string) (dao.Resource, error) {
	output, err := d.client.GetUser(ctx, &iam.GetUserInput{
		UserName: &id,
	})
	if err != nil {
		return nil, fmt.Errorf("get user %s: %w", id, err)
	}

	detail := UserDetail{User: *output.User}

	// Fetch access keys
	if keys, err := d.client.ListAccessKeys(ctx, &iam.ListAccessKeysInput{UserName: &id}); err == nil {
		detail.AccessKeys = keys.AccessKeyMetadata
	}

	// Fetch MFA devices
	if mfa, err := d.client.ListMFADevices(ctx, &iam.ListMFADevicesInput{UserName: &id}); err == nil {
		detail.MFADevices = mfa.MFADevices
	}

	// Fetch groups
	if groups, err := d.client.ListGroupsForUser(ctx, &iam.ListGroupsForUserInput{UserName: &id}); err == nil {
		detail.Groups = groups.Groups
	}

	// Fetch attached policies
	if policies, err := d.client.ListAttachedUserPolicies(ctx, &iam.ListAttachedUserPoliciesInput{UserName: &id}); err == nil {
		detail.AttachedPolicies = policies.AttachedPolicies
	}

	// Fetch inline policy names
	if inline, err := d.client.ListUserPolicies(ctx, &iam.ListUserPoliciesInput{UserName: &id}); err == nil {
		detail.InlinePolicies = inline.PolicyNames
	}

	return NewUserResourceWithDetail(detail), nil
}

func (d *UserDAO) Delete(ctx context.Context, id string) error {
	_, err := d.client.DeleteUser(ctx, &iam.DeleteUserInput{
		UserName: &id,
	})
	if err != nil {
		return fmt.Errorf("delete user %s: %w", id, err)
	}
	return nil
}

// UserResource wraps an IAM User
type UserResource struct {
	dao.BaseResource
	Item             types.User
	AccessKeys       []types.AccessKeyMetadata
	MFADevices       []types.MFADevice
	Groups           []types.Group
	AttachedPolicies []types.AttachedPolicy
	InlinePolicies   []string
}

// NewUserResource creates a new UserResource
func NewUserResource(user types.User) *UserResource {
	name := appaws.Str(user.UserName)

	return &UserResource{
		BaseResource: dao.BaseResource{
			ID:   name,
			Name: name,
			ARN:  appaws.Str(user.Arn),
			Tags: appaws.TagsToMap(user.Tags),
			Data: user,
		},
		Item: user,
	}
}

// NewUserResourceWithDetail creates a new UserResource with extended details
func NewUserResourceWithDetail(detail UserDetail) *UserResource {
	name := appaws.Str(detail.User.UserName)

	return &UserResource{
		BaseResource: dao.BaseResource{
			ID:   name,
			Name: name,
			ARN:  appaws.Str(detail.User.Arn),
			Tags: appaws.TagsToMap(detail.User.Tags),
			Data: detail.User,
		},
		Item:             detail.User,
		AccessKeys:       detail.AccessKeys,
		MFADevices:       detail.MFADevices,
		Groups:           detail.Groups,
		AttachedPolicies: detail.AttachedPolicies,
		InlinePolicies:   detail.InlinePolicies,
	}
}

// Path returns the user path
func (r *UserResource) Path() string {
	if r.Item.Path != nil {
		return *r.Item.Path
	}
	return ""
}

// Arn returns the user ARN
func (r *UserResource) Arn() string {
	if r.Item.Arn != nil {
		return *r.Item.Arn
	}
	return ""
}

// UserId returns the user ID
func (r *UserResource) UserId() string {
	if r.Item.UserId != nil {
		return *r.Item.UserId
	}
	return ""
}
