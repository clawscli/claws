package rules

import (
	"context"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/eventbridge"
	"github.com/aws/aws-sdk-go-v2/service/eventbridge/types"

	appaws "github.com/clawscli/claws/internal/aws"
	"github.com/clawscli/claws/internal/dao"
	apperrors "github.com/clawscli/claws/internal/errors"
	"github.com/clawscli/claws/internal/log"
)

// RuleDAO provides data access for EventBridge rules
type RuleDAO struct {
	dao.BaseDAO
	client *eventbridge.Client
}

// NewRuleDAO creates a new RuleDAO
func NewRuleDAO(ctx context.Context) (dao.DAO, error) {
	cfg, err := appaws.NewConfig(ctx)
	if err != nil {
		return nil, apperrors.Wrap(err, "new "+ServiceResourcePath+" dao")
	}
	return &RuleDAO{
		BaseDAO: dao.NewBaseDAO("events", "rules"),
		client:  eventbridge.NewFromConfig(cfg),
	}, nil
}

func (d *RuleDAO) List(ctx context.Context) ([]dao.Resource, error) {
	// First get all event buses
	busInput := &eventbridge.ListEventBusesInput{}
	busOutput, err := d.client.ListEventBuses(ctx, busInput)
	if err != nil {
		return nil, apperrors.Wrap(err, "list event buses")
	}

	var resources []dao.Resource

	// List rules for each event bus
	for _, bus := range busOutput.EventBuses {
		input := &eventbridge.ListRulesInput{
			EventBusName: bus.Name,
		}

		output, err := d.client.ListRules(ctx, input)
		if err != nil {
			log.Warn("failed to list rules for event bus", "eventBus", appaws.Str(bus.Name), "error", err)
			continue
		}

		for _, rule := range output.Rules {
			resources = append(resources, NewRuleResource(rule))
		}
	}

	return resources, nil
}

func (d *RuleDAO) Get(ctx context.Context, id string) (dao.Resource, error) {
	ruleName, eventBusName := parseRuleID(id)
	input := &eventbridge.DescribeRuleInput{
		Name: &ruleName,
	}
	if eventBusName != "" {
		input.EventBusName = &eventBusName
	}

	output, err := d.client.DescribeRule(ctx, input)
	if err != nil {
		return nil, apperrors.Wrapf(err, "describe rule %s", id)
	}

	// Convert DescribeRuleOutput to types.Rule
	rule := types.Rule{
		Name:               output.Name,
		Arn:                output.Arn,
		State:              output.State,
		Description:        output.Description,
		ScheduleExpression: output.ScheduleExpression,
		EventPattern:       output.EventPattern,
		EventBusName:       output.EventBusName,
	}

	res := NewRuleResource(rule)

	// Store role ARN
	if output.RoleArn != nil {
		res.RoleArn = *output.RoleArn
	}

	// Fetch targets
	targetsInput := &eventbridge.ListTargetsByRuleInput{
		Rule:         &ruleName,
		EventBusName: output.EventBusName,
	}
	if targetsOutput, err := d.client.ListTargetsByRule(ctx, targetsInput); err == nil {
		res.Targets = targetsOutput.Targets
	}

	return res, nil
}

func (d *RuleDAO) Delete(ctx context.Context, id string) error {
	ruleName, eventBusName := parseRuleID(id)
	// First, need to remove all targets
	targetsInput := &eventbridge.ListTargetsByRuleInput{
		Rule: &ruleName,
	}
	if eventBusName != "" {
		targetsInput.EventBusName = &eventBusName
	}
	targetsOutput, err := d.client.ListTargetsByRule(ctx, targetsInput)
	if err == nil && len(targetsOutput.Targets) > 0 {
		var targetIds []string
		for _, target := range targetsOutput.Targets {
			if target.Id != nil {
				targetIds = append(targetIds, *target.Id)
			}
		}
		if len(targetIds) > 0 {
			removeInput := &eventbridge.RemoveTargetsInput{
				Rule: &ruleName,
				Ids:  targetIds,
			}
			if eventBusName != "" {
				removeInput.EventBusName = &eventBusName
			}
			_, err = d.client.RemoveTargets(ctx, removeInput)
			if err != nil {
				return apperrors.Wrapf(err, "remove targets for rule %s", id)
			}
		}
	}

	input := &eventbridge.DeleteRuleInput{
		Name: &ruleName,
	}
	if eventBusName != "" {
		input.EventBusName = &eventBusName
	}

	_, err = d.client.DeleteRule(ctx, input)
	if err != nil {
		return apperrors.Wrapf(err, "delete rule %s", id)
	}

	return nil
}

// RuleResource wraps an EventBridge rule
type RuleResource struct {
	dao.BaseResource
	Item    types.Rule
	Targets []types.Target
	RoleArn string
}

// NewRuleResource creates a new RuleResource
func NewRuleResource(rule types.Rule) *RuleResource {
	name := appaws.Str(rule.Name)
	eventBusName := appaws.Str(rule.EventBusName)
	id := ruleID(name, eventBusName)

	return &RuleResource{
		BaseResource: dao.BaseResource{
			ID:   id,
			Name: name,
			ARN:  appaws.Str(rule.Arn),
			Tags: nil,
			Data: rule,
		},
		Item: rule,
	}
}

func ruleID(name, eventBusName string) string {
	if eventBusName == "" {
		return name
	}
	return eventBusName + "/" + name
}

func parseRuleID(id string) (name, eventBusName string) {
	idx := strings.LastIndex(id, "/")
	if idx < 0 {
		return id, ""
	}
	return id[idx+1:], id[:idx]
}

// ARN returns the rule ARN
func (r *RuleResource) ARN() string {
	if r.Item.Arn != nil {
		return *r.Item.Arn
	}
	return ""
}

// State returns the rule state
func (r *RuleResource) State() string {
	return string(r.Item.State)
}

// Description returns the rule description
func (r *RuleResource) Description() string {
	if r.Item.Description != nil {
		return *r.Item.Description
	}
	return ""
}

// ScheduleExpression returns the schedule expression (for scheduled rules)
func (r *RuleResource) ScheduleExpression() string {
	if r.Item.ScheduleExpression != nil {
		return *r.Item.ScheduleExpression
	}
	return ""
}

// EventPattern returns the event pattern (for event-triggered rules)
func (r *RuleResource) EventPattern() string {
	if r.Item.EventPattern != nil {
		return *r.Item.EventPattern
	}
	return ""
}

// EventBusName returns the event bus name
func (r *RuleResource) EventBusName() string {
	if r.Item.EventBusName != nil {
		return *r.Item.EventBusName
	}
	return "default"
}

// RuleType returns the type of rule (Schedule or Event)
func (r *RuleResource) RuleType() string {
	if r.ScheduleExpression() != "" {
		return "Schedule"
	}
	return "Event"
}
