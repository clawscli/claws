package alarms

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/render"
)

var _ render.Navigator = (*AlarmRenderer)(nil)

type AlarmRenderer struct {
	render.BaseRenderer
}

func NewAlarmRenderer() render.Renderer {
	return &AlarmRenderer{
		BaseRenderer: render.BaseRenderer{
			Service:  "cloudwatch",
			Resource: "alarms",
			Cols: []render.Column{
				{Name: "NAME", Width: 40, Getter: getName},
				{Name: "TYPE", Width: 10, Getter: getType},
				{Name: "STATE", Width: 18, Getter: getState},
				{Name: "ACTIONS", Width: 8, Getter: getActions},
				{Name: "UPDATED", Width: 20, Getter: getUpdated},
			},
		},
	}
}

func getName(r dao.Resource) string {
	alarm, ok := r.(*AlarmResource)
	if !ok {
		return ""
	}
	return alarm.GetName()
}

func getType(r dao.Resource) string {
	alarm, ok := r.(*AlarmResource)
	if !ok {
		return ""
	}
	return alarm.AlarmType
}

func getState(r dao.Resource) string {
	alarm, ok := r.(*AlarmResource)
	if !ok {
		return ""
	}
	return alarm.StateValue
}

func getActions(r dao.Resource) string {
	alarm, ok := r.(*AlarmResource)
	if !ok {
		return ""
	}
	return alarm.ActionsEnabledStr()
}

func getUpdated(r dao.Resource) string {
	alarm, ok := r.(*AlarmResource)
	if !ok {
		return ""
	}
	return alarm.StateUpdatedStr()
}

func (r *AlarmRenderer) RenderDetail(resource dao.Resource) string {
	alarm, ok := resource.(*AlarmResource)
	if !ok {
		return ""
	}

	d := render.NewDetailBuilder()

	d.Title("CloudWatch Alarm", alarm.GetName())

	d.Section("Basic Information")
	d.Field("Name", alarm.GetName())
	d.Field("ARN", alarm.GetARN())
	d.Field("Type", alarm.AlarmType)
	if alarm.AlarmDescription != "" {
		d.Field("Description", alarm.AlarmDescription)
	}

	d.Section("State")
	d.Field("State Value", alarm.StateValue)
	if alarm.StateReason != "" {
		d.Field("State Reason", alarm.StateReason)
	}
	if alarm.StateReasonData != "" {
		d.Field("State Reason Data", alarm.StateReasonData)
	}
	if alarm.StateUpdatedTimestamp != nil {
		d.Field("State Updated", alarm.StateUpdatedTimestamp.Format("2006-01-02 15:04:05 MST"))
	}
	if alarm.StateTransitionedTimestamp != nil {
		d.Field("State Transitioned", alarm.StateTransitionedTimestamp.Format("2006-01-02 15:04:05 MST"))
	}

	if alarm.IsMetricAlarm() {
		d.Section("Metric Configuration")
		if alarm.Namespace != "" {
			d.Field("Namespace", alarm.Namespace)
		}
		if alarm.MetricName != "" {
			d.Field("Metric Name", alarm.MetricName)
		}
		if dims := alarm.DimensionsStr(); dims != "" {
			d.Field("Dimensions", dims)
		}
		if alarm.Statistic != "" {
			d.Field("Statistic", alarm.Statistic)
		}
		if alarm.ExtendedStatistic != "" {
			d.Field("Extended Statistic", alarm.ExtendedStatistic)
		}
		if alarm.Period > 0 {
			d.Field("Period", fmt.Sprintf("%d seconds", alarm.Period))
		}
		if alarm.EvaluationPeriods > 0 {
			d.Field("Evaluation Periods", fmt.Sprintf("%d", alarm.EvaluationPeriods))
		}
		if alarm.DatapointsToAlarm > 0 {
			d.Field("Datapoints to Alarm", fmt.Sprintf("%d", alarm.DatapointsToAlarm))
		}
		if alarm.Threshold != nil {
			d.Field("Threshold", fmt.Sprintf("%.6f", *alarm.Threshold))
		}
		if alarm.ThresholdMetricId != "" {
			d.Field("Threshold Metric ID", alarm.ThresholdMetricId)
		}
		if alarm.ComparisonOperator != "" {
			d.Field("Comparison Operator", alarm.ComparisonOperator)
		}
		if alarm.TreatMissingData != "" {
			d.Field("Treat Missing Data", alarm.TreatMissingData)
		}
		if alarm.EvaluateLowSampleCountPercentile != "" {
			d.Field("Evaluate Low Sample Count", alarm.EvaluateLowSampleCountPercentile)
		}
		if alarm.Unit != "" {
			d.Field("Unit", alarm.Unit)
		}

		if len(alarm.Metrics) > 0 {
			d.Section("Metric Math")
			for i, m := range alarm.Metrics {
				prefix := fmt.Sprintf("[%d] ", i+1)
				if m.Id != nil {
					d.Field(prefix+"ID", *m.Id)
				}
				if m.Expression != nil {
					d.Field(prefix+"Expression", *m.Expression)
				}
				if m.Label != nil {
					d.Field(prefix+"Label", *m.Label)
				}
				if m.MetricStat != nil && m.MetricStat.Metric != nil {
					metric := m.MetricStat.Metric
					if metric.Namespace != nil {
						d.Field(prefix+"Namespace", *metric.Namespace)
					}
					if metric.MetricName != nil {
						d.Field(prefix+"Metric", *metric.MetricName)
					}
				}
				d.Field(prefix+"Return Data", fmt.Sprintf("%v", m.ReturnData == nil || *m.ReturnData))
			}
		}
	}

	if alarm.IsCompositeAlarm() {
		d.Section("Composite Alarm Rule")
		if alarm.AlarmRule != "" {
			d.Field("Alarm Rule", alarm.AlarmRule)
		}
		if alarm.ActionsSuppressor != "" {
			d.Field("Actions Suppressor", alarm.ActionsSuppressor)
		}
		if alarm.ActionsSuppressorExtensionPeriod > 0 {
			d.Field("Suppressor Extension Period", fmt.Sprintf("%d seconds", alarm.ActionsSuppressorExtensionPeriod))
		}
		if alarm.ActionsSuppressorWaitPeriod > 0 {
			d.Field("Suppressor Wait Period", fmt.Sprintf("%d seconds", alarm.ActionsSuppressorWaitPeriod))
		}
	}

	d.Section("Actions Configuration")
	d.Field("Actions Enabled", alarm.ActionsEnabledStr())
	if len(alarm.AlarmActions) > 0 {
		d.Field("Alarm Actions", strings.Join(alarm.AlarmActions, "\n"))
	} else {
		d.Field("Alarm Actions", render.Empty)
	}
	if len(alarm.OKActions) > 0 {
		d.Field("OK Actions", strings.Join(alarm.OKActions, "\n"))
	} else {
		d.Field("OK Actions", render.Empty)
	}
	if len(alarm.InsufficientDataActions) > 0 {
		d.Field("Insufficient Data Actions", strings.Join(alarm.InsufficientDataActions, "\n"))
	} else {
		d.Field("Insufficient Data Actions", render.Empty)
	}

	d.Section("Timestamps")
	if alarm.AlarmConfigurationUpdatedTimestamp != nil {
		d.Field("Configuration Updated", alarm.AlarmConfigurationUpdatedTimestamp.Format("2006-01-02 15:04:05 MST"))
	}

	d.Section("Full Details")
	if alarm.IsMetricAlarm() && alarm.MetricAlarmItem != nil {
		if jsonBytes, err := json.MarshalIndent(alarm.MetricAlarmItem, "", "  "); err == nil {
			d.Line(string(jsonBytes))
		}
	} else if alarm.IsCompositeAlarm() && alarm.CompositeAlarmItem != nil {
		if jsonBytes, err := json.MarshalIndent(alarm.CompositeAlarmItem, "", "  "); err == nil {
			d.Line(string(jsonBytes))
		}
	}

	return d.String()
}

