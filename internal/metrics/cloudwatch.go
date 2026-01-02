// Package metrics provides CloudWatch metrics fetching and sparkline rendering.
package metrics

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"

	appaws "github.com/clawscli/claws/internal/aws"
	"github.com/clawscli/claws/internal/config"
	"github.com/clawscli/claws/internal/render"
)

const (
	metricPeriod         = 60
	maxQueriesPerRequest = 500
)

type Fetcher struct {
	client *cloudwatch.Client
}

func NewFetcher(ctx context.Context) (*Fetcher, error) {
	cfg, err := appaws.NewConfig(ctx)
	if err != nil {
		return nil, err
	}
	return &Fetcher{client: cloudwatch.NewFromConfig(cfg)}, nil
}

func (f *Fetcher) Fetch(ctx context.Context, resourceIDs []string, spec *render.MetricSpec) (*MetricData, error) {
	if len(resourceIDs) == 0 || spec == nil {
		return NewMetricData(spec), nil
	}

	queries := f.buildQueries(resourceIDs, spec)
	endTime := time.Now().Truncate(time.Minute)
	startTime := endTime.Add(-config.File().MetricsWindow())

	data := NewMetricData(spec)

	for i := 0; i < len(queries); i += maxQueriesPerRequest {
		if ctx.Err() != nil {
			return data, ctx.Err()
		}

		end := i + maxQueriesPerRequest
		if end > len(queries) {
			end = len(queries)
		}
		batch := queries[i:end]

		input := &cloudwatch.GetMetricDataInput{
			StartTime:         aws.Time(startTime),
			EndTime:           aws.Time(endTime),
			MetricDataQueries: batch,
			ScanBy:            types.ScanByTimestampAscending,
		}

		output, err := f.client.GetMetricData(ctx, input)
		if err != nil {
			return nil, fmt.Errorf("GetMetricData failed: %w", err)
		}

		f.processResults(output.MetricDataResults, resourceIDs, data)
	}

	return data, nil
}

func (f *Fetcher) buildQueries(resourceIDs []string, spec *render.MetricSpec) []types.MetricDataQuery {
	queries := make([]types.MetricDataQuery, len(resourceIDs))
	for i, resourceID := range resourceIDs {
		queries[i] = types.MetricDataQuery{
			Id: aws.String(fmt.Sprintf("m%d", i)),
			MetricStat: &types.MetricStat{
				Metric: &types.Metric{
					Namespace:  aws.String(spec.Namespace),
					MetricName: aws.String(spec.MetricName),
					Dimensions: []types.Dimension{
						{
							Name:  aws.String(spec.DimensionName),
							Value: aws.String(resourceID),
						},
					},
				},
				Period: aws.Int32(metricPeriod),
				Stat:   aws.String(spec.Stat),
			},
		}
	}
	return queries
}

func (f *Fetcher) processResults(results []types.MetricDataResult, resourceIDs []string, data *MetricData) {
	idToResource := make(map[string]string, len(resourceIDs))
	for i, id := range resourceIDs {
		idToResource[fmt.Sprintf("m%d", i)] = id
	}

	for _, result := range results {
		queryID := aws.ToString(result.Id)
		resourceID, ok := idToResource[queryID]
		if !ok {
			continue
		}

		metricResult := &MetricResult{
			ResourceID: resourceID,
			Values:     result.Values,
			HasData:    len(result.Values) > 0,
		}
		if metricResult.HasData {
			metricResult.Latest = result.Values[len(result.Values)-1]
		}
		data.Results[resourceID] = metricResult
	}
}
