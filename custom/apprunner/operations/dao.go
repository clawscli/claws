package operations

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/apprunner"
	"github.com/aws/aws-sdk-go-v2/service/apprunner/types"

	appaws "github.com/clawscli/claws/internal/aws"
	"github.com/clawscli/claws/internal/dao"
	apperrors "github.com/clawscli/claws/internal/errors"
)

// OperationDAO provides data access for App Runner operations.
type OperationDAO struct {
	dao.BaseDAO
	client *apprunner.Client
}

// NewOperationDAO creates a new OperationDAO.
func NewOperationDAO(ctx context.Context) (dao.DAO, error) {
	cfg, err := appaws.NewConfig(ctx)
	if err != nil {
		return nil, apperrors.Wrap(err, "new "+ServiceResourcePath+" dao")
	}
	return &OperationDAO{
		BaseDAO: dao.NewBaseDAO("apprunner", "operations"),
		client:  apprunner.NewFromConfig(cfg),
	}, nil
}

// List returns operations (first page only for backwards compatibility).
// For paginated access, use ListPage instead.
func (d *OperationDAO) List(ctx context.Context) ([]dao.Resource, error) {
	resources, _, err := d.ListPage(ctx, 20, "")
	return resources, err
}

// ListPage returns a page of App Runner operations.
// Implements dao.PaginatedDAO interface.
func (d *OperationDAO) ListPage(ctx context.Context, pageSize int, pageToken string) ([]dao.Resource, string, error) {
	serviceArn := dao.GetFilterFromContext(ctx, "ServiceArn")
	if serviceArn == "" {
		return nil, "", fmt.Errorf("service ARN filter required")
	}

	maxResults := int32(pageSize)
	if maxResults > 20 {
		maxResults = 20 // AWS API max
	}

	input := &apprunner.ListOperationsInput{
		ServiceArn: &serviceArn,
		MaxResults: &maxResults,
	}
	if pageToken != "" {
		input.NextToken = &pageToken
	}

	output, err := d.client.ListOperations(ctx, input)
	if err != nil {
		return nil, "", apperrors.Wrap(err, "list app runner operations")
	}

	resources := make([]dao.Resource, len(output.OperationSummaryList))
	for i, op := range output.OperationSummaryList {
		resources[i] = NewOperationResource(op)
	}

	nextToken := ""
	if output.NextToken != nil {
		nextToken = *output.NextToken
	}

	return resources, nextToken, nil
}

// Get returns a specific operation by ID.
func (d *OperationDAO) Get(ctx context.Context, id string) (dao.Resource, error) {
	// App Runner doesn't have a GetOperation API, so we list and find
	serviceArn := dao.GetFilterFromContext(ctx, "ServiceArn")
	if serviceArn == "" {
		return nil, fmt.Errorf("service ARN filter required")
	}

	resources, err := d.List(ctx)
	if err != nil {
		return nil, err
	}

	for _, r := range resources {
		if r.GetID() == id {
			return r, nil
		}
	}
	return nil, fmt.Errorf("operation not found: %s", id)
}

// Delete is not supported for operations.
func (d *OperationDAO) Delete(ctx context.Context, id string) error {
	return fmt.Errorf("delete not supported for operations")
}

// OperationResource wraps an App Runner operation.
type OperationResource struct {
	dao.BaseResource
	Item types.OperationSummary
}

// NewOperationResource creates a new OperationResource.
func NewOperationResource(op types.OperationSummary) *OperationResource {
	return &OperationResource{
		BaseResource: dao.BaseResource{
			ID:   appaws.Str(op.Id),
			ARN:  appaws.Str(op.TargetArn),
			Data: op,
		},
		Item: op,
	}
}

// OperationType returns the operation type.
func (r *OperationResource) OperationType() string {
	return string(r.Item.Type)
}

// Status returns the operation status.
func (r *OperationResource) Status() string {
	return string(r.Item.Status)
}

// StartedAt returns when the operation started.
func (r *OperationResource) StartedAt() *time.Time {
	return r.Item.StartedAt
}

// EndedAt returns when the operation ended.
func (r *OperationResource) EndedAt() *time.Time {
	return r.Item.EndedAt
}

// UpdatedAt returns when the operation was last updated.
func (r *OperationResource) UpdatedAt() *time.Time {
	return r.Item.UpdatedAt
}

// TargetArn returns the target ARN.
func (r *OperationResource) TargetArn() string {
	return appaws.Str(r.Item.TargetArn)
}
