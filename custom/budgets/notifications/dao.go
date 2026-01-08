package notifications

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

// NotificationDAO provides data access for Budget notifications.
type NotificationDAO struct {
	dao.BaseDAO
	client    *budgets.Client
	stsClient *sts.Client
}

// NewNotificationDAO creates a new NotificationDAO.
func NewNotificationDAO(ctx context.Context) (dao.DAO, error) {
	cfg, err := appaws.NewConfig(ctx)
	if err != nil {
		return nil, apperrors.Wrap(err, "new "+ServiceResourcePath+" dao")
	}
	return &NotificationDAO{
		BaseDAO:   dao.NewBaseDAO("budgets", "notifications"),
		client:    budgets.NewFromConfig(cfg),
		stsClient: sts.NewFromConfig(cfg),
	}, nil
}

// List returns all notifications for a budget.
func (d *NotificationDAO) List(ctx context.Context) ([]dao.Resource, error) {
	budgetName := dao.GetFilterFromContext(ctx, "BudgetName")
	if budgetName == "" {
		return nil, fmt.Errorf("budget name filter required")
	}

	// Get account ID
	identity, err := d.stsClient.GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{})
	if err != nil {
		return nil, apperrors.Wrap(err, "get caller identity")
	}
	accountID := appaws.Str(identity.Account)

	notifications, err := appaws.Paginate(ctx, func(token *string) ([]types.Notification, *string, error) {
		output, err := d.client.DescribeNotificationsForBudget(ctx, &budgets.DescribeNotificationsForBudgetInput{
			AccountId:  &accountID,
			BudgetName: &budgetName,
			NextToken:  token,
		})
		if err != nil {
			return nil, nil, apperrors.Wrap(err, "describe notifications for budget")
		}
		return output.Notifications, output.NextToken, nil
	})
	if err != nil {
		return nil, err
	}

	resources := make([]dao.Resource, len(notifications))
	for i, notif := range notifications {
		resources[i] = NewNotificationResource(notif, budgetName, i)
	}
	return resources, nil
}

// Get returns a specific notification (by index as ID).
func (d *NotificationDAO) Get(ctx context.Context, id string) (dao.Resource, error) {
	resources, err := d.List(ctx)
	if err != nil {
		return nil, err
	}
	for _, r := range resources {
		if r.GetID() == id {
			return r, nil
		}
	}
	return nil, fmt.Errorf("notification not found: %s", id)
}

// Delete is not directly supported for notifications.
func (d *NotificationDAO) Delete(ctx context.Context, id string) error {
	return fmt.Errorf("delete not supported; use budget console to manage notifications")
}

// NotificationResource wraps a Budget notification.
type NotificationResource struct {
	dao.BaseResource
	Item       types.Notification
	BudgetName string
}

// NewNotificationResource creates a new NotificationResource.
func NewNotificationResource(notif types.Notification, budgetName string, index int) *NotificationResource {
	id := fmt.Sprintf("%s-%s-%s-%.2f",
		budgetName,
		string(notif.NotificationType),
		string(notif.ComparisonOperator),
		notif.Threshold,
	)
	return &NotificationResource{
		BaseResource: dao.BaseResource{
			ID:   id,
			ARN:  "",
			Data: notif,
		},
		Item:       notif,
		BudgetName: budgetName,
	}
}

// NotificationType returns the notification type.
func (r *NotificationResource) NotificationType() string {
	return string(r.Item.NotificationType)
}

// ComparisonOperator returns the comparison operator.
func (r *NotificationResource) ComparisonOperator() string {
	return string(r.Item.ComparisonOperator)
}

// Threshold returns the threshold percentage.
func (r *NotificationResource) Threshold() float64 {
	return r.Item.Threshold
}

// ThresholdType returns the threshold type (PERCENTAGE or ABSOLUTE_VALUE).
func (r *NotificationResource) ThresholdType() string {
	return string(r.Item.ThresholdType)
}

// NotificationState returns the notification state.
func (r *NotificationResource) NotificationState() string {
	return string(r.Item.NotificationState)
}
