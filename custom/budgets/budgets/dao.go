package budgets

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/budgets"
	"github.com/aws/aws-sdk-go-v2/service/budgets/types"
	"github.com/aws/aws-sdk-go-v2/service/sts"

	appaws "github.com/clawscli/claws/internal/aws"
	"github.com/clawscli/claws/internal/dao"
	apperrors "github.com/clawscli/claws/internal/errors"
)

// BudgetDAO provides data access for AWS Budgets.
type BudgetDAO struct {
	dao.BaseDAO
	client    *budgets.Client
	stsClient *sts.Client
}

// NewBudgetDAO creates a new BudgetDAO.
func NewBudgetDAO(ctx context.Context) (dao.DAO, error) {
	cfg, err := appaws.NewConfig(ctx)
	if err != nil {
		return nil, apperrors.Wrap(err, "new "+ServiceResourcePath+" dao")
	}
	return &BudgetDAO{
		BaseDAO:   dao.NewBaseDAO("budgets", "budgets"),
		client:    budgets.NewFromConfig(cfg),
		stsClient: sts.NewFromConfig(cfg),
	}, nil
}

// List returns all budgets.
func (d *BudgetDAO) List(ctx context.Context) ([]dao.Resource, error) {
	// Get account ID
	identity, err := d.stsClient.GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{})
	if err != nil {
		return nil, apperrors.Wrap(err, "get caller identity")
	}
	accountID := appaws.Str(identity.Account)

	budgetList, err := appaws.Paginate(ctx, func(token *string) ([]types.Budget, *string, error) {
		output, err := d.client.DescribeBudgets(ctx, &budgets.DescribeBudgetsInput{
			AccountId: &accountID,
			NextToken: token,
		})
		if err != nil {
			return nil, nil, apperrors.Wrap(err, "describe budgets")
		}
		return output.Budgets, output.NextToken, nil
	})
	if err != nil {
		return nil, err
	}

	resources := make([]dao.Resource, len(budgetList))
	for i, budget := range budgetList {
		resources[i] = NewBudgetResource(budget, accountID)
	}
	return resources, nil
}

// Get returns a specific budget by name.
func (d *BudgetDAO) Get(ctx context.Context, id string) (dao.Resource, error) {
	// Get account ID
	identity, err := d.stsClient.GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{})
	if err != nil {
		return nil, apperrors.Wrap(err, "get caller identity")
	}
	accountID := appaws.Str(identity.Account)

	output, err := d.client.DescribeBudget(ctx, &budgets.DescribeBudgetInput{
		AccountId:  &accountID,
		BudgetName: &id,
	})
	if err != nil {
		return nil, apperrors.Wrapf(err, "describe budget %s", id)
	}
	return NewBudgetResource(*output.Budget, accountID), nil
}

// Delete deletes a budget by name.
func (d *BudgetDAO) Delete(ctx context.Context, id string) error {
	// Get account ID
	identity, err := d.stsClient.GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{})
	if err != nil {
		return apperrors.Wrap(err, "get caller identity")
	}
	accountID := appaws.Str(identity.Account)

	_, err = d.client.DeleteBudget(ctx, &budgets.DeleteBudgetInput{
		AccountId:  &accountID,
		BudgetName: &id,
	})
	if err != nil {
		return apperrors.Wrapf(err, "delete budget %s", id)
	}
	return nil
}

// BudgetResource wraps an AWS Budget.
type BudgetResource struct {
	dao.BaseResource
	Item      types.Budget
	AccountID string
}

// NewBudgetResource creates a new BudgetResource.
func NewBudgetResource(budget types.Budget, accountID string) *BudgetResource {
	return &BudgetResource{
		BaseResource: dao.BaseResource{
			ID:   appaws.Str(budget.BudgetName),
			ARN:  fmt.Sprintf("arn:aws:budgets::%s:budget/%s", accountID, appaws.Str(budget.BudgetName)),
			Data: budget,
		},
		Item:      budget,
		AccountID: accountID,
	}
}

// Name returns the budget name.
func (r *BudgetResource) Name() string {
	return appaws.Str(r.Item.BudgetName)
}

// BudgetType returns the budget type.
func (r *BudgetResource) BudgetType() string {
	return string(r.Item.BudgetType)
}

// TimeUnit returns the time unit.
func (r *BudgetResource) TimeUnit() string {
	return string(r.Item.TimeUnit)
}

// BudgetLimit returns the budget limit.
func (r *BudgetResource) BudgetLimit() (string, string) {
	if r.Item.BudgetLimit != nil {
		return appaws.Str(r.Item.BudgetLimit.Amount), appaws.Str(r.Item.BudgetLimit.Unit)
	}
	return "", ""
}

// ActualSpend returns the actual spend.
func (r *BudgetResource) ActualSpend() (string, string) {
	if r.Item.CalculatedSpend != nil && r.Item.CalculatedSpend.ActualSpend != nil {
		return appaws.Str(r.Item.CalculatedSpend.ActualSpend.Amount), appaws.Str(r.Item.CalculatedSpend.ActualSpend.Unit)
	}
	return "", ""
}

// ForecastedSpend returns the forecasted spend.
func (r *BudgetResource) ForecastedSpend() (string, string) {
	if r.Item.CalculatedSpend != nil && r.Item.CalculatedSpend.ForecastedSpend != nil {
		return appaws.Str(r.Item.CalculatedSpend.ForecastedSpend.Amount), appaws.Str(r.Item.CalculatedSpend.ForecastedSpend.Unit)
	}
	return "", ""
}
