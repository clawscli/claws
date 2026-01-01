package monitors

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/costexplorer"
	"github.com/aws/aws-sdk-go-v2/service/costexplorer/types"

	appaws "github.com/clawscli/claws/internal/aws"
	"github.com/clawscli/claws/internal/dao"
	apperrors "github.com/clawscli/claws/internal/errors"
)

// MonitorDAO provides data access for Cost Anomaly Monitors.
type MonitorDAO struct {
	dao.BaseDAO
	client *costexplorer.Client
}

// NewMonitorDAO creates a new MonitorDAO.
func NewMonitorDAO(ctx context.Context) (dao.DAO, error) {
	// Cost Explorer API is only available in us-east-1
	cfg, err := appaws.NewConfigWithRegion(ctx, appaws.CostExplorerRegion)
	if err != nil {
		return nil, apperrors.Wrap(err, "new ce/monitors dao")
	}
	return &MonitorDAO{
		BaseDAO: dao.NewBaseDAO("ce", "monitors"),
		client:  costexplorer.NewFromConfig(cfg),
	}, nil
}

// List returns all anomaly monitors.
func (d *MonitorDAO) List(ctx context.Context) ([]dao.Resource, error) {
	monitors, err := appaws.Paginate(ctx, func(token *string) ([]types.AnomalyMonitor, *string, error) {
		output, err := d.client.GetAnomalyMonitors(ctx, &costexplorer.GetAnomalyMonitorsInput{
			NextPageToken: token,
		})
		if err != nil {
			return nil, nil, apperrors.Wrap(err, "list anomaly monitors")
		}
		return output.AnomalyMonitors, output.NextPageToken, nil
	})
	if err != nil {
		return nil, err
	}

	resources := make([]dao.Resource, len(monitors))
	for i, monitor := range monitors {
		resources[i] = NewMonitorResource(monitor)
	}
	return resources, nil
}

// Get returns a specific monitor by ARN.
func (d *MonitorDAO) Get(ctx context.Context, id string) (dao.Resource, error) {
	output, err := d.client.GetAnomalyMonitors(ctx, &costexplorer.GetAnomalyMonitorsInput{
		MonitorArnList: []string{id},
	})
	if err != nil {
		return nil, apperrors.Wrapf(err, "get anomaly monitor %s", id)
	}

	if len(output.AnomalyMonitors) == 0 {
		return nil, fmt.Errorf("monitor not found: %s", id)
	}

	return NewMonitorResource(output.AnomalyMonitors[0]), nil
}

// Delete is not supported for monitors (requires DeleteAnomalyMonitor API).
func (d *MonitorDAO) Delete(ctx context.Context, id string) error {
	return fmt.Errorf("delete not supported for cost anomaly monitors")
}

// Supports returns true for List and Get operations only.
func (d *MonitorDAO) Supports(op dao.Operation) bool {
	return op == dao.OpList || op == dao.OpGet
}

// MonitorResource wraps a Cost Anomaly Monitor.
type MonitorResource struct {
	dao.BaseResource
}

// NewMonitorResource creates a new MonitorResource.
func NewMonitorResource(monitor types.AnomalyMonitor) *MonitorResource {
	arn := appaws.Str(monitor.MonitorArn)
	name := appaws.Str(monitor.MonitorName)

	return &MonitorResource{
		BaseResource: dao.BaseResource{
			ID:   arn,
			Name: name,
			ARN:  arn,
			Data: monitor,
		},
	}
}

// item returns the underlying SDK type.
func (r *MonitorResource) item() types.AnomalyMonitor {
	return r.Data.(types.AnomalyMonitor)
}

// MonitorName returns the monitor name.
func (r *MonitorResource) MonitorName() string {
	return appaws.Str(r.item().MonitorName)
}

// MonitorType returns the monitor type.
func (r *MonitorResource) MonitorType() string {
	return string(r.item().MonitorType)
}

// MonitorDimension returns the dimension being monitored.
func (r *MonitorResource) MonitorDimension() string {
	return string(r.item().MonitorDimension)
}

// CreationDate returns the creation date.
func (r *MonitorResource) CreationDate() string {
	return appaws.Str(r.item().CreationDate)
}

// LastEvaluatedDate returns when the monitor last evaluated.
func (r *MonitorResource) LastEvaluatedDate() string {
	return appaws.Str(r.item().LastEvaluatedDate)
}

// LastUpdatedDate returns when the monitor was last updated.
func (r *MonitorResource) LastUpdatedDate() string {
	return appaws.Str(r.item().LastUpdatedDate)
}

// DimensionalValueCount returns the count of evaluated dimensions.
func (r *MonitorResource) DimensionalValueCount() int32 {
	return r.item().DimensionalValueCount
}
