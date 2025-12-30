package models

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/sagemaker"
	"github.com/aws/aws-sdk-go-v2/service/sagemaker/types"

	appaws "github.com/clawscli/claws/internal/aws"
	"github.com/clawscli/claws/internal/dao"
)

// ModelDAO provides data access for SageMaker models.
type ModelDAO struct {
	dao.BaseDAO
	client *sagemaker.Client
}

// NewModelDAO creates a new ModelDAO.
func NewModelDAO(ctx context.Context) (dao.DAO, error) {
	cfg, err := appaws.NewConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("new sagemaker/models dao: %w", err)
	}
	return &ModelDAO{
		BaseDAO: dao.NewBaseDAO("sagemaker", "models"),
		client:  sagemaker.NewFromConfig(cfg),
	}, nil
}

// List returns all SageMaker models.
func (d *ModelDAO) List(ctx context.Context) ([]dao.Resource, error) {
	models, err := appaws.Paginate(ctx, func(token *string) ([]types.ModelSummary, *string, error) {
		output, err := d.client.ListModels(ctx, &sagemaker.ListModelsInput{
			NextToken: token,
		})
		if err != nil {
			return nil, nil, fmt.Errorf("list sagemaker models: %w", err)
		}
		return output.Models, output.NextToken, nil
	})
	if err != nil {
		return nil, err
	}

	resources := make([]dao.Resource, len(models))
	for i, model := range models {
		resources[i] = NewModelResource(model)
	}
	return resources, nil
}

// Get returns a specific model.
func (d *ModelDAO) Get(ctx context.Context, id string) (dao.Resource, error) {
	output, err := d.client.DescribeModel(ctx, &sagemaker.DescribeModelInput{
		ModelName: &id,
	})
	if err != nil {
		return nil, fmt.Errorf("describe sagemaker model: %w", err)
	}
	// Convert to summary for consistent resource type
	summary := types.ModelSummary{
		ModelName:    output.ModelName,
		ModelArn:     output.ModelArn,
		CreationTime: output.CreationTime,
	}
	r := NewModelResource(summary)
	r.ExecutionRoleArn = appaws.Str(output.ExecutionRoleArn)
	if output.EnableNetworkIsolation != nil {
		r.EnableNetworkIsolation = *output.EnableNetworkIsolation
	}
	if output.PrimaryContainer != nil {
		r.PrimaryContainerImage = appaws.Str(output.PrimaryContainer.Image)
		r.PrimaryContainerModel = appaws.Str(output.PrimaryContainer.ModelDataUrl)
		r.PrimaryContainerMode = string(output.PrimaryContainer.Mode)
	}
	r.ContainerCount = len(output.Containers)
	r.VpcConfig = output.VpcConfig
	return r, nil
}

// Delete deletes a model.
func (d *ModelDAO) Delete(ctx context.Context, id string) error {
	_, err := d.client.DeleteModel(ctx, &sagemaker.DeleteModelInput{
		ModelName: &id,
	})
	if err != nil {
		return fmt.Errorf("delete sagemaker model: %w", err)
	}
	return nil
}

// ModelResource wraps a SageMaker model.
type ModelResource struct {
	dao.BaseResource
	Model                  types.ModelSummary
	ExecutionRoleArn       string
	EnableNetworkIsolation bool
	PrimaryContainerImage  string
	PrimaryContainerModel  string
	PrimaryContainerMode   string
	ContainerCount         int
	VpcConfig              *types.VpcConfig
}

// NewModelResource creates a new ModelResource.
func NewModelResource(model types.ModelSummary) *ModelResource {
	return &ModelResource{
		BaseResource: dao.BaseResource{
			ID:  appaws.Str(model.ModelName),
			ARN: appaws.Str(model.ModelArn),
		},
		Model: model,
	}
}

// CreatedAt returns when the model was created.
func (r *ModelResource) CreatedAt() *time.Time {
	return r.Model.CreationTime
}

// GetExecutionRoleArn returns the execution role ARN.
func (r *ModelResource) GetExecutionRoleArn() string {
	return r.ExecutionRoleArn
}

// GetEnableNetworkIsolation returns network isolation setting.
func (r *ModelResource) GetEnableNetworkIsolation() bool {
	return r.EnableNetworkIsolation
}

// GetPrimaryContainerImage returns the primary container image.
func (r *ModelResource) GetPrimaryContainerImage() string {
	return r.PrimaryContainerImage
}

// GetPrimaryContainerModel returns the primary container model data.
func (r *ModelResource) GetPrimaryContainerModel() string {
	return r.PrimaryContainerModel
}

// GetPrimaryContainerMode returns the primary container mode.
func (r *ModelResource) GetPrimaryContainerMode() string {
	return r.PrimaryContainerMode
}

// GetContainerCount returns the number of containers.
func (r *ModelResource) GetContainerCount() int {
	return r.ContainerCount
}

// GetVpcConfig returns the VPC configuration.
func (r *ModelResource) GetVpcConfig() *types.VpcConfig {
	return r.VpcConfig
}
