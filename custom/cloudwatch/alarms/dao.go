package alarms

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"

	appaws "github.com/clawscli/claws/internal/aws"
	"github.com/clawscli/claws/internal/dao"
)

// AlarmDAO provides data access for CloudWatch Alarms
type AlarmDAO struct {
	dao.BaseDAO
	client *cloudwatch.Client
}

// NewAlarmDAO creates a new AlarmDAO
func NewAlarmDAO(ctx context.Context) (dao.DAO, error) {
	cfg, err := appaws.NewConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("new cloudwatch/alarms dao: %w", err)
	}
	return &AlarmDAO{
		BaseDAO: dao.NewBaseDAO("cloudwatch", "alarms"),
		client:  cloudwatch.NewFromConfig(cfg),
	}, nil
}

func (d *AlarmDAO) List(ctx context.Context) ([]dao.Resource, error) {
	stateFilter := dao.GetFilterFromContext(ctx, "StateValue")

	input := &cloudwatch.DescribeAlarmsInput{}
	if stateFilter != "" {
		input.StateValue = types.StateValue(stateFilter)
	}

	var allMetricAlarms []types.MetricAlarm
	var allCompositeAlarms []types.CompositeAlarm

	// Manual pagination: API returns both MetricAlarms and CompositeAlarms
	paginator := cloudwatch.NewDescribeAlarmsPaginator(d.client, input)
	for paginator.HasMorePages() {
		output, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("describe alarms: %w", err)
		}
		allMetricAlarms = append(allMetricAlarms, output.MetricAlarms...)
		allCompositeAlarms = append(allCompositeAlarms, output.CompositeAlarms...)
	}

	resources := make([]dao.Resource, 0, len(allMetricAlarms)+len(allCompositeAlarms))
	for _, a := range allMetricAlarms {
		resources = append(resources, NewMetricAlarmResource(a))
	}
	for _, a := range allCompositeAlarms {
		resources = append(resources, NewCompositeAlarmResource(a))
	}

	return resources, nil
}

func (d *AlarmDAO) Get(ctx context.Context, id string) (dao.Resource, error) {
	input := &cloudwatch.DescribeAlarmsInput{
		AlarmNames: []string{id},
	}

	output, err := d.client.DescribeAlarms(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("describe alarm %s: %w", id, err)
	}

	for _, a := range output.MetricAlarms {
		if appaws.Str(a.AlarmName) == id {
			return NewMetricAlarmResource(a), nil
		}
	}

	for _, a := range output.CompositeAlarms {
		if appaws.Str(a.AlarmName) == id {
			return NewCompositeAlarmResource(a), nil
		}
	}

	return nil, fmt.Errorf("alarm not found: %s", id)
}

func (d *AlarmDAO) Delete(ctx context.Context, id string) error {
	input := &cloudwatch.DeleteAlarmsInput{
		AlarmNames: []string{id},
	}

	_, err := d.client.DeleteAlarms(ctx, input)
	if err != nil {
		if appaws.IsNotFound(err) {
			return nil // Already deleted
		}
		return fmt.Errorf("delete alarm %s: %w", id, err)
	}

	return nil
}

type AlarmResource struct {
	dao.BaseResource
	AlarmType                          string
	StateValue                         string
	StateReason                        string
	StateReasonData                    string
	StateUpdatedTimestamp              *time.Time
	StateTransitionedTimestamp         *time.Time
	ActionsEnabled                     bool
	AlarmActions                       []string
	OKActions                          []string
	InsufficientDataActions            []string
	AlarmDescription                   string
	AlarmConfigurationUpdatedTimestamp *time.Time

	Namespace                        string
	MetricName                       string
	Dimensions                       []types.Dimension
	Statistic                        string
	ExtendedStatistic                string
	Period                           int32
	EvaluationPeriods                int32
	DatapointsToAlarm                int32
	Threshold                        *float64
	ThresholdMetricId                string
	ComparisonOperator               string
	TreatMissingData                 string
	EvaluateLowSampleCountPercentile string
	Unit                             string
	Metrics                          []types.MetricDataQuery

	AlarmRule                        string
	ActionsSuppressor                string
	ActionsSuppressorExtensionPeriod int32
	ActionsSuppressorWaitPeriod      int32

	MetricAlarmItem    *types.MetricAlarm
	CompositeAlarmItem *types.CompositeAlarm
}

