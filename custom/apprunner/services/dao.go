package services

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/apprunner"
	"github.com/aws/aws-sdk-go-v2/service/apprunner/types"

	appaws "github.com/clawscli/claws/internal/aws"
	"github.com/clawscli/claws/internal/dao"
	apperrors "github.com/clawscli/claws/internal/errors"
)

// ServiceDAO provides data access for App Runner services.
type ServiceDAO struct {
	dao.BaseDAO
	client *apprunner.Client
}

// NewServiceDAO creates a new ServiceDAO.
func NewServiceDAO(ctx context.Context) (dao.DAO, error) {
	cfg, err := appaws.NewConfig(ctx)
	if err != nil {
		return nil, apperrors.Wrap(err, "new "+ServiceResourcePath+" dao")
	}
	return &ServiceDAO{
		BaseDAO: dao.NewBaseDAO("apprunner", "services"),
		client:  apprunner.NewFromConfig(cfg),
	}, nil
}

// List returns all App Runner services.
func (d *ServiceDAO) List(ctx context.Context) ([]dao.Resource, error) {
	services, err := appaws.Paginate(ctx, func(token *string) ([]types.ServiceSummary, *string, error) {
		output, err := d.client.ListServices(ctx, &apprunner.ListServicesInput{
			NextToken: token,
		})
		if err != nil {
			return nil, nil, apperrors.Wrap(err, "list app runner services")
		}
		return output.ServiceSummaryList, output.NextToken, nil
	})
	if err != nil {
		return nil, err
	}

	resources := make([]dao.Resource, len(services))
	for i, svc := range services {
		resources[i] = NewServiceResource(svc)
	}
	return resources, nil
}

// Get returns a specific App Runner service by ARN.
func (d *ServiceDAO) Get(ctx context.Context, id string) (dao.Resource, error) {
	output, err := d.client.DescribeService(ctx, &apprunner.DescribeServiceInput{
		ServiceArn: &id,
	})
	if err != nil {
		return nil, apperrors.Wrapf(err, "describe app runner service %s", id)
	}
	return NewServiceResourceFromDetail(*output.Service), nil
}

// Delete deletes an App Runner service by ARN.
func (d *ServiceDAO) Delete(ctx context.Context, id string) error {
	_, err := d.client.DeleteService(ctx, &apprunner.DeleteServiceInput{
		ServiceArn: &id,
	})
	if err != nil {
		return apperrors.Wrapf(err, "delete app runner service %s", id)
	}
	return nil
}

// ServiceResource wraps an App Runner service.
type ServiceResource struct {
	dao.BaseResource
	Summary *types.ServiceSummary
	Detail  *types.Service
}

// NewServiceResource creates a new ServiceResource from summary.
func NewServiceResource(svc types.ServiceSummary) *ServiceResource {
	return &ServiceResource{
		BaseResource: dao.BaseResource{
			ID:   appaws.Str(svc.ServiceName),
			ARN:  appaws.Str(svc.ServiceArn),
			Data: svc,
		},
		Summary: &svc,
	}
}

// NewServiceResourceFromDetail creates a new ServiceResource from detail.
func NewServiceResourceFromDetail(svc types.Service) *ServiceResource {
	return &ServiceResource{
		BaseResource: dao.BaseResource{
			ID:   appaws.Str(svc.ServiceName),
			ARN:  appaws.Str(svc.ServiceArn),
			Data: svc,
		},
		Detail: &svc,
	}
}

// ServiceName returns the service name.
func (r *ServiceResource) ServiceName() string {
	return r.ID
}

// Status returns the service status.
func (r *ServiceResource) Status() string {
	if r.Summary != nil {
		return string(r.Summary.Status)
	}
	if r.Detail != nil {
		return string(r.Detail.Status)
	}
	return ""
}

// ServiceUrl returns the service URL.
func (r *ServiceResource) ServiceUrl() string {
	if r.Summary != nil {
		return appaws.Str(r.Summary.ServiceUrl)
	}
	if r.Detail != nil {
		return appaws.Str(r.Detail.ServiceUrl)
	}
	return ""
}

// CreatedAt returns when the service was created.
func (r *ServiceResource) CreatedAt() *time.Time {
	if r.Summary != nil {
		return r.Summary.CreatedAt
	}
	if r.Detail != nil {
		return r.Detail.CreatedAt
	}
	return nil
}

// UpdatedAt returns when the service was updated.
func (r *ServiceResource) UpdatedAt() *time.Time {
	if r.Summary != nil {
		return r.Summary.UpdatedAt
	}
	if r.Detail != nil {
		return r.Detail.UpdatedAt
	}
	return nil
}

// ServiceId returns the service ID.
func (r *ServiceResource) ServiceId() string {
	if r.Detail != nil {
		return appaws.Str(r.Detail.ServiceId)
	}
	return ""
}

// AutoDeploymentsEnabled returns whether auto deployments are enabled.
func (r *ServiceResource) AutoDeploymentsEnabled() bool {
	if r.Detail != nil && r.Detail.SourceConfiguration != nil {
		return appaws.Bool(r.Detail.SourceConfiguration.AutoDeploymentsEnabled)
	}
	return false
}

