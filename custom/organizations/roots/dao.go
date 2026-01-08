package roots

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/organizations"
	"github.com/aws/aws-sdk-go-v2/service/organizations/types"

	appaws "github.com/clawscli/claws/internal/aws"
	"github.com/clawscli/claws/internal/dao"
	apperrors "github.com/clawscli/claws/internal/errors"
)

// RootDAO provides data access for Organizations roots.
type RootDAO struct {
	dao.BaseDAO
	client *organizations.Client
}

// NewRootDAO creates a new RootDAO.
func NewRootDAO(ctx context.Context) (dao.DAO, error) {
	cfg, err := appaws.NewConfig(ctx)
	if err != nil {
		return nil, apperrors.Wrap(err, "new "+ServiceResourcePath+" dao")
	}
	return &RootDAO{
		BaseDAO: dao.NewBaseDAO("organizations", "roots"),
		client:  organizations.NewFromConfig(cfg),
	}, nil
}

// List returns all roots in the organization.
func (d *RootDAO) List(ctx context.Context) ([]dao.Resource, error) {
	roots, err := appaws.Paginate(ctx, func(token *string) ([]types.Root, *string, error) {
		output, err := d.client.ListRoots(ctx, &organizations.ListRootsInput{
			NextToken: token,
		})
		if err != nil {
			return nil, nil, apperrors.Wrap(err, "list organizations roots")
		}
		return output.Roots, output.NextToken, nil
	})
	if err != nil {
		return nil, err
	}

	resources := make([]dao.Resource, len(roots))
	for i, root := range roots {
		resources[i] = NewRootResource(root)
	}
	return resources, nil
}

// Get returns a specific root.
func (d *RootDAO) Get(ctx context.Context, id string) (dao.Resource, error) {
	resources, err := d.List(ctx)
	if err != nil {
		return nil, err
	}
	for _, r := range resources {
		if r.GetID() == id {
			return r, nil
		}
	}
	return nil, fmt.Errorf("root not found: %s", id)
}

// Delete is not supported for roots.
func (d *RootDAO) Delete(ctx context.Context, id string) error {
	return fmt.Errorf("delete not supported for organization roots")
}

// RootResource wraps an Organizations root.
type RootResource struct {
	dao.BaseResource
	Root *types.Root
}

// NewRootResource creates a new RootResource.
func NewRootResource(root types.Root) *RootResource {
	return &RootResource{
		BaseResource: dao.BaseResource{
			ID:   appaws.Str(root.Id),
			ARN:  appaws.Str(root.Arn),
			Data: root,
		},
		Root: &root,
	}
}

// Name returns the root name.
func (r *RootResource) Name() string {
	if r.Root != nil && r.Root.Name != nil {
		return *r.Root.Name
	}
	return ""
}

// PolicyTypes returns the enabled policy types.
func (r *RootResource) PolicyTypes() []types.PolicyTypeSummary {
	if r.Root != nil {
		return r.Root.PolicyTypes
	}
	return nil
}
