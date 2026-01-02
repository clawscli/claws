package savingsplans

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/savingsplans"
	"github.com/aws/aws-sdk-go-v2/service/savingsplans/types"

	appaws "github.com/clawscli/claws/internal/aws"
	"github.com/clawscli/claws/internal/dao"
	apperrors "github.com/clawscli/claws/internal/errors"
)

// SavingsPlanDAO provides data access for Savings Plans
type SavingsPlanDAO struct {
	dao.BaseDAO
	client *savingsplans.Client
}

// NewSavingsPlanDAO creates a new SavingsPlanDAO
func NewSavingsPlanDAO(ctx context.Context) (dao.DAO, error) {
	cfg, err := appaws.NewConfig(ctx)
	if err != nil {
		return nil, apperrors.Wrap(err, "new risp/savings-plans dao")
	}
	return &SavingsPlanDAO{
		BaseDAO: dao.NewBaseDAO("risp", "savings-plans"),
		client:  savingsplans.NewFromConfig(cfg),
	}, nil
}

func (d *SavingsPlanDAO) List(ctx context.Context) ([]dao.Resource, error) {
	input := &savingsplans.DescribeSavingsPlansInput{}

	var resources []dao.Resource
	for {
		output, err := d.client.DescribeSavingsPlans(ctx, input)
		if err != nil {
			return nil, apperrors.Wrap(err, "describe savings plans")
		}

		for _, sp := range output.SavingsPlans {
			resources = append(resources, NewSavingsPlanResource(sp))
		}

		if output.NextToken == nil {
			break
		}
		input.NextToken = output.NextToken
	}

	return resources, nil
}

func (d *SavingsPlanDAO) Get(ctx context.Context, id string) (dao.Resource, error) {
	input := &savingsplans.DescribeSavingsPlansInput{
		SavingsPlanIds: []string{id},
	}

	output, err := d.client.DescribeSavingsPlans(ctx, input)
	if err != nil {
		return nil, apperrors.Wrapf(err, "describe savings plan %s", id)
	}

	if len(output.SavingsPlans) == 0 {
		return nil, fmt.Errorf("savings plan not found: %s", id)
	}

	return NewSavingsPlanResource(output.SavingsPlans[0]), nil
}

func (d *SavingsPlanDAO) Delete(ctx context.Context, id string) error {
	return fmt.Errorf("delete not supported for savings plans")
}

// SavingsPlanResource wraps a Savings Plan
type SavingsPlanResource struct {
	dao.BaseResource
	Item types.SavingsPlan
}

// NewSavingsPlanResource creates a new SavingsPlanResource
func NewSavingsPlanResource(sp types.SavingsPlan) *SavingsPlanResource {
	id := appaws.Str(sp.SavingsPlanId)
	return &SavingsPlanResource{
		BaseResource: dao.BaseResource{
			ID:   id,
			Name: id,
			Tags: sp.Tags,
			Data: sp,
		},
		Item: sp,
	}
}

// State returns the plan state
func (r *SavingsPlanResource) State() string {
	return string(r.Item.State)
}

// PlanType returns the savings plan type
func (r *SavingsPlanResource) PlanType() string {
	return string(r.Item.SavingsPlanType)
}

// Commitment returns the hourly commitment amount
func (r *SavingsPlanResource) Commitment() string {
	return appaws.Str(r.Item.Commitment)
}

// PaymentOption returns the payment option
func (r *SavingsPlanResource) PaymentOption() string {
	return string(r.Item.PaymentOption)
}

// ProductTypes returns the applicable product types
func (r *SavingsPlanResource) ProductTypes() string {
	var types []string
	for _, pt := range r.Item.ProductTypes {
		types = append(types, string(pt))
	}
	return strings.Join(types, ", ")
}

// Region returns the region (for EC2 Instance SP)
func (r *SavingsPlanResource) Region() string {
	return appaws.Str(r.Item.Region)
}

// EC2InstanceFamily returns the instance family (for EC2 Instance SP)
func (r *SavingsPlanResource) EC2InstanceFamily() string {
	return appaws.Str(r.Item.Ec2InstanceFamily)
}

// Currency returns the currency code
func (r *SavingsPlanResource) Currency() string {
	return string(r.Item.Currency)
}

// UpfrontPayment returns the upfront payment amount
func (r *SavingsPlanResource) UpfrontPayment() string {
	return appaws.Str(r.Item.UpfrontPaymentAmount)
}

// RecurringPayment returns the recurring payment amount
func (r *SavingsPlanResource) RecurringPayment() string {
	return appaws.Str(r.Item.RecurringPaymentAmount)
}

// Description returns the plan description
func (r *SavingsPlanResource) Description() string {
	return appaws.Str(r.Item.Description)
}

// StartTime returns the start time
func (r *SavingsPlanResource) StartTime() *time.Time {
	if r.Item.Start == nil {
		return nil
	}
	t, err := time.Parse(time.RFC3339, *r.Item.Start)
	if err != nil {
		return nil
	}
	return &t
}

// EndTime returns the end time
func (r *SavingsPlanResource) EndTime() *time.Time {
	if r.Item.End == nil {
		return nil
	}
	t, err := time.Parse(time.RFC3339, *r.Item.End)
	if err != nil {
		return nil
	}
	return &t
}

// Duration returns the term duration as a formatted string
func (r *SavingsPlanResource) Duration() string {
	seconds := r.Item.TermDurationInSeconds
	if seconds == 0 {
		return ""
	}
	years := seconds / (365 * 24 * 60 * 60)
	if years >= 1 {
		return fmt.Sprintf("%dy", years)
	}
	return fmt.Sprintf("%ds", seconds)
}

// ARN returns the plan ARN
func (r *SavingsPlanResource) ARN() string {
	return appaws.Str(r.Item.SavingsPlanArn)
}