func NewMetricAlarmResource(a types.MetricAlarm) *AlarmResource {
	name := appaws.Str(a.AlarmName)

	r := &AlarmResource{
		BaseResource: dao.BaseResource{
			ID:   name,
			Name: name,
			ARN:  appaws.Str(a.AlarmArn),
			Data: a,
		},
		AlarmType:                          "Metric",
		StateValue:                         string(a.StateValue),
		StateReason:                        appaws.Str(a.StateReason),
		StateReasonData:                    appaws.Str(a.StateReasonData),
		StateUpdatedTimestamp:              a.StateUpdatedTimestamp,
		StateTransitionedTimestamp:         a.StateTransitionedTimestamp,
		ActionsEnabled:                     appaws.Bool(a.ActionsEnabled),
		AlarmDescription:                   appaws.Str(a.AlarmDescription),
		AlarmConfigurationUpdatedTimestamp: a.AlarmConfigurationUpdatedTimestamp,
		MetricAlarmItem:                    &a,

		Namespace:                        appaws.Str(a.Namespace),
		MetricName:                       appaws.Str(a.MetricName),
		Dimensions:                       a.Dimensions,
		Statistic:                        string(a.Statistic),
		ExtendedStatistic:                appaws.Str(a.ExtendedStatistic),
		Period:                           appaws.Int32(a.Period),
		EvaluationPeriods:                appaws.Int32(a.EvaluationPeriods),
		DatapointsToAlarm:                appaws.Int32(a.DatapointsToAlarm),
		Threshold:                        a.Threshold,
		ThresholdMetricId:                appaws.Str(a.ThresholdMetricId),
		ComparisonOperator:               string(a.ComparisonOperator),
		TreatMissingData:                 appaws.Str(a.TreatMissingData),
		EvaluateLowSampleCountPercentile: appaws.Str(a.EvaluateLowSampleCountPercentile),
		Unit:                             string(a.Unit),
		Metrics:                          a.Metrics,
	}

	r.AlarmActions = a.AlarmActions
	r.OKActions = a.OKActions
	r.InsufficientDataActions = a.InsufficientDataActions

	return r
}

func NewCompositeAlarmResource(a types.CompositeAlarm) *AlarmResource {
	name := appaws.Str(a.AlarmName)

	r := &AlarmResource{
		BaseResource: dao.BaseResource{
			ID:   name,
			Name: name,
			ARN:  appaws.Str(a.AlarmArn),
			Data: a,
		},
		AlarmType:                          "Composite",
		StateValue:                         string(a.StateValue),
		StateReason:                        appaws.Str(a.StateReason),
		StateReasonData:                    appaws.Str(a.StateReasonData),
		StateUpdatedTimestamp:              a.StateUpdatedTimestamp,
		StateTransitionedTimestamp:         a.StateTransitionedTimestamp,
		ActionsEnabled:                     appaws.Bool(a.ActionsEnabled),
		AlarmDescription:                   appaws.Str(a.AlarmDescription),
		AlarmConfigurationUpdatedTimestamp: a.AlarmConfigurationUpdatedTimestamp,
		CompositeAlarmItem:                 &a,

		AlarmRule:                        appaws.Str(a.AlarmRule),
		ActionsSuppressor:                appaws.Str(a.ActionsSuppressor),
		ActionsSuppressorExtensionPeriod: appaws.Int32(a.ActionsSuppressorExtensionPeriod),
		ActionsSuppressorWaitPeriod:      appaws.Int32(a.ActionsSuppressorWaitPeriod),
	}

	r.AlarmActions = a.AlarmActions
	r.OKActions = a.OKActions
	r.InsufficientDataActions = a.InsufficientDataActions

	return r
}

func (r *AlarmResource) IsMetricAlarm() bool {
	return r.AlarmType == "Metric"
}

func (r *AlarmResource) IsCompositeAlarm() bool {
	return r.AlarmType == "Composite"
}

func (r *AlarmResource) ActionsEnabledStr() string {
	if r.ActionsEnabled {
		return "Enabled"
	}
	return "Disabled"
}

func (r *AlarmResource) StateUpdatedStr() string {
	if r.StateUpdatedTimestamp != nil {
		return r.StateUpdatedTimestamp.Format("2006-01-02 15:04:05 MST")
	}
	return ""
}

func (r *AlarmResource) DimensionsStr() string {
	if len(r.Dimensions) == 0 {
		return ""
	}
	result := ""
	for i, d := range r.Dimensions {
		if i > 0 {
			result += ", "
		}
		result += fmt.Sprintf("%s=%s", appaws.Str(d.Name), appaws.Str(d.Value))
	}
	return result
}
