package users

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider/types"

	appaws "github.com/clawscli/claws/internal/aws"
	"github.com/clawscli/claws/internal/dao"
	apperrors "github.com/clawscli/claws/internal/errors"
)

// UserDAO provides data access for Cognito users
type UserDAO struct {
	dao.BaseDAO
	client *cognitoidentityprovider.Client
}

// NewUserDAO creates a new UserDAO
func NewUserDAO(ctx context.Context) (dao.DAO, error) {
	cfg, err := appaws.NewConfig(ctx)
	if err != nil {
		return nil, apperrors.Wrap(err, "new cognito-idp/users dao")
	}
	return &UserDAO{
		BaseDAO: dao.NewBaseDAO("cognito-idp", "users"),
		client:  cognitoidentityprovider.NewFromConfig(cfg),
	}, nil
}

// List returns users (first page only for backwards compatibility).
// For paginated access, use ListPage instead.
func (d *UserDAO) List(ctx context.Context) ([]dao.Resource, error) {
	resources, _, err := d.ListPage(ctx, 60, "")
	return resources, err
}

// ListPage returns a page of Cognito users.
// Implements dao.PaginatedDAO interface.
func (d *UserDAO) ListPage(ctx context.Context, pageSize int, pageToken string) ([]dao.Resource, string, error) {
	// Get user pool ID from filter context
	userPoolId := dao.GetFilterFromContext(ctx, "UserPoolId")
	if userPoolId == "" {
		return nil, "", fmt.Errorf("user pool ID filter required")
	}

	limit := int32(pageSize)
	if limit > 60 {
		limit = 60 // AWS API max
	}

	input := &cognitoidentityprovider.ListUsersInput{
		UserPoolId: &userPoolId,
		Limit:      &limit,
	}
	if pageToken != "" {
		input.PaginationToken = &pageToken
	}

	output, err := d.client.ListUsers(ctx, input)
	if err != nil {
		return nil, "", apperrors.Wrap(err, "list users")
	}

	resources := make([]dao.Resource, len(output.Users))
	for i, user := range output.Users {
		resources[i] = NewUserResource(user, userPoolId)
	}

	nextToken := ""
	if output.PaginationToken != nil {
		nextToken = *output.PaginationToken
	}

	return resources, nextToken, nil
}

// Get returns a specific user
func (d *UserDAO) Get(ctx context.Context, username string) (dao.Resource, error) {
	userPoolId := dao.GetFilterFromContext(ctx, "UserPoolId")
	if userPoolId == "" {
		return nil, fmt.Errorf("user pool ID filter required")
	}

	input := &cognitoidentityprovider.AdminGetUserInput{
		UserPoolId: &userPoolId,
		Username:   &username,
	}

	output, err := d.client.AdminGetUser(ctx, input)
	if err != nil {
		return nil, apperrors.Wrapf(err, "get user %s", username)
	}

	return NewUserResourceFromDetail(output, userPoolId), nil
}

// Delete deletes a Cognito user
func (d *UserDAO) Delete(ctx context.Context, username string) error {
	userPoolId := dao.GetFilterFromContext(ctx, "UserPoolId")
	if userPoolId == "" {
		return fmt.Errorf("user pool ID filter required")
	}

	_, err := d.client.AdminDeleteUser(ctx, &cognitoidentityprovider.AdminDeleteUserInput{
		UserPoolId: &userPoolId,
		Username:   &username,
	})
	if err != nil {
		return apperrors.Wrapf(err, "delete user %s", username)
	}
	return nil
}

// Supports returns supported operations
func (d *UserDAO) Supports(op dao.Operation) bool {
	switch op {
	case dao.OpList, dao.OpGet, dao.OpDelete:
		return true
	default:
		return false
	}
}

// UserResource represents a Cognito user
type UserResource struct {
	dao.BaseResource
	User       *types.UserType
	Detail     *cognitoidentityprovider.AdminGetUserOutput
	UserPoolId string
}

