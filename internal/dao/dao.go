package dao

import (
	"context"
)

// Resource represents a generic AWS resource
type Resource interface {
	GetID() string
	GetName() string
	GetARN() string
	GetTags() map[string]string
	Raw() any
}

// DAO defines the interface for data access operations on AWS resources
type DAO interface {
	// ServiceName returns the AWS service name (e.g., "ec2", "s3")
	ServiceName() string

	// ResourceType returns the resource type (e.g., "instances", "buckets")
	ResourceType() string

	// List retrieves all resources of this type
	List(ctx context.Context) ([]Resource, error)

	// Get retrieves a single resource by ID
	Get(ctx context.Context, id string) (Resource, error)

	// Delete removes a resource by ID (if supported)
	Delete(ctx context.Context, id string) error

	// Supports returns whether this DAO supports the given operation
	Supports(op Operation) bool
}

// Operation represents a supported operation type
type Operation string

const (
	OpList   Operation = "list"
	OpGet    Operation = "get"
	OpCreate Operation = "create"
	OpDelete Operation = "delete"
	OpUpdate Operation = "update"
)

// BaseResource provides a default implementation of Resource
type BaseResource struct {
	ID   string
	Name string
	ARN  string
	Tags map[string]string
	Data any
}

func (r *BaseResource) GetID() string              { return r.ID }
func (r *BaseResource) GetName() string            { return r.Name }
func (r *BaseResource) GetARN() string             { return r.ARN }
func (r *BaseResource) GetTags() map[string]string { return r.Tags }
func (r *BaseResource) Raw() any                   { return r.Data }

// BaseDAO provides common DAO functionality.
// Embed this in your DAO struct to get default implementations.
type BaseDAO struct {
	service  string
	resource string
}

// NewBaseDAO creates a new BaseDAO with the given service and resource names.
func NewBaseDAO(service, resource string) BaseDAO {
	return BaseDAO{service: service, resource: resource}
}

func (d *BaseDAO) ServiceName() string  { return d.service }
func (d *BaseDAO) ResourceType() string { return d.resource }

// Supports returns true for List, Get, and Delete operations by default.
// Override this method if your DAO has different capabilities.
func (d *BaseDAO) Supports(op Operation) bool {
	switch op {
	case OpList, OpGet, OpDelete:
		return true
	default:
		return false
	}
}

// Factory creates DAO instances
type Factory func(ctx context.Context) (DAO, error)

// PaginatedDAO extends DAO with pagination support for large result sets.
// Implement this interface for resources that can have thousands of items
// (e.g., CloudTrail events, CloudWatch logs).
// ResourceBrowser will automatically detect and use pagination when available.
type PaginatedDAO interface {
	DAO
	// ListPage retrieves a page of resources.
	// pageSize: number of items to retrieve (e.g., 100)
	// pageToken: token for the next page (empty string for first page)
	// Returns: resources, next page token (empty if no more pages), error
	ListPage(ctx context.Context, pageSize int, pageToken string) ([]Resource, string, error)
}

// Context key types for filter values
type filterContextKey string

const filterPrefix filterContextKey = "dao_filter_"

// WithFilter adds a filter value to the context
func WithFilter(ctx context.Context, key, value string) context.Context {
	return context.WithValue(ctx, filterPrefix+filterContextKey(key), value)
}

// GetFilterFromContext retrieves a filter value from the context
func GetFilterFromContext(ctx context.Context, key string) string {
	if v := ctx.Value(filterPrefix + filterContextKey(key)); v != nil {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}