// Cpu returns the CPU configuration.
func (r *ServiceResource) Cpu() string {
	if r.Detail != nil && r.Detail.InstanceConfiguration != nil {
		return appaws.Str(r.Detail.InstanceConfiguration.Cpu)
	}
	return ""
}

// Memory returns the memory configuration.
func (r *ServiceResource) Memory() string {
	if r.Detail != nil && r.Detail.InstanceConfiguration != nil {
		return appaws.Str(r.Detail.InstanceConfiguration.Memory)
	}
	return ""
}

// InstanceRoleArn returns the instance role ARN.
func (r *ServiceResource) InstanceRoleArn() string {
	if r.Detail != nil && r.Detail.InstanceConfiguration != nil {
		return appaws.Str(r.Detail.InstanceConfiguration.InstanceRoleArn)
	}
	return ""
}

// HealthCheckPath returns the health check path.
func (r *ServiceResource) HealthCheckPath() string {
	if r.Detail != nil && r.Detail.HealthCheckConfiguration != nil {
		return appaws.Str(r.Detail.HealthCheckConfiguration.Path)
	}
	return ""
}

// HealthCheckProtocol returns the health check protocol.
func (r *ServiceResource) HealthCheckProtocol() string {
	if r.Detail != nil && r.Detail.HealthCheckConfiguration != nil {
		return string(r.Detail.HealthCheckConfiguration.Protocol)
	}
	return ""
}

// HealthCheckInterval returns the health check interval in seconds.
func (r *ServiceResource) HealthCheckInterval() int32 {
	if r.Detail != nil && r.Detail.HealthCheckConfiguration != nil {
		return appaws.Int32(r.Detail.HealthCheckConfiguration.Interval)
	}
	return 0
}

// HealthCheckTimeout returns the health check timeout in seconds.
func (r *ServiceResource) HealthCheckTimeout() int32 {
	if r.Detail != nil && r.Detail.HealthCheckConfiguration != nil {
		return appaws.Int32(r.Detail.HealthCheckConfiguration.Timeout)
	}
	return 0
}

// SourceType returns the source type (CODE_REPOSITORY or IMAGE_REPOSITORY).
func (r *ServiceResource) SourceType() string {
	if r.Detail != nil && r.Detail.SourceConfiguration != nil {
		if r.Detail.SourceConfiguration.CodeRepository != nil {
			return "CODE_REPOSITORY"
		}
		if r.Detail.SourceConfiguration.ImageRepository != nil {
			return "IMAGE_REPOSITORY"
		}
	}
	return ""
}

// ImageIdentifier returns the container image identifier.
func (r *ServiceResource) ImageIdentifier() string {
	if r.Detail != nil && r.Detail.SourceConfiguration != nil && r.Detail.SourceConfiguration.ImageRepository != nil {
		return appaws.Str(r.Detail.SourceConfiguration.ImageRepository.ImageIdentifier)
	}
	return ""
}

// RepositoryUrl returns the code repository URL.
func (r *ServiceResource) RepositoryUrl() string {
	if r.Detail != nil && r.Detail.SourceConfiguration != nil && r.Detail.SourceConfiguration.CodeRepository != nil {
		return appaws.Str(r.Detail.SourceConfiguration.CodeRepository.RepositoryUrl)
	}
	return ""
}

// DeletedAt returns when the service was deleted.
func (r *ServiceResource) DeletedAt() *time.Time {
	if r.Detail != nil {
		return r.Detail.DeletedAt
	}
	return nil
}

// EncryptionKeyArn returns the encryption key ARN.
func (r *ServiceResource) EncryptionKeyArn() string {
	if r.Detail != nil && r.Detail.EncryptionConfiguration != nil {
		return appaws.Str(r.Detail.EncryptionConfiguration.KmsKey)
	}
	return ""
}

// ObservabilityEnabled returns whether observability is enabled.
func (r *ServiceResource) ObservabilityEnabled() bool {
	if r.Detail != nil && r.Detail.ObservabilityConfiguration != nil {
		return r.Detail.ObservabilityConfiguration.ObservabilityEnabled
	}
	return false
}

// NetworkEgressType returns the network egress type.
func (r *ServiceResource) NetworkEgressType() string {
	if r.Detail != nil && r.Detail.NetworkConfiguration != nil && r.Detail.NetworkConfiguration.EgressConfiguration != nil {
		return string(r.Detail.NetworkConfiguration.EgressConfiguration.EgressType)
	}
	return ""
}

// VpcConnectorArn returns the VPC connector ARN.
func (r *ServiceResource) VpcConnectorArn() string {
	if r.Detail != nil && r.Detail.NetworkConfiguration != nil && r.Detail.NetworkConfiguration.EgressConfiguration != nil {
		return appaws.Str(r.Detail.NetworkConfiguration.EgressConfiguration.VpcConnectorArn)
	}
	return ""
}

// IngressIsPublic returns whether the ingress is public.
func (r *ServiceResource) IngressIsPublic() bool {
	if r.Detail != nil && r.Detail.NetworkConfiguration != nil && r.Detail.NetworkConfiguration.IngressConfiguration != nil {
		return r.Detail.NetworkConfiguration.IngressConfiguration.IsPubliclyAccessible
	}
	return true // default to true for App Runner
}
