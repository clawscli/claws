package costs

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/costexplorer"
	"github.com/aws/aws-sdk-go-v2/service/costexplorer/types"

	appaws "github.com/clawscli/claws/internal/aws"
	"github.com/clawscli/claws/internal/dao"
	apperrors "github.com/clawscli/claws/internal/errors"
)

// CostDAO provides data access for AWS Cost Explorer.
type CostDAO struct {
	dao.BaseDAO
	client *costexplorer.Client
}

// NewCostDAO creates a new CostDAO.
func NewCostDAO(ctx context.Context) (dao.DAO, error) {
	// Cost Explorer API is only available in us-east-1
	cfg, err := appaws.NewConfigWithRegion(ctx, appaws.CostExplorerRegion)
	if err != nil {
		return nil, apperrors.Wrap(err, "new ce/costs dao")
	}
	return &CostDAO{
		BaseDAO: dao.NewBaseDAO("ce", "costs"),
		client:  costexplorer.NewFromConfig(cfg),
	}, nil
}

// List returns costs grouped by service for the current month.
func (d *CostDAO) List(ctx context.Context) ([]dao.Resource, error) {
	// Get current month's date range (use UTC for consistency)
	now := time.Now().UTC()
	startOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	endOfMonth := startOfMonth.AddDate(0, 1, 0)

	start := startOfMonth.Format("2006-01-02")
	end := endOfMonth.Format("2006-01-02")

	output, err := d.client.GetCostAndUsage(ctx, &costexplorer.GetCostAndUsageInput{
		TimePeriod: &types.DateInterval{
			Start: &start,
			End:   &end,
		},
		Granularity: types.GranularityMonthly,
		Metrics:     []string{"UnblendedCost", "UsageQuantity"},
		GroupBy: []types.GroupDefinition{
			{
				Type: types.GroupDefinitionTypeDimension,
				Key:  appaws.StringPtr("SERVICE"),
			},
		},
	})
	if err != nil {
		return nil, apperrors.Wrap(err, "get cost and usage")
	}

	var resources []dao.Resource
	for _, result := range output.ResultsByTime {
		for _, group := range result.Groups {
			if len(group.Keys) > 0 {
				resources = append(resources, NewCostResource(group, start, end))
			}
		}
	}
	return resources, nil
}

// Get returns cost for a specific service.
func (d *CostDAO) Get(ctx context.Context, id string) (dao.Resource, error) {
	// Get current month's date range (use UTC for consistency)
	now := time.Now().UTC()
	startOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	endOfMonth := startOfMonth.AddDate(0, 1, 0)

	start := startOfMonth.Format("2006-01-02")
	end := endOfMonth.Format("2006-01-02")

	output, err := d.client.GetCostAndUsage(ctx, &costexplorer.GetCostAndUsageInput{
		TimePeriod: &types.DateInterval{
			Start: &start,
			End:   &end,
		},
		Granularity: types.GranularityMonthly,
		Metrics:     []string{"UnblendedCost", "UsageQuantity"},
		Filter: &types.Expression{
			Dimensions: &types.DimensionValues{
				Key:    types.DimensionService,
				Values: []string{id},
			},
		},
		GroupBy: []types.GroupDefinition{
			{
				Type: types.GroupDefinitionTypeDimension,
				Key:  appaws.StringPtr("SERVICE"),
			},
		},
	})
	if err != nil {
		return nil, apperrors.Wrapf(err, "get cost for service %s", id)
	}

	for _, result := range output.ResultsByTime {
		for _, group := range result.Groups {
			if len(group.Keys) > 0 && group.Keys[0] == id {
				return NewCostResource(group, start, end), nil
			}
		}
	}
	return nil, fmt.Errorf("cost data not found for service: %s", id)
}

// Delete is not supported for cost data.
func (d *CostDAO) Delete(ctx context.Context, id string) error {
	return fmt.Errorf("delete not supported for cost data")
}

// Supports returns true for List and Get operations only.
func (d *CostDAO) Supports(op dao.Operation) bool {
	return op == dao.OpList || op == dao.OpGet
}

// CostResource wraps AWS cost data for a service.
type CostResource struct {
	dao.BaseResource
	ServiceName   string
	Cost          string
	CostUnit      string
	UsageQuantity string
	UsageUnit     string
	StartDate     string
	EndDate       string
}

// NewCostResource creates a new CostResource.
func NewCostResource(group types.Group, start, end string) *CostResource {
	serviceName := ""
	if len(group.Keys) > 0 {
		serviceName = group.Keys[0]
	}

	cost := ""
	costUnit := ""
	usageQty := ""
	usageUnit := ""

	if m, ok := group.Metrics["UnblendedCost"]; ok {
		cost = appaws.Str(m.Amount)
		costUnit = appaws.Str(m.Unit)
	}
	if m, ok := group.Metrics["UsageQuantity"]; ok {
		usageQty = appaws.Str(m.Amount)
		usageUnit = appaws.Str(m.Unit)
	}

	return &CostResource{
		BaseResource: dao.BaseResource{
			ID: serviceName,
			// Pseudo-ARN: Cost Explorer aggregates don't have real ARNs.
			// Format "ce::<service>" enables internal resource identification.
			ARN: fmt.Sprintf("ce::%s", serviceName),
		},
		ServiceName:   serviceName,
		Cost:          cost,
		CostUnit:      costUnit,
		UsageQuantity: usageQty,
		UsageUnit:     usageUnit,
		StartDate:     start,
		EndDate:       end,
	}
}
