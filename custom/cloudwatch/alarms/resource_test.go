package alarms

import (
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
)

func TestNewMetricAlarmResource(t *testing.T) {
	now := time.Now()
	alarm := types.MetricAlarm{
		AlarmName:             aws.String("test-metric-alarm"),
		AlarmArn:              aws.String("arn:aws:cloudwatch:us-east-1:123456789012:alarm:test-metric-alarm"),
		StateValue:            types.StateValueAlarm,
		StateReason:           aws.String("Threshold crossed"),
		ActionsEnabled:        aws.Bool(true),
		AlarmDescription:      aws.String("Test alarm description"),
		StateUpdatedTimestamp: &now,
		Namespace:             aws.String("AWS/EC2"),
		MetricName:            aws.String("CPUUtilization"),
		Statistic:             types.StatisticAverage,
		Period:                aws.Int32(300),
		EvaluationPeriods:     aws.Int32(3),
		Threshold:             aws.Float64(80.0),
		ComparisonOperator:    types.ComparisonOperatorGreaterThanThreshold,
		Dimensions: []types.Dimension{
			{Name: aws.String("InstanceId"), Value: aws.String("i-1234567890abcdef0")},
		},
		AlarmActions: []string{"arn:aws:sns:us-east-1:123456789012:my-topic"},
		OKActions:    []string{"arn:aws:sns:us-east-1:123456789012:ok-topic"},
	}

	resource := NewMetricAlarmResource(alarm)

	tests := []struct {
		name     string
		got      interface{}
		expected interface{}
	}{
		{"GetID", resource.GetID(), "test-metric-alarm"},
		{"GetName", resource.GetName(), "test-metric-alarm"},
		{"ARN", resource.ARN, "arn:aws:cloudwatch:us-east-1:123456789012:alarm:test-metric-alarm"},
		{"AlarmType", resource.AlarmType, "Metric"},
		{"StateValue", resource.StateValue, "ALARM"},
		{"ActionsEnabled", resource.ActionsEnabled, true},
		{"Namespace", resource.Namespace, "AWS/EC2"},
		{"MetricName", resource.MetricName, "CPUUtilization"},
		{"Period", resource.Period, int32(300)},
		{"EvaluationPeriods", resource.EvaluationPeriods, int32(3)},
		{"IsMetricAlarm", resource.IsMetricAlarm(), true},
		{"IsCompositeAlarm", resource.IsCompositeAlarm(), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.expected {
				t.Errorf("%s = %v, want %v", tt.name, tt.got, tt.expected)
			}
		})
	}
}

func TestNewCompositeAlarmResource(t *testing.T) {
	now := time.Now()
	alarm := types.CompositeAlarm{
		AlarmName:             aws.String("test-composite-alarm"),
		AlarmArn:              aws.String("arn:aws:cloudwatch:us-east-1:123456789012:alarm:test-composite-alarm"),
		StateValue:            types.StateValueOk,
		StateReason:           aws.String("All child alarms OK"),
		ActionsEnabled:        aws.Bool(false),
		AlarmDescription:      aws.String("Composite alarm description"),
		StateUpdatedTimestamp: &now,
		AlarmRule:             aws.String("ALARM(child-alarm-1) OR ALARM(child-alarm-2)"),
		AlarmActions:          []string{"arn:aws:sns:us-east-1:123456789012:my-topic"},
	}

	resource := NewCompositeAlarmResource(alarm)

	tests := []struct {
		name     string
		got      interface{}
		expected interface{}
	}{
		{"GetID", resource.GetID(), "test-composite-alarm"},
		{"GetName", resource.GetName(), "test-composite-alarm"},
		{"AlarmType", resource.AlarmType, "Composite"},
		{"StateValue", resource.StateValue, "OK"},
		{"ActionsEnabled", resource.ActionsEnabled, false},
		{"AlarmRule", resource.AlarmRule, "ALARM(child-alarm-1) OR ALARM(child-alarm-2)"},
		{"IsMetricAlarm", resource.IsMetricAlarm(), false},
		{"IsCompositeAlarm", resource.IsCompositeAlarm(), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.expected {
				t.Errorf("%s = %v, want %v", tt.name, tt.got, tt.expected)
			}
		})
	}
}

