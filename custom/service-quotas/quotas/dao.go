package quotas

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/servicequotas"
	"github.com/aws/aws-sdk-go-v2/service/servicequotas/types"

	appaws "github.com/clawscli/claws/internal/aws"
	"github.com/clawscli/claws/internal/dao"
	apperrors "github.com/clawscli/claws/internal/errors"
)

// QuotaResource wraps a Service Quota
type QuotaResource struct {
	dao.BaseResource
	Item types.ServiceQuota
}

// GetID returns the quota code
func (r *QuotaResource) GetID() string {
	if r.Item.QuotaCode != nil {
		return *r.Item.QuotaCode
	}
	return ""
}

// GetARN returns the quota ARN
func (r *QuotaResource) GetARN() string {
	if r.Item.QuotaArn != nil {
		return *r.Item.QuotaArn
	}
	return ""
}

// GetName returns the quota name
func (r *QuotaResource) GetName() string {
	if r.Item.QuotaName != nil {
		return *r.Item.QuotaName
	}
	return ""
}

// QuotaCode returns the quota code
func (r *QuotaResource) QuotaCode() string {
	if r.Item.QuotaCode != nil {
		return *r.Item.QuotaCode
	}
	return ""
}

// QuotaName returns the quota name
func (r *QuotaResource) QuotaName() string {
	if r.Item.QuotaName != nil {
		return *r.Item.QuotaName
	}
	return ""
}

// ServiceCode returns the service code
func (r *QuotaResource) ServiceCode() string {
	if r.Item.ServiceCode != nil {
		return *r.Item.ServiceCode
	}
	return ""
}

// ServiceName returns the service name
func (r *QuotaResource) ServiceName() string {
	if r.Item.ServiceName != nil {
		return *r.Item.ServiceName
	}
	return ""
}

// Value returns the quota value
func (r *QuotaResource) Value() float64 {
	if r.Item.Value != nil {
		return *r.Item.Value
	}
	return 0
}

// Unit returns the quota unit
func (r *QuotaResource) Unit() string {
	if r.Item.Unit != nil {
		return *r.Item.Unit
	}
	return ""
}

// Adjustable returns whether the quota is adjustable
func (r *QuotaResource) Adjustable() bool {
	return r.Item.Adjustable
}

// GlobalQuota returns whether the quota is global
func (r *QuotaResource) GlobalQuota() bool {
	return r.Item.GlobalQuota
}

// Description returns the quota description
func (r *QuotaResource) Description() string {
	if r.Item.Description != nil {
		return *r.Item.Description
	}
	return ""
}

// NewQuotaResource creates a new QuotaResource
func NewQuotaResource(quota types.ServiceQuota) *QuotaResource {
	return &QuotaResource{
		BaseResource: dao.BaseResource{
			ID:   appaws.Str(quota.QuotaCode),
			Name: appaws.Str(quota.QuotaName),
			ARN:  appaws.Str(quota.QuotaArn),
			Data: quota,
		},
		Item: quota,
	}
}

// QuotaDAO handles Service Quotas quotas
type QuotaDAO struct {
	dao.BaseDAO
	client *servicequotas.Client
}

// NewQuotaDAO creates a new QuotaDAO
func NewQuotaDAO(ctx context.Context) (dao.DAO, error) {
	cfg, err := appaws.NewConfig(ctx)
	if err != nil {
		return nil, apperrors.Wrap(err, "new servicequotas/quotas dao")
	}
	return &QuotaDAO{
		BaseDAO: dao.NewBaseDAO("service-quotas", "quotas"),
		client:  servicequotas.NewFromConfig(cfg),
	}, nil
}

// List returns all quotas for a service
func (d *QuotaDAO) List(ctx context.Context) ([]dao.Resource, error) {
	serviceCode := dao.GetFilterFromContext(ctx, "ServiceCode")
	if serviceCode == "" {
		return nil, fmt.Errorf("ServiceCode filter required. Navigate from services (q key) or use :service-quotas/services")
	}

	var resources []dao.Resource
	paginator := servicequotas.NewListServiceQuotasPaginator(d.client, &servicequotas.ListServiceQuotasInput{
		ServiceCode: &serviceCode,
	})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, apperrors.Wrap(err, "list quotas")
		}

		for _, quota := range page.Quotas {
			resources = append(resources, NewQuotaResource(quota))
		}
	}

	return resources, nil
}

// Get returns a specific quota
func (d *QuotaDAO) Get(ctx context.Context, id string) (dao.Resource, error) {
	// We need the service code to get a quota
	serviceCode := dao.GetFilterFromContext(ctx, "ServiceCode")
	if serviceCode == "" {
		return nil, fmt.Errorf("service code required")
	}

	output, err := d.client.GetServiceQuota(ctx, &servicequotas.GetServiceQuotaInput{
		ServiceCode: &serviceCode,
		QuotaCode:   &id,
	})
	if err != nil {
		return nil, apperrors.Wrap(err, "get quota")
	}

	if output.Quota == nil {
		return nil, nil
	}

	return NewQuotaResource(*output.Quota), nil
}

// Delete is not supported for quotas
func (d *QuotaDAO) Delete(ctx context.Context, id string) error {
	return fmt.Errorf("delete not supported for quotas")
}

// Supports returns whether an operation is supported
func (d *QuotaDAO) Supports(op dao.Operation) bool {
	switch op {
	case dao.OpList, dao.OpGet:
		return true
	default:
		return false
	}
}
