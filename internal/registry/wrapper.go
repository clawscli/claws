package registry

import (
	"context"
	"strings"

	"github.com/clawscli/claws/internal/aws"
	"github.com/clawscli/claws/internal/dao"
)

// stripRegionPrefix removes region prefix from ID if it matches the expected region (format: "region:id" → "id")
func stripRegionPrefix(id, region string) string {
	if region == "" {
		return id
	}
	prefix := region + ":"
	if strings.HasPrefix(id, prefix) {
		return id[len(prefix):]
	}
	return id
}

// RegionalDAOWrapper wraps any DAO to automatically support multi-region queries.
// It detects region overrides in the context and wraps returned resources with region metadata.
//
// This allows all 164 custom DAOs to support multi-region without modification:
// - Gets the region from context.Context (set by view layer for multi-region queries)
// - Wraps each resource with region metadata via dao.WrapWithRegion()
// - Resources returned from List/Get have GetResourceRegion() support
//
// For single-region queries (no region override), resources are unwrapped
// to preserve backward compatibility with existing code.
type RegionalDAOWrapper struct {
	delegate dao.DAO
	ctx      context.Context
	region   string // empty string = use default region
}

// NewRegionalDAOWrapper creates a new wrapper for a DAO.
// It automatically detects region overrides from the context.
func NewRegionalDAOWrapper(ctx context.Context, delegate dao.DAO) dao.DAO {
	region := aws.GetRegionFromContext(ctx)

	// If no region override, return unwrapped DAO for backward compatibility
	if region == "" {
		return delegate
	}

	return &RegionalDAOWrapper{
		delegate: delegate,
		ctx:      ctx,
		region:   region,
	}
}

func (w *RegionalDAOWrapper) ServiceName() string {
	return w.delegate.ServiceName()
}

func (w *RegionalDAOWrapper) ResourceType() string {
	return w.delegate.ResourceType()
}

// List wraps all resources with region metadata
func (w *RegionalDAOWrapper) List(ctx context.Context) ([]dao.Resource, error) {
	resources, err := w.delegate.List(ctx)
	if err != nil {
		return nil, err
	}

	// Wrap each resource with region metadata
	wrapped := make([]dao.Resource, len(resources))
	for i, res := range resources {
		wrapped[i] = dao.WrapWithRegion(res, w.region)
	}
	return wrapped, nil
}

// Get wraps the resource with region metadata
func (w *RegionalDAOWrapper) Get(ctx context.Context, id string) (dao.Resource, error) {
	res, err := w.delegate.Get(ctx, stripRegionPrefix(id, w.region))
	if err != nil {
		return nil, err
	}
	return dao.WrapWithRegion(res, w.region), nil
}

// Delete delegates to wrapped DAO
func (w *RegionalDAOWrapper) Delete(ctx context.Context, id string) error {
	return w.delegate.Delete(ctx, stripRegionPrefix(id, w.region))
}

// Supports delegates to wrapped DAO
func (w *RegionalDAOWrapper) Supports(op dao.Operation) bool {
	return w.delegate.Supports(op)
}

// PaginatedDAOWrapper wraps a PaginatedDAO to support multi-region pagination.
// Preserves pagination support while adding region wrapping.
type PaginatedDAOWrapper struct {
	RegionalDAOWrapper
	delegate dao.PaginatedDAO
}

// NewPaginatedDAOWrapper creates a wrapper for a PaginatedDAO
func NewPaginatedDAOWrapper(ctx context.Context, delegate dao.PaginatedDAO) dao.PaginatedDAO {
	region := aws.GetRegionFromContext(ctx)

	// If no region override, return unwrapped DAO
	if region == "" {
		return delegate
	}

	return &PaginatedDAOWrapper{
		RegionalDAOWrapper: RegionalDAOWrapper{
			delegate: delegate,
			ctx:      ctx,
			region:   region,
		},
		delegate: delegate,
	}
}

// ListPage wraps all resources with region metadata
func (w *PaginatedDAOWrapper) ListPage(ctx context.Context, pageSize int, pageToken string) ([]dao.Resource, string, error) {
	resources, nextToken, err := w.delegate.ListPage(ctx, pageSize, pageToken)
	if err != nil {
		return nil, "", err
	}

	// Wrap each resource with region metadata
	wrapped := make([]dao.Resource, len(resources))
	for i, res := range resources {
		wrapped[i] = dao.WrapWithRegion(res, w.region)
	}
	return wrapped, nextToken, nil
}
