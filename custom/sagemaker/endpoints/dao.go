package endpoints

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/sagemaker"
	"github.com/aws/aws-sdk-go-v2/service/sagemaker/types"

	appaws "github.com/clawscli/claws/internal/aws"
	"github.com/clawscli/claws/internal/dao"
	apperrors "github.com/clawscli/claws/internal/errors"
)

// EndpointDAO provides data access for SageMaker endpoints.
type EndpointDAO struct {
	dao.BaseDAO
	client *sagemaker.Client
}

// NewEndpointDAO creates a new EndpointDAO.
func NewEndpointDAO(ctx context.Context) (dao.DAO, error) {
	cfg, err := appaws.NewConfig(ctx)
	if err != nil {
		return nil, apperrors.Wrap(err, "new "+ServiceResourcePath+" dao")
	}
	return &EndpointDAO{
		BaseDAO: dao.NewBaseDAO("sagemaker", "endpoints"),
		client:  sagemaker.NewFromConfig(cfg),
	}, nil
}

// List returns all SageMaker endpoints.
func (d *EndpointDAO) List(ctx context.Context) ([]dao.Resource, error) {
	endpoints, err := appaws.Paginate(ctx, func(token *string) ([]types.EndpointSummary, *string, error) {
		output, err := d.client.ListEndpoints(ctx, &sagemaker.ListEndpointsInput{
			NextToken: token,
		})
		if err != nil {
			return nil, nil, apperrors.Wrap(err, "list sagemaker endpoints")
		}
		return output.Endpoints, output.NextToken, nil
	})
	if err != nil {
		return nil, err
	}

	resources := make([]dao.Resource, len(endpoints))
	for i, endpoint := range endpoints {
		resources[i] = NewEndpointResource(endpoint)
	}
	return resources, nil
}

// Get returns a specific endpoint.
func (d *EndpointDAO) Get(ctx context.Context, id string) (dao.Resource, error) {
	output, err := d.client.DescribeEndpoint(ctx, &sagemaker.DescribeEndpointInput{
		EndpointName: &id,
	})
	if err != nil {
		return nil, apperrors.Wrap(err, "describe sagemaker endpoint")
	}
	// Convert to summary for consistent resource type
	summary := types.EndpointSummary{
		EndpointName:     output.EndpointName,
		EndpointArn:      output.EndpointArn,
		EndpointStatus:   output.EndpointStatus,
		CreationTime:     output.CreationTime,
		LastModifiedTime: output.LastModifiedTime,
	}
	r := NewEndpointResource(summary)
	r.EndpointConfigName = appaws.Str(output.EndpointConfigName)
	r.FailureReason = appaws.Str(output.FailureReason)
	r.ProductionVariants = output.ProductionVariants
	r.DataCaptureConfig = output.DataCaptureConfig
	return r, nil
}

// Delete deletes an endpoint.
func (d *EndpointDAO) Delete(ctx context.Context, id string) error {
	_, err := d.client.DeleteEndpoint(ctx, &sagemaker.DeleteEndpointInput{
		EndpointName: &id,
	})
	if err != nil {
		return apperrors.Wrap(err, "delete sagemaker endpoint")
	}
	return nil
}

// EndpointResource wraps a SageMaker endpoint.
type EndpointResource struct {
	dao.BaseResource
	Endpoint           types.EndpointSummary
	EndpointConfigName string
	FailureReason      string
	ProductionVariants []types.ProductionVariantSummary
	DataCaptureConfig  *types.DataCaptureConfigSummary
}

// NewEndpointResource creates a new EndpointResource.
func NewEndpointResource(endpoint types.EndpointSummary) *EndpointResource {
	return &EndpointResource{
		BaseResource: dao.BaseResource{
			ID:   appaws.Str(endpoint.EndpointName),
			ARN:  appaws.Str(endpoint.EndpointArn),
			Data: endpoint,
		},
		Endpoint: endpoint,
	}
}

// Status returns the endpoint status.
func (r *EndpointResource) Status() string {
	return string(r.Endpoint.EndpointStatus)
}

// CreatedAt returns when the endpoint was created.
func (r *EndpointResource) CreatedAt() *time.Time {
	return r.Endpoint.CreationTime
}

// LastModifiedAt returns when the endpoint was last modified.
func (r *EndpointResource) LastModifiedAt() *time.Time {
	return r.Endpoint.LastModifiedTime
}

// GetEndpointConfigName returns the endpoint configuration name.
func (r *EndpointResource) GetEndpointConfigName() string {
	return r.EndpointConfigName
}

// GetFailureReason returns the failure reason.
func (r *EndpointResource) GetFailureReason() string {
	return r.FailureReason
}

// GetProductionVariants returns the production variants.
func (r *EndpointResource) GetProductionVariants() []types.ProductionVariantSummary {
	return r.ProductionVariants
}

// GetDataCaptureConfig returns the data capture configuration.
func (r *EndpointResource) GetDataCaptureConfig() *types.DataCaptureConfigSummary {
	return r.DataCaptureConfig
}