func (r *AlarmRenderer) RenderSummary(resource dao.Resource) []render.SummaryField {
	alarm, ok := resource.(*AlarmResource)
	if !ok {
		return r.BaseRenderer.RenderSummary(resource)
	}

	fields := []render.SummaryField{
		{Label: "Name", Value: alarm.GetName()},
		{Label: "Type", Value: alarm.AlarmType},
		{Label: "State", Value: alarm.StateValue},
		{Label: "Actions", Value: alarm.ActionsEnabledStr()},
	}

	if alarm.IsMetricAlarm() {
		if alarm.Namespace != "" {
			fields = append(fields, render.SummaryField{Label: "Namespace", Value: alarm.Namespace})
		}
		if alarm.MetricName != "" {
			fields = append(fields, render.SummaryField{Label: "Metric", Value: alarm.MetricName})
		}
	}

	if alarm.StateUpdatedTimestamp != nil {
		fields = append(fields, render.SummaryField{Label: "Updated", Value: alarm.StateUpdatedStr()})
	}

	return fields
}

func (r *AlarmRenderer) Navigations(resource dao.Resource) []render.Navigation {
	alarm, ok := resource.(*AlarmResource)
	if !ok {
		return nil
	}

	var navs []render.Navigation

	if len(alarm.AlarmActions) > 0 && strings.Contains(alarm.AlarmActions[0], ":sns:") {
		navs = append(navs, render.Navigation{
			Key:         "t",
			Label:       "SNS Topic",
			Service:     "sns",
			Resource:    "topics",
			FilterField: "TopicArn",
			FilterValue: alarm.AlarmActions[0],
		})
	}

	return navs
}
