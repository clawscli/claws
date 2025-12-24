package monitors

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/costexplorer"
	"github.com/aws/aws-sdk-go-v2/service/costexplorer/types"
	appaws "github.com/clawscli/claws/internal/aws"
	"github.com/clawscli/claws/internal/dao"
)

// MonitorDAO provides data access for Cost Anomaly Monitors.
type MonitorDAO struct {
	dao.BaseDAO
	client *costexplorer.Client
}

// NewMonitorDAO creates a new MonitorDAO.
func NewMonitorDAO(ctx context.Context) (dao.DAO, error) {
	cfg, err := appaws.NewConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("new costexplorer/monitors dao: %w", err)
	}
	return &MonitorDAO{
		BaseDAO: dao.NewBaseDAO("cost-explorer", "monitors"),
		client:  costexplorer.NewFromConfig(cfg),
	}, nil
}

// List returns all anomaly monitors.
func (d *MonitorDAO) List(ctx context.Context) ([]dao.Resource, error) {
	var resources []dao.Resource
	var nextToken *string

	for {
		input := &costexplorer.GetAnomalyMonitorsInput{
			NextPageToken: nextToken,
		}

		output, err := d.client.GetAnomalyMonitors(ctx, input)
		if err != nil {
			return nil, fmt.Errorf("get anomaly monitors: %w", err)
		}

		for _, monitor := range output.AnomalyMonitors {
			resources = append(resources, NewMonitorResource(monitor))
		}

		if output.NextPageToken == nil {
			break
		}
		nextToken = output.NextPageToken
	}

	return resources, nil
}

// Get returns a specific monitor by ARN.
func (d *MonitorDAO) Get(ctx context.Context, id string) (dao.Resource, error) {
	output, err := d.client.GetAnomalyMonitors(ctx, &costexplorer.GetAnomalyMonitorsInput{
		MonitorArnList: []string{id},
	})
	if err != nil {
		return nil, fmt.Errorf("get anomaly monitor %s: %w", id, err)
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
