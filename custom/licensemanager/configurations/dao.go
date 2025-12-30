package configurations

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/licensemanager"
	"github.com/aws/aws-sdk-go-v2/service/licensemanager/types"

	appaws "github.com/clawscli/claws/internal/aws"
	"github.com/clawscli/claws/internal/dao"
	apperrors "github.com/clawscli/claws/internal/errors"
)

// ConfigurationDAO provides data access for License Manager configurations.
type ConfigurationDAO struct {
	dao.BaseDAO
	client *licensemanager.Client
}

// NewConfigurationDAO creates a new ConfigurationDAO.
func NewConfigurationDAO(ctx context.Context) (dao.DAO, error) {
	cfg, err := appaws.NewConfig(ctx)
	if err != nil {
		return nil, apperrors.Wrap(err, "new licensemanager/configurations dao")
	}
	return &ConfigurationDAO{
		BaseDAO: dao.NewBaseDAO("license-manager", "configurations"),
		client:  licensemanager.NewFromConfig(cfg),
	}, nil
}

// List returns all license configurations.
func (d *ConfigurationDAO) List(ctx context.Context) ([]dao.Resource, error) {
	configs, err := appaws.Paginate(ctx, func(token *string) ([]types.LicenseConfiguration, *string, error) {
		output, err := d.client.ListLicenseConfigurations(ctx, &licensemanager.ListLicenseConfigurationsInput{
			NextToken: token,
		})
		if err != nil {
			return nil, nil, apperrors.Wrap(err, "list license configurations")
		}
		return output.LicenseConfigurations, output.NextToken, nil
	})
	if err != nil {
		return nil, err
	}

	resources := make([]dao.Resource, len(configs))
	for i, config := range configs {
		resources[i] = NewConfigurationResource(config)
	}
	return resources, nil
}

// Get returns a specific license configuration.
func (d *ConfigurationDAO) Get(ctx context.Context, arn string) (dao.Resource, error) {
	output, err := d.client.GetLicenseConfiguration(ctx, &licensemanager.GetLicenseConfigurationInput{
		LicenseConfigurationArn: &arn,
	})
	if err != nil {
		return nil, apperrors.Wrap(err, "get license configuration")
	}

	return &ConfigurationResource{
		BaseResource: dao.BaseResource{
			ID:  appaws.Str(output.Name),
			ARN: appaws.Str(output.LicenseConfigurationArn),
		},
		Config: &types.LicenseConfiguration{
			LicenseConfigurationArn: output.LicenseConfigurationArn,
			Name:                    output.Name,
			Description:             output.Description,
			LicenseCountingType:     output.LicenseCountingType,
			LicenseCount:            output.LicenseCount,
			ConsumedLicenses:        output.ConsumedLicenses,
			Status:                  output.Status,
		},
	}, nil
}

// Delete deletes a license configuration.
func (d *ConfigurationDAO) Delete(ctx context.Context, arn string) error {
	_, err := d.client.DeleteLicenseConfiguration(ctx, &licensemanager.DeleteLicenseConfigurationInput{
		LicenseConfigurationArn: &arn,
	})
	if err != nil {
		return apperrors.Wrap(err, "delete license configuration")
	}
	return nil
}

// ConfigurationResource wraps a License Manager configuration.
type ConfigurationResource struct {
	dao.BaseResource
	Config *types.LicenseConfiguration
}

// NewConfigurationResource creates a new ConfigurationResource.
func NewConfigurationResource(config types.LicenseConfiguration) *ConfigurationResource {
	return &ConfigurationResource{
		BaseResource: dao.BaseResource{
			ID:  appaws.Str(config.Name),
			ARN: appaws.Str(config.LicenseConfigurationArn),
		},
		Config: &config,
	}
}

// Name returns the configuration name.
func (r *ConfigurationResource) Name() string {
	if r.Config != nil && r.Config.Name != nil {
		return *r.Config.Name
	}
	return ""
}

// Description returns the configuration description.
func (r *ConfigurationResource) Description() string {
	if r.Config != nil && r.Config.Description != nil {
		return *r.Config.Description
	}
	return ""
}

// LicenseCountingType returns the license counting type.
func (r *ConfigurationResource) LicenseCountingType() string {
	if r.Config != nil {
		return string(r.Config.LicenseCountingType)
	}
	return ""
}

// LicenseCount returns the license count.
func (r *ConfigurationResource) LicenseCount() int64 {
	if r.Config != nil && r.Config.LicenseCount != nil {
		return *r.Config.LicenseCount
	}
	return 0
}

// ConsumedLicenses returns the consumed license count.
func (r *ConfigurationResource) ConsumedLicenses() int64 {
	if r.Config != nil && r.Config.ConsumedLicenses != nil {
		return *r.Config.ConsumedLicenses
	}
	return 0
}

// Status returns the configuration status.
func (r *ConfigurationResource) Status() string {
	if r.Config != nil && r.Config.Status != nil {
		return *r.Config.Status
	}
	return ""
}
