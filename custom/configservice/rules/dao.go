package rules

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/configservice"
	"github.com/aws/aws-sdk-go-v2/service/configservice/types"

	appaws "github.com/clawscli/claws/internal/aws"
	"github.com/clawscli/claws/internal/dao"
	apperrors "github.com/clawscli/claws/internal/errors"
)

// RuleDAO provides data access for AWS Config rules.
type RuleDAO struct {
	dao.BaseDAO
	client *configservice.Client
}

// NewRuleDAO creates a new RuleDAO.
func NewRuleDAO(ctx context.Context) (dao.DAO, error) {
	cfg, err := appaws.NewConfig(ctx)
	if err != nil {
		return nil, apperrors.Wrap(err, "new configservice/rules dao")
	}
	return &RuleDAO{
		BaseDAO: dao.NewBaseDAO("configservice", "rules"),
		client:  configservice.NewFromConfig(cfg),
	}, nil
}

// List returns rules (first page only for backwards compatibility).
// For paginated access, use ListPage instead.
func (d *RuleDAO) List(ctx context.Context) ([]dao.Resource, error) {
	resources, _, err := d.ListPage(ctx, 25, "")
	return resources, err
}

// ListPage returns a page of Config rules.
// Implements dao.PaginatedDAO interface.
func (d *RuleDAO) ListPage(ctx context.Context, pageSize int, pageToken string) ([]dao.Resource, string, error) {
	// DescribeConfigRules doesn't have MaxItems parameter,
	// but we still implement ListPage for consistent interface
	input := &configservice.DescribeConfigRulesInput{}
	if pageToken != "" {
		input.NextToken = &pageToken
	}

	output, err := d.client.DescribeConfigRules(ctx, input)
	if err != nil {
		return nil, "", apperrors.Wrap(err, "describe config rules")
	}

	resources := make([]dao.Resource, len(output.ConfigRules))
	for i, rule := range output.ConfigRules {
		resources[i] = NewRuleResource(rule)
	}

	nextToken := ""
	if output.NextToken != nil {
		nextToken = *output.NextToken
	}

	return resources, nextToken, nil
}

// Get returns a specific Config rule by name.
func (d *RuleDAO) Get(ctx context.Context, id string) (dao.Resource, error) {
	output, err := d.client.DescribeConfigRules(ctx, &configservice.DescribeConfigRulesInput{
		ConfigRuleNames: []string{id},
	})
	if err != nil {
		return nil, apperrors.Wrapf(err, "describe config rule %s", id)
	}
	if len(output.ConfigRules) == 0 {
		return nil, fmt.Errorf("config rule not found: %s", id)
	}
	return NewRuleResource(output.ConfigRules[0]), nil
}

// Delete deletes a Config rule by name.
func (d *RuleDAO) Delete(ctx context.Context, id string) error {
	_, err := d.client.DeleteConfigRule(ctx, &configservice.DeleteConfigRuleInput{
		ConfigRuleName: &id,
	})
	if err != nil {
		return apperrors.Wrapf(err, "delete config rule %s", id)
	}
	return nil
}

// RuleResource wraps an AWS Config rule.
type RuleResource struct {
	dao.BaseResource
	Item types.ConfigRule
}

// NewRuleResource creates a new RuleResource.
func NewRuleResource(rule types.ConfigRule) *RuleResource {
	return &RuleResource{
		BaseResource: dao.BaseResource{
			ID:  appaws.Str(rule.ConfigRuleName),
			ARN: appaws.Str(rule.ConfigRuleArn),
		},
		Item: rule,
	}
}

// Name returns the rule name.
func (r *RuleResource) Name() string {
	return appaws.Str(r.Item.ConfigRuleName)
}

// Description returns the rule description.
func (r *RuleResource) Description() string {
	return appaws.Str(r.Item.Description)
}

// State returns the rule state.
func (r *RuleResource) State() string {
	return string(r.Item.ConfigRuleState)
}

// Source returns the rule source (AWS managed or custom).
func (r *RuleResource) Source() types.Source {
	return *r.Item.Source
}

// SourceIdentifier returns the source identifier.
func (r *RuleResource) SourceIdentifier() string {
	if r.Item.Source != nil {
		return appaws.Str(r.Item.Source.SourceIdentifier)
	}
	return ""
}

// SourceOwner returns the source owner (AWS or CUSTOM_LAMBDA).
func (r *RuleResource) SourceOwner() string {
	if r.Item.Source != nil {
		return string(r.Item.Source.Owner)
	}
	return ""
}

// MaximumExecutionFrequency returns the rule execution frequency.
func (r *RuleResource) MaximumExecutionFrequency() string {
	return string(r.Item.MaximumExecutionFrequency)
}

// InputParameters returns the rule input parameters.
func (r *RuleResource) InputParameters() string {
	return appaws.Str(r.Item.InputParameters)
}

// CreatedBy returns who created the rule.
func (r *RuleResource) CreatedBy() string {
	return appaws.Str(r.Item.CreatedBy)
}
