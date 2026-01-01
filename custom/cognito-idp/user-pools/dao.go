package userpools

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider/types"

	appaws "github.com/clawscli/claws/internal/aws"
	"github.com/clawscli/claws/internal/dao"
	apperrors "github.com/clawscli/claws/internal/errors"
)

// UserPoolDAO provides data access for Cognito user pools
type UserPoolDAO struct {
	dao.BaseDAO
	client *cognitoidentityprovider.Client
}

// NewUserPoolDAO creates a new UserPoolDAO
func NewUserPoolDAO(ctx context.Context) (dao.DAO, error) {
	cfg, err := appaws.NewConfig(ctx)
	if err != nil {
		return nil, apperrors.Wrap(err, "new cognito-idp/user-pools dao")
	}
	return &UserPoolDAO{
		BaseDAO: dao.NewBaseDAO("cognito-idp", "user-pools"),
		client:  cognitoidentityprovider.NewFromConfig(cfg),
	}, nil
}

// List returns all Cognito user pools
func (d *UserPoolDAO) List(ctx context.Context) ([]dao.Resource, error) {
	pools, err := appaws.Paginate(ctx, func(token *string) ([]types.UserPoolDescriptionType, *string, error) {
		output, err := d.client.ListUserPools(ctx, &cognitoidentityprovider.ListUserPoolsInput{
			NextToken:  token,
			MaxResults: appaws.Int32Ptr(60),
		})
		if err != nil {
			return nil, nil, apperrors.Wrap(err, "list user pools")
		}
		return output.UserPools, output.NextToken, nil
	})
	if err != nil {
		return nil, err
	}

	resources := make([]dao.Resource, len(pools))
	for i, pool := range pools {
		resources[i] = NewUserPoolResource(pool)
	}

	return resources, nil
}

// Get returns a specific Cognito user pool by ID
func (d *UserPoolDAO) Get(ctx context.Context, id string) (dao.Resource, error) {
	input := &cognitoidentityprovider.DescribeUserPoolInput{
		UserPoolId: &id,
	}

	output, err := d.client.DescribeUserPool(ctx, input)
	if err != nil {
		return nil, apperrors.Wrapf(err, "describe user pool %s", id)
	}

	return NewUserPoolResourceFromDetail(output.UserPool), nil
}

// Delete deletes a Cognito user pool
func (d *UserPoolDAO) Delete(ctx context.Context, id string) error {
	input := &cognitoidentityprovider.DeleteUserPoolInput{
		UserPoolId: &id,
	}

	_, err := d.client.DeleteUserPool(ctx, input)
	if err != nil {
		return apperrors.Wrapf(err, "delete user pool %s", id)
	}

	return nil
}

// UserPoolResource represents a Cognito user pool
type UserPoolResource struct {
	dao.BaseResource
	Summary *types.UserPoolDescriptionType
	Detail  *types.UserPoolType
}

// NewUserPoolResource creates a new UserPoolResource from summary
func NewUserPoolResource(summary types.UserPoolDescriptionType) *UserPoolResource {
	poolId := appaws.Str(summary.Id)
	name := appaws.Str(summary.Name)

	return &UserPoolResource{
		BaseResource: dao.BaseResource{
			ID:   poolId,
			Name: name,
			ARN:  "", // ARN not available in summary
			Tags: make(map[string]string),
			Data: summary,
		},
		Summary: &summary,
	}
}

// NewUserPoolResourceFromDetail creates a new UserPoolResource from detail
func NewUserPoolResourceFromDetail(detail *types.UserPoolType) *UserPoolResource {
	poolId := appaws.Str(detail.Id)
	name := appaws.Str(detail.Name)
	arn := appaws.Str(detail.Arn)

	return &UserPoolResource{
		BaseResource: dao.BaseResource{
			ID:   poolId,
			Name: name,
			ARN:  arn,
			Tags: make(map[string]string),
			Data: detail,
		},
		Detail: detail,
	}
}

// PoolId returns the user pool ID
func (r *UserPoolResource) PoolId() string {
	if r.Summary != nil {
		return appaws.Str(r.Summary.Id)
	}
	if r.Detail != nil {
		return appaws.Str(r.Detail.Id)
	}
	return ""
}

// PoolName returns the user pool name
func (r *UserPoolResource) PoolName() string {
	if r.Summary != nil {
		return appaws.Str(r.Summary.Name)
	}
	if r.Detail != nil {
		return appaws.Str(r.Detail.Name)
	}
	return ""
}

