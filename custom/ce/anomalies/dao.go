package anomalies

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

// AnomalyLookbackDays is the number of days to look back for anomalies.
const AnomalyLookbackDays = 90

// AnomalyDAO provides data access for Cost Anomaly Detection.
type AnomalyDAO struct {
	dao.BaseDAO
	client *costexplorer.Client
}

// NewAnomalyDAO creates a new AnomalyDAO.
func NewAnomalyDAO(ctx context.Context) (dao.DAO, error) {
	// Cost Explorer API is only available in us-east-1
	cfg, err := appaws.NewConfigWithRegion(ctx, appaws.CostExplorerRegion)
	if err != nil {
		return nil, apperrors.Wrap(err, "new ce/anomalies dao")
	}
	return &AnomalyDAO{
		BaseDAO: dao.NewBaseDAO("ce", "anomalies"),
		client:  costexplorer.NewFromConfig(cfg),
	}, nil
}

// List returns cost anomalies from the last 90 days.
func (d *AnomalyDAO) List(ctx context.Context) ([]dao.Resource, error) {
	// Get anomalies for lookback period (use UTC for consistency)
	now := time.Now().UTC()
	start := now.AddDate(0, 0, -AnomalyLookbackDays).Format("2006-01-02")
	end := now.Format("2006-01-02")

	anomalies, err := appaws.Paginate(ctx, func(token *string) ([]types.Anomaly, *string, error) {
		output, err := d.client.GetAnomalies(ctx, &costexplorer.GetAnomaliesInput{
			DateInterval: &types.AnomalyDateInterval{
				StartDate: &start,
				EndDate:   &end,
			},
			NextPageToken: token,
		})
		if err != nil {
			return nil, nil, apperrors.Wrap(err, "list anomalies")
		}
		return output.Anomalies, output.NextPageToken, nil
	})
	if err != nil {
		return nil, err
	}

	resources := make([]dao.Resource, len(anomalies))
	for i, anomaly := range anomalies {
		resources[i] = NewAnomalyResource(anomaly)
	}
	return resources, nil
}

// Get returns a specific anomaly by ID.
func (d *AnomalyDAO) Get(ctx context.Context, id string) (dao.Resource, error) {
	// GetAnomalies doesn't support filtering by ID, so we need to list and find
	resources, err := d.List(ctx)
	if err != nil {
		return nil, err
	}

	for _, r := range resources {
		if r.GetID() == id {
			return r, nil
		}
	}
	return nil, fmt.Errorf("anomaly not found: %s", id)
}

// Delete is not supported for anomalies.
func (d *AnomalyDAO) Delete(ctx context.Context, id string) error {
	return fmt.Errorf("delete not supported for cost anomalies")
}

// Supports returns true only for List operation.
// Get() is implemented via List() scan, so we disable auto-refresh in DetailView.
func (d *AnomalyDAO) Supports(op dao.Operation) bool {
	return op == dao.OpList
}

// AnomalyResource wraps a Cost Anomaly.
type AnomalyResource struct {
	dao.BaseResource
}

// NewAnomalyResource creates a new AnomalyResource.
func NewAnomalyResource(anomaly types.Anomaly) *AnomalyResource {
	return &AnomalyResource{
		BaseResource: dao.BaseResource{
			ID:   appaws.Str(anomaly.AnomalyId),
			Name: appaws.Str(anomaly.DimensionValue),
			Data: anomaly,
		},
	}
}

// item returns the underlying SDK type.
func (r *AnomalyResource) item() types.Anomaly {
	return r.Data.(types.Anomaly)
}

// DimensionValue returns the dimension (usually service name).
func (r *AnomalyResource) DimensionValue() string {
	return appaws.Str(r.item().DimensionValue)
}

// StartDate returns when the anomaly started.
func (r *AnomalyResource) StartDate() string {
	return appaws.Str(r.item().AnomalyStartDate)
}

// EndDate returns when the anomaly ended.
func (r *AnomalyResource) EndDate() string {
	return appaws.Str(r.item().AnomalyEndDate)
}

// TotalImpact returns the total cost impact.
func (r *AnomalyResource) TotalImpact() float64 {
	if r.item().Impact != nil {
		return r.item().Impact.TotalImpact
	}
	return 0
}

// TotalActualSpend returns the actual spend.
func (r *AnomalyResource) TotalActualSpend() float64 {
	if r.item().Impact != nil {
		return appaws.Float64(r.item().Impact.TotalActualSpend)
	}
	return 0
}

// TotalExpectedSpend returns the expected spend.
func (r *AnomalyResource) TotalExpectedSpend() float64 {
	if r.item().Impact != nil {
		return appaws.Float64(r.item().Impact.TotalExpectedSpend)
	}
	return 0
}

// TotalImpactPercentage returns the impact as a percentage.
func (r *AnomalyResource) TotalImpactPercentage() float64 {
	if r.item().Impact != nil {
		return appaws.Float64(r.item().Impact.TotalImpactPercentage)
	}
	return 0
}

// MaxScore returns the maximum anomaly score.
func (r *AnomalyResource) MaxScore() float64 {
	if r.item().AnomalyScore != nil {
		return r.item().AnomalyScore.MaxScore
	}
	return 0
}

// CurrentScore returns the current anomaly score.
func (r *AnomalyResource) CurrentScore() float64 {
	if r.item().AnomalyScore != nil {
		return r.item().AnomalyScore.CurrentScore
	}
	return 0
}

// MonitorArn returns the monitor ARN.
func (r *AnomalyResource) MonitorArn() string {
	return appaws.Str(r.item().MonitorArn)
}

// RootCauses returns the root causes.
func (r *AnomalyResource) RootCauses() []types.RootCause {
	return r.item().RootCauses
}

// Feedback returns the feedback status.
func (r *AnomalyResource) Feedback() string {
	return string(r.item().Feedback)
}