func TestAlarmResource_ActionsEnabledStr(t *testing.T) {
	tests := []struct {
		enabled  bool
		expected string
	}{
		{true, "Enabled"},
		{false, "Disabled"},
	}

	for _, tt := range tests {
		alarm := types.MetricAlarm{
			AlarmName:      aws.String("test"),
			ActionsEnabled: aws.Bool(tt.enabled),
		}
		resource := NewMetricAlarmResource(alarm)
		if got := resource.ActionsEnabledStr(); got != tt.expected {
			t.Errorf("ActionsEnabledStr() with %v = %q, want %q", tt.enabled, got, tt.expected)
		}
	}
}

func TestAlarmResource_DimensionsStr(t *testing.T) {
	tests := []struct {
		name       string
		dimensions []types.Dimension
		expected   string
	}{
		{
			name:       "empty",
			dimensions: nil,
			expected:   "",
		},
		{
			name: "single",
			dimensions: []types.Dimension{
				{Name: aws.String("InstanceId"), Value: aws.String("i-123")},
			},
			expected: "InstanceId=i-123",
		},
		{
			name: "multiple",
			dimensions: []types.Dimension{
				{Name: aws.String("InstanceId"), Value: aws.String("i-123")},
				{Name: aws.String("AutoScalingGroupName"), Value: aws.String("my-asg")},
			},
			expected: "InstanceId=i-123, AutoScalingGroupName=my-asg",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			alarm := types.MetricAlarm{
				AlarmName:  aws.String("test"),
				Dimensions: tt.dimensions,
			}
			resource := NewMetricAlarmResource(alarm)
			if got := resource.DimensionsStr(); got != tt.expected {
				t.Errorf("DimensionsStr() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestAlarmResource_StateUpdatedStr(t *testing.T) {
	t.Run("with timestamp", func(t *testing.T) {
		ts := time.Date(2025, 12, 28, 10, 30, 45, 0, time.UTC)
		alarm := types.MetricAlarm{
			AlarmName:             aws.String("test"),
			StateUpdatedTimestamp: &ts,
		}
		resource := NewMetricAlarmResource(alarm)
		expected := "2025-12-28 10:30:45 UTC"
		if got := resource.StateUpdatedStr(); got != expected {
			t.Errorf("StateUpdatedStr() = %q, want %q", got, expected)
		}
	})

	t.Run("nil timestamp", func(t *testing.T) {
		alarm := types.MetricAlarm{
			AlarmName:             aws.String("test"),
			StateUpdatedTimestamp: nil,
		}
		resource := NewMetricAlarmResource(alarm)
		if got := resource.StateUpdatedStr(); got != "" {
			t.Errorf("StateUpdatedStr() = %q, want empty", got)
		}
	})
}

func TestAlarmResource_MinimalAlarm(t *testing.T) {
	alarm := types.MetricAlarm{
		AlarmName: aws.String("minimal"),
	}
	resource := NewMetricAlarmResource(alarm)

	tests := []struct {
		name     string
		got      interface{}
		expected interface{}
	}{
		{"GetID", resource.GetID(), "minimal"},
		{"Namespace", resource.Namespace, ""},
		{"MetricName", resource.MetricName, ""},
		{"DimensionsStr", resource.DimensionsStr(), ""},
		{"StateUpdatedStr", resource.StateUpdatedStr(), ""},
		{"ActionsEnabled", resource.ActionsEnabled, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.expected {
				t.Errorf("%s = %v, want %v", tt.name, tt.got, tt.expected)
			}
		})
	}
}

func TestAlarmResource_NilAlarmName(t *testing.T) {
	alarm := types.MetricAlarm{
		AlarmName: nil,
	}
	resource := NewMetricAlarmResource(alarm)

	if resource.GetID() != "" {
		t.Errorf("GetID() = %q, want empty for nil name", resource.GetID())
	}
	if resource.GetName() != "" {
		t.Errorf("GetName() = %q, want empty for nil name", resource.GetName())
	}
}

func TestAlarmResource_AlarmActions(t *testing.T) {
	alarm := types.MetricAlarm{
		AlarmName:               aws.String("test"),
		AlarmActions:            []string{"action1", "action2"},
		OKActions:               []string{"ok1"},
		InsufficientDataActions: []string{"insuf1", "insuf2", "insuf3"},
	}
	resource := NewMetricAlarmResource(alarm)

	if len(resource.AlarmActions) != 2 {
		t.Errorf("AlarmActions len = %d, want 2", len(resource.AlarmActions))
	}
	if len(resource.OKActions) != 1 {
		t.Errorf("OKActions len = %d, want 1", len(resource.OKActions))
	}
	if len(resource.InsufficientDataActions) != 3 {
		t.Errorf("InsufficientDataActions len = %d, want 3", len(resource.InsufficientDataActions))
	}
}