// Status returns the user pool status
func (r *UserPoolResource) Status() string {
	// User pools that exist are always active
	// The Status field is deprecated in the AWS SDK
	if r.Summary != nil || r.Detail != nil {
		return "Active"
	}
	return ""
}

// CreatedAt returns the creation date
func (r *UserPoolResource) CreatedAt() string {
	if r.Summary != nil && r.Summary.CreationDate != nil {
		return r.Summary.CreationDate.Format("2006-01-02 15:04:05")
	}
	if r.Detail != nil && r.Detail.CreationDate != nil {
		return r.Detail.CreationDate.Format("2006-01-02 15:04:05")
	}
	return ""
}

// LastModifiedDate returns the last modified date
func (r *UserPoolResource) LastModifiedDate() string {
	if r.Summary != nil && r.Summary.LastModifiedDate != nil {
		return r.Summary.LastModifiedDate.Format("2006-01-02 15:04:05")
	}
	if r.Detail != nil && r.Detail.LastModifiedDate != nil {
		return r.Detail.LastModifiedDate.Format("2006-01-02 15:04:05")
	}
	return ""
}

// MfaConfiguration returns the MFA configuration
func (r *UserPoolResource) MfaConfiguration() string {
	if r.Detail != nil {
		return string(r.Detail.MfaConfiguration)
	}
	return ""
}

// EstimatedNumberOfUsers returns the estimated number of users
func (r *UserPoolResource) EstimatedNumberOfUsers() int32 {
	if r.Detail != nil {
		return r.Detail.EstimatedNumberOfUsers
	}
	return 0
}

// Domain returns the domain prefix
func (r *UserPoolResource) Domain() string {
	if r.Detail != nil {
		return appaws.Str(r.Detail.Domain)
	}
	return ""
}

// CustomDomain returns the custom domain
func (r *UserPoolResource) CustomDomain() string {
	if r.Detail != nil {
		return appaws.Str(r.Detail.CustomDomain)
	}
	return ""
}

// DeletionProtection returns whether deletion protection is enabled
func (r *UserPoolResource) DeletionProtection() string {
	if r.Detail != nil {
		return string(r.Detail.DeletionProtection)
	}
	return ""
}

// UsernameAttributes returns the username attributes
func (r *UserPoolResource) UsernameAttributes() []string {
	if r.Detail != nil {
		var attrs []string
		for _, attr := range r.Detail.UsernameAttributes {
			attrs = append(attrs, string(attr))
		}
		return attrs
	}
	return nil
}

// AutoVerifiedAttributes returns the auto-verified attributes
func (r *UserPoolResource) AutoVerifiedAttributes() []string {
	if r.Detail != nil {
		var attrs []string
		for _, attr := range r.Detail.AutoVerifiedAttributes {
			attrs = append(attrs, string(attr))
		}
		return attrs
	}
	return nil
}

// LambdaConfig returns information about Lambda triggers
func (r *UserPoolResource) LambdaConfig() map[string]string {
	if r.Detail == nil || r.Detail.LambdaConfig == nil {
		return nil
	}
	config := make(map[string]string)
	lc := r.Detail.LambdaConfig
	if lc.PreSignUp != nil {
		config["PreSignUp"] = *lc.PreSignUp
	}
	if lc.PostConfirmation != nil {
		config["PostConfirmation"] = *lc.PostConfirmation
	}
	if lc.PreAuthentication != nil {
		config["PreAuthentication"] = *lc.PreAuthentication
	}
	if lc.PostAuthentication != nil {
		config["PostAuthentication"] = *lc.PostAuthentication
	}
	if lc.DefineAuthChallenge != nil {
		config["DefineAuthChallenge"] = *lc.DefineAuthChallenge
	}
	if lc.CreateAuthChallenge != nil {
		config["CreateAuthChallenge"] = *lc.CreateAuthChallenge
	}
	if lc.VerifyAuthChallengeResponse != nil {
		config["VerifyAuthChallengeResponse"] = *lc.VerifyAuthChallengeResponse
	}
	if lc.PreTokenGeneration != nil {
		config["PreTokenGeneration"] = *lc.PreTokenGeneration
	}
	if lc.UserMigration != nil {
		config["UserMigration"] = *lc.UserMigration
	}
	if lc.CustomMessage != nil {
		config["CustomMessage"] = *lc.CustomMessage
	}
	return config
}
