package accounts

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/organizations"
	"github.com/aws/aws-sdk-go-v2/service/organizations/types"

	appaws "github.com/clawscli/claws/internal/aws"
	"github.com/clawscli/claws/internal/dao"
	apperrors "github.com/clawscli/claws/internal/errors"
)

// AccountDAO provides data access for Organizations accounts.
type AccountDAO struct {
	dao.BaseDAO
	client *organizations.Client
}

// NewAccountDAO creates a new AccountDAO.
func NewAccountDAO(ctx context.Context) (dao.DAO, error) {
	cfg, err := appaws.NewConfig(ctx)
	if err != nil {
		return nil, apperrors.Wrap(err, "new "+ServiceResourcePath+" dao")
	}
	return &AccountDAO{
		BaseDAO: dao.NewBaseDAO("organizations", "accounts"),
		client:  organizations.NewFromConfig(cfg),
	}, nil
}

// List returns all accounts in the organization.
func (d *AccountDAO) List(ctx context.Context) ([]dao.Resource, error) {
	accounts, err := appaws.Paginate(ctx, func(token *string) ([]types.Account, *string, error) {
		output, err := d.client.ListAccounts(ctx, &organizations.ListAccountsInput{
			NextToken: token,
		})
		if err != nil {
			return nil, nil, apperrors.Wrap(err, "list organizations accounts")
		}
		return output.Accounts, output.NextToken, nil
	})
	if err != nil {
		return nil, err
	}

	resources := make([]dao.Resource, len(accounts))
	for i, account := range accounts {
		resources[i] = NewAccountResource(account)
	}
	return resources, nil
}

// Get returns a specific account.
func (d *AccountDAO) Get(ctx context.Context, id string) (dao.Resource, error) {
	output, err := d.client.DescribeAccount(ctx, &organizations.DescribeAccountInput{
		AccountId: &id,
	})
	if err != nil {
		return nil, apperrors.Wrap(err, "describe organizations account")
	}
	return NewAccountResource(*output.Account), nil
}

// Delete removes an account from the organization.
func (d *AccountDAO) Delete(ctx context.Context, id string) error {
	_, err := d.client.RemoveAccountFromOrganization(ctx, &organizations.RemoveAccountFromOrganizationInput{
		AccountId: &id,
	})
	if err != nil {
		return apperrors.Wrap(err, "remove account from organization")
	}
	return nil
}

// AccountResource wraps an Organizations account.
type AccountResource struct {
	dao.BaseResource
	Account *types.Account
}

// NewAccountResource creates a new AccountResource.
func NewAccountResource(account types.Account) *AccountResource {
	return &AccountResource{
		BaseResource: dao.BaseResource{
			ID:  appaws.Str(account.Id),
			ARN: appaws.Str(account.Arn),
		},
		Account: &account,
	}
}

// Name returns the account name.
func (r *AccountResource) Name() string {
	if r.Account != nil && r.Account.Name != nil {
		return *r.Account.Name
	}
	return ""
}

// Email returns the account email.
func (r *AccountResource) Email() string {
	if r.Account != nil && r.Account.Email != nil {
		return *r.Account.Email
	}
	return ""
}

// Status returns the account status.
func (r *AccountResource) Status() string {
	if r.Account != nil {
		return string(r.Account.Status)
	}
	return ""
}

// JoinedMethod returns how the account joined.
func (r *AccountResource) JoinedMethod() string {
	if r.Account != nil {
		return string(r.Account.JoinedMethod)
	}
	return ""
}

// JoinedTimestamp returns when the account joined.
func (r *AccountResource) JoinedTimestamp() *time.Time {
	if r.Account != nil {
		return r.Account.JoinedTimestamp
	}
	return nil
}