// NewUserResource creates a new UserResource from list
func NewUserResource(user types.UserType, userPoolId string) *UserResource {
	username := appaws.Str(user.Username)

	return &UserResource{
		BaseResource: dao.BaseResource{
			ID:   username,
			Name: username,
			ARN:  "",
			Tags: make(map[string]string),
			Data: user,
		},
		User:       &user,
		UserPoolId: userPoolId,
	}
}

// NewUserResourceFromDetail creates a new UserResource from detail
func NewUserResourceFromDetail(detail *cognitoidentityprovider.AdminGetUserOutput, userPoolId string) *UserResource {
	username := appaws.Str(detail.Username)

	return &UserResource{
		BaseResource: dao.BaseResource{
			ID:   username,
			Name: username,
			ARN:  "",
			Tags: make(map[string]string),
			Data: detail,
		},
		Detail:     detail,
		UserPoolId: userPoolId,
	}
}

// Username returns the username
func (r *UserResource) Username() string {
	if r.User != nil {
		return appaws.Str(r.User.Username)
	}
	if r.Detail != nil {
		return appaws.Str(r.Detail.Username)
	}
	return ""
}

// Status returns the user status
func (r *UserResource) Status() string {
	if r.User != nil {
		return string(r.User.UserStatus)
	}
	if r.Detail != nil {
		return string(r.Detail.UserStatus)
	}
	return ""
}

// Enabled returns whether the user is enabled
func (r *UserResource) Enabled() bool {
	if r.User != nil {
		return r.User.Enabled
	}
	if r.Detail != nil {
		return r.Detail.Enabled
	}
	return false
}

// Email returns the user's email
func (r *UserResource) Email() string {
	return r.getAttribute("email")
}

// PhoneNumber returns the user's phone number
func (r *UserResource) PhoneNumber() string {
	return r.getAttribute("phone_number")
}

// Name returns the user's name
func (r *UserResource) Name() string {
	return r.getAttribute("name")
}

// GivenName returns the user's given name
func (r *UserResource) GivenName() string {
	return r.getAttribute("given_name")
}

// FamilyName returns the user's family name
func (r *UserResource) FamilyName() string {
	return r.getAttribute("family_name")
}

func (r *UserResource) getAttribute(name string) string {
	var attrs []types.AttributeType
	if r.User != nil {
		attrs = r.User.Attributes
	} else if r.Detail != nil {
		attrs = r.Detail.UserAttributes
	}

	for _, attr := range attrs {
		if attr.Name != nil && *attr.Name == name {
			return appaws.Str(attr.Value)
		}
	}
	return ""
}

// Attributes returns all user attributes
func (r *UserResource) Attributes() map[string]string {
	result := make(map[string]string)
	var attrs []types.AttributeType

	if r.User != nil {
		attrs = r.User.Attributes
	} else if r.Detail != nil {
		attrs = r.Detail.UserAttributes
	}

	for _, attr := range attrs {
		if attr.Name != nil && attr.Value != nil {
			result[*attr.Name] = *attr.Value
		}
	}
	return result
}

// CreatedAt returns the creation date
func (r *UserResource) CreatedAt() string {
	if r.User != nil && r.User.UserCreateDate != nil {
		return r.User.UserCreateDate.Format("2006-01-02 15:04:05")
	}
	if r.Detail != nil && r.Detail.UserCreateDate != nil {
		return r.Detail.UserCreateDate.Format("2006-01-02 15:04:05")
	}
	return ""
}

// CreatedAtTime returns the creation date as time.Time
func (r *UserResource) CreatedAtTime() *time.Time {
	if r.User != nil {
		return r.User.UserCreateDate
	}
	if r.Detail != nil {
		return r.Detail.UserCreateDate
	}
	return nil
}

// LastModifiedDate returns the last modified date
func (r *UserResource) LastModifiedDate() string {
	if r.User != nil && r.User.UserLastModifiedDate != nil {
		return r.User.UserLastModifiedDate.Format("2006-01-02 15:04:05")
	}
	if r.Detail != nil && r.Detail.UserLastModifiedDate != nil {
		return r.Detail.UserLastModifiedDate.Format("2006-01-02 15:04:05")
	}
	return ""
}
