package matchmakingconfigs

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/gamelift"
	"github.com/aws/aws-sdk-go-v2/service/gamelift/types"

	appaws "github.com/clawscli/claws/internal/aws"
	"github.com/clawscli/claws/internal/dao"
	apperrors "github.com/clawscli/claws/internal/errors"
)

// MatchmakingConfigDAO provides data access for GameLift matchmaking configurations.
type MatchmakingConfigDAO struct {
	dao.BaseDAO
	client *gamelift.Client
}

// NewMatchmakingConfigDAO creates a new MatchmakingConfigDAO.
func NewMatchmakingConfigDAO(ctx context.Context) (dao.DAO, error) {
	cfg, err := appaws.NewConfig(ctx)
	if err != nil {
		return nil, apperrors.Wrap(err, "new "+ServiceResourcePath+" dao")
	}
	return &MatchmakingConfigDAO{
		BaseDAO: dao.NewBaseDAO("gamelift", "matchmaking-configs"),
		client:  gamelift.NewFromConfig(cfg),
	}, nil
}

// List returns all GameLift matchmaking configurations.
func (d *MatchmakingConfigDAO) List(ctx context.Context) ([]dao.Resource, error) {
	configs, err := appaws.Paginate(ctx, func(token *string) ([]types.MatchmakingConfiguration, *string, error) {
		output, err := d.client.DescribeMatchmakingConfigurations(ctx, &gamelift.DescribeMatchmakingConfigurationsInput{
			NextToken: token,
		})
		if err != nil {
			return nil, nil, apperrors.Wrap(err, "describe gamelift matchmaking configurations")
		}
		return output.Configurations, output.NextToken, nil
	})
	if err != nil {
		return nil, err
	}

	resources := make([]dao.Resource, len(configs))
	for i, config := range configs {
		resources[i] = NewMatchmakingConfigResource(config)
	}
	return resources, nil
}

// Get returns a specific GameLift matchmaking configuration by name.
func (d *MatchmakingConfigDAO) Get(ctx context.Context, id string) (dao.Resource, error) {
	output, err := d.client.DescribeMatchmakingConfigurations(ctx, &gamelift.DescribeMatchmakingConfigurationsInput{
		Names: []string{id},
	})
	if err != nil {
		return nil, apperrors.Wrapf(err, "describe gamelift matchmaking configuration %s", id)
	}
	if len(output.Configurations) == 0 {
		return nil, fmt.Errorf("gamelift matchmaking configuration %s not found", id)
	}
	return NewMatchmakingConfigResource(output.Configurations[0]), nil
}

// Delete deletes a GameLift matchmaking configuration by name.
func (d *MatchmakingConfigDAO) Delete(ctx context.Context, id string) error {
	_, err := d.client.DeleteMatchmakingConfiguration(ctx, &gamelift.DeleteMatchmakingConfigurationInput{
		Name: &id,
	})
	if err != nil {
		return apperrors.Wrapf(err, "delete gamelift matchmaking configuration %s", id)
	}
	return nil
}

// MatchmakingConfigResource wraps a GameLift matchmaking configuration.
type MatchmakingConfigResource struct {
	dao.BaseResource
	Config types.MatchmakingConfiguration
}

// NewMatchmakingConfigResource creates a new MatchmakingConfigResource.
func NewMatchmakingConfigResource(config types.MatchmakingConfiguration) *MatchmakingConfigResource {
	return &MatchmakingConfigResource{
		BaseResource: dao.BaseResource{
			ID:   appaws.Str(config.Name),
			Name: appaws.Str(config.Name),
			ARN:  appaws.Str(config.ConfigurationArn),
			Data: config,
		},
		Config: config,
	}
}

// RuleSetName returns the rule set name.
func (r *MatchmakingConfigResource) RuleSetName() string {
	return appaws.Str(r.Config.RuleSetName)
}

// RuleSetArn returns the rule set ARN.
func (r *MatchmakingConfigResource) RuleSetArn() string {
	return appaws.Str(r.Config.RuleSetArn)
}

// FlexMatchMode returns the FlexMatch mode.
func (r *MatchmakingConfigResource) FlexMatchMode() string {
	return string(r.Config.FlexMatchMode)
}

// BackfillMode returns the backfill mode.
func (r *MatchmakingConfigResource) BackfillMode() string {
	return string(r.Config.BackfillMode)
}

// AcceptanceRequired returns whether acceptance is required.
func (r *MatchmakingConfigResource) AcceptanceRequired() bool {
	return appaws.Bool(r.Config.AcceptanceRequired)
}

// AcceptanceTimeoutSeconds returns the acceptance timeout.
func (r *MatchmakingConfigResource) AcceptanceTimeoutSeconds() int32 {
	return appaws.Int32(r.Config.AcceptanceTimeoutSeconds)
}

// RequestTimeoutSeconds returns the request timeout.
func (r *MatchmakingConfigResource) RequestTimeoutSeconds() int32 {
	return appaws.Int32(r.Config.RequestTimeoutSeconds)
}

// AdditionalPlayerCount returns the additional player count.
func (r *MatchmakingConfigResource) AdditionalPlayerCount() int32 {
	return appaws.Int32(r.Config.AdditionalPlayerCount)
}

// Description returns the description.
func (r *MatchmakingConfigResource) Description() string {
	return appaws.Str(r.Config.Description)
}

// GameSessionQueueArns returns the game session queue ARNs.
func (r *MatchmakingConfigResource) GameSessionQueueArns() []string {
	return r.Config.GameSessionQueueArns
}

// NotificationTarget returns the SNS notification target.
func (r *MatchmakingConfigResource) NotificationTarget() string {
	return appaws.Str(r.Config.NotificationTarget)
}

// CreationTime returns when the configuration was created.
func (r *MatchmakingConfigResource) CreationTime() *time.Time {
	return r.Config.CreationTime
}

// GameProperties returns the game properties.
func (r *MatchmakingConfigResource) GameProperties() []types.GameProperty {
	return r.Config.GameProperties
}

// GameSessionData returns the game session data.
func (r *MatchmakingConfigResource) GameSessionData() string {
	return appaws.Str(r.Config.GameSessionData)
}

// CustomEventData returns the custom event data.
func (r *MatchmakingConfigResource) CustomEventData() string {
	return appaws.Str(r.Config.CustomEventData)
}
