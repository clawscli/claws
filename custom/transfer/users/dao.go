package users

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/transfer"
	"github.com/aws/aws-sdk-go-v2/service/transfer/types"

	appaws "github.com/clawscli/claws/internal/aws"
	"github.com/clawscli/claws/internal/dao"
)

// UserDAO provides data access for Transfer Family users.
type UserDAO struct {
	dao.BaseDAO
	client *transfer.Client
}

// NewUserDAO creates a new UserDAO.
func NewUserDAO(ctx context.Context) (dao.DAO, error) {
	cfg, err := appaws.NewConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("new transfer/users dao: %w", err)
	}
	return &UserDAO{
		BaseDAO: dao.NewBaseDAO("transfer", "users"),
		client:  transfer.NewFromConfig(cfg),
	}, nil
}

// List returns all users for a Transfer Family server.
func (d *UserDAO) List(ctx context.Context) ([]dao.Resource, error) {
	serverId := dao.GetFilterFromContext(ctx, "ServerId")
	if serverId == "" {
		return nil, fmt.Errorf("server ID filter required")
	}

	users, err := appaws.Paginate(ctx, func(token *string) ([]types.ListedUser, *string, error) {
		output, err := d.client.ListUsers(ctx, &transfer.ListUsersInput{
			ServerId:  &serverId,
			NextToken: token,
		})
		if err != nil {
			return nil, nil, fmt.Errorf("list transfer users: %w", err)
		}
		return output.Users, output.NextToken, nil
	})
	if err != nil {
		return nil, err
	}

	resources := make([]dao.Resource, len(users))
	for i, user := range users {
		resources[i] = NewUserResource(user, serverId)
	}
	return resources, nil
}

// Get returns a specific user by username.
func (d *UserDAO) Get(ctx context.Context, username string) (dao.Resource, error) {
	serverId := dao.GetFilterFromContext(ctx, "ServerId")
	if serverId == "" {
		return nil, fmt.Errorf("server ID filter required")
	}

	output, err := d.client.DescribeUser(ctx, &transfer.DescribeUserInput{
		ServerId: &serverId,
		UserName: &username,
	})
	if err != nil {
		return nil, fmt.Errorf("describe transfer user %s: %w", username, err)
	}
	return NewUserResourceFromDetail(*output.User, serverId), nil
}

// Delete deletes a Transfer Family user.
func (d *UserDAO) Delete(ctx context.Context, username string) error {
	serverId := dao.GetFilterFromContext(ctx, "ServerId")
	if serverId == "" {
		return fmt.Errorf("server ID filter required")
	}

	_, err := d.client.DeleteUser(ctx, &transfer.DeleteUserInput{
		ServerId: &serverId,
		UserName: &username,
	})
	if err != nil {
		return fmt.Errorf("delete transfer user %s: %w", username, err)
	}
	return nil
}

// UserResource wraps a Transfer Family user.
type UserResource struct {
	dao.BaseResource
	Summary  *types.ListedUser
	Detail   *types.DescribedUser
	ServerId string
}

// NewUserResource creates a new UserResource from summary.
func NewUserResource(user types.ListedUser, serverId string) *UserResource {
	return &UserResource{
		BaseResource: dao.BaseResource{
			ID:  appaws.Str(user.UserName),
			ARN: appaws.Str(user.Arn),
		},
		Summary:  &user,
		ServerId: serverId,
	}
}

// NewUserResourceFromDetail creates a new UserResource from detail.
func NewUserResourceFromDetail(user types.DescribedUser, serverId string) *UserResource {
	return &UserResource{
		BaseResource: dao.BaseResource{
			ID:  appaws.Str(user.UserName),
			ARN: appaws.Str(user.Arn),
		},
		Detail:   &user,
		ServerId: serverId,
	}
}

// UserName returns the username.
func (r *UserResource) UserName() string {
	return r.ID
}

// HomeDirectory returns the home directory.
func (r *UserResource) HomeDirectory() string {
	if r.Summary != nil {
		return appaws.Str(r.Summary.HomeDirectory)
	}
	if r.Detail != nil {
		return appaws.Str(r.Detail.HomeDirectory)
	}
	return ""
}

// HomeDirectoryType returns the home directory type.
func (r *UserResource) HomeDirectoryType() string {
	if r.Summary != nil {
		return string(r.Summary.HomeDirectoryType)
	}
	if r.Detail != nil {
		return string(r.Detail.HomeDirectoryType)
	}
	return ""
}

// Role returns the IAM role ARN.
func (r *UserResource) Role() string {
	if r.Summary != nil {
		return appaws.Str(r.Summary.Role)
	}
	if r.Detail != nil {
		return appaws.Str(r.Detail.Role)
	}
	return ""
}

// SshPublicKeyCount returns the SSH public key count.
func (r *UserResource) SshPublicKeyCount() int {
	if r.Summary != nil && r.Summary.SshPublicKeyCount != nil {
		return int(*r.Summary.SshPublicKeyCount)
	}
	if r.Detail != nil {
		return len(r.Detail.SshPublicKeys)
	}
	return 0
}

// Policy returns the session policy.
func (r *UserResource) Policy() string {
	if r.Detail != nil {
		return appaws.Str(r.Detail.Policy)
	}
	return ""
}

// PosixProfile returns the POSIX profile.
func (r *UserResource) PosixProfile() *types.PosixProfile {
	if r.Detail != nil {
		return r.Detail.PosixProfile
	}
	return nil
}

// HomeDirectoryMappings returns the home directory mappings.
func (r *UserResource) HomeDirectoryMappings() []types.HomeDirectoryMapEntry {
	if r.Detail != nil {
		return r.Detail.HomeDirectoryMappings
	}
	return nil
}

// Tags returns the user tags.
func (r *UserResource) Tags() []types.Tag {
	if r.Detail != nil {
		return r.Detail.Tags
	}
	return nil
}

// SshPublicKeys returns the SSH public keys.
func (r *UserResource) SshPublicKeys() []types.SshPublicKey {
	if r.Detail != nil {
		return r.Detail.SshPublicKeys
	}
	return nil
}
