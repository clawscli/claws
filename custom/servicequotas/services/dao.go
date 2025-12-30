package services

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/servicequotas"
	"github.com/aws/aws-sdk-go-v2/service/servicequotas/types"

	appaws "github.com/clawscli/claws/internal/aws"
	"github.com/clawscli/claws/internal/dao"
	apperrors "github.com/clawscli/claws/internal/errors"
)

// ServiceResource wraps a Service Quotas service
type ServiceResource struct {
	dao.BaseResource
	Item types.ServiceInfo
}

// GetID returns the service code
func (r *ServiceResource) GetID() string {
	if r.Item.ServiceCode != nil {
		return *r.Item.ServiceCode
	}
	return ""
}

// GetARN returns empty (services don't have ARNs)
func (r *ServiceResource) GetARN() string {
	return ""
}

// GetName returns the service name
func (r *ServiceResource) GetName() string {
	if r.Item.ServiceName != nil {
		return *r.Item.ServiceName
	}
	return ""
}

// ServiceCode returns the service code
func (r *ServiceResource) ServiceCode() string {
	if r.Item.ServiceCode != nil {
		return *r.Item.ServiceCode
	}
	return ""
}

// ServiceName returns the service name
func (r *ServiceResource) ServiceName() string {
	if r.Item.ServiceName != nil {
		return *r.Item.ServiceName
	}
	return ""
}

// NewServiceResource creates a new ServiceResource
func NewServiceResource(svc types.ServiceInfo) *ServiceResource {
	return &ServiceResource{
		BaseResource: dao.BaseResource{
			ID:   appaws.Str(svc.ServiceCode),
			Name: appaws.Str(svc.ServiceName),
			Data: svc,
		},
		Item: svc,
	}
}

// ServiceDAO handles Service Quotas services
type ServiceDAO struct {
	dao.BaseDAO
	client *servicequotas.Client
}

// NewServiceDAO creates a new ServiceDAO
func NewServiceDAO(ctx context.Context) (dao.DAO, error) {
	cfg, err := appaws.NewConfig(ctx)
	if err != nil {
		return nil, apperrors.Wrap(err, "new servicequotas/services dao")
	}
	return &ServiceDAO{
		BaseDAO: dao.NewBaseDAO("service-quotas", "services"),
		client:  servicequotas.NewFromConfig(cfg),
	}, nil
}

// List returns all services with quotas
func (d *ServiceDAO) List(ctx context.Context) ([]dao.Resource, error) {
	var resources []dao.Resource
	paginator := servicequotas.NewListServicesPaginator(d.client, &servicequotas.ListServicesInput{})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, apperrors.Wrap(err, "list services")
		}

		for _, svc := range page.Services {
			resources = append(resources, NewServiceResource(svc))
		}
	}

	return resources, nil
}

// Get returns a specific service (not really needed, but implement for interface)
func (d *ServiceDAO) Get(ctx context.Context, id string) (dao.Resource, error) {
	// Service Quotas doesn't have a GetService API, so we list and filter
	resources, err := d.List(ctx)
	if err != nil {
		return nil, err
	}

	for _, r := range resources {
		if r.GetID() == id {
			return r, nil
		}
	}

	return nil, nil
}

// Delete is not supported for services
func (d *ServiceDAO) Delete(ctx context.Context, id string) error {
	return fmt.Errorf("delete not supported for services")
}

// Supports returns whether an operation is supported
func (d *ServiceDAO) Supports(op dao.Operation) bool {
	switch op {
	case dao.OpList, dao.OpGet:
		return true
	default:
		return false
	}
}
