package ous

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/organizations"
	"github.com/aws/aws-sdk-go-v2/service/organizations/types"

	appaws "github.com/clawscli/claws/internal/aws"
	"github.com/clawscli/claws/internal/dao"
	apperrors "github.com/clawscli/claws/internal/errors"
)

// OUDAO provides data access for Organizations OUs.
type OUDAO struct {
	dao.BaseDAO
	client *organizations.Client
}

// NewOUDAO creates a new OUDAO.
func NewOUDAO(ctx context.Context) (dao.DAO, error) {
	cfg, err := appaws.NewConfig(ctx)
	if err != nil {
		return nil, apperrors.Wrap(err, "new "+ServiceResourcePath+" dao")
	}
	return &OUDAO{
		BaseDAO: dao.NewBaseDAO("organizations", "ous"),
		client:  organizations.NewFromConfig(cfg),
	}, nil
}

// List returns OUs under the specified parent.
func (d *OUDAO) List(ctx context.Context) ([]dao.Resource, error) {
	parentId := dao.GetFilterFromContext(ctx, "ParentId")
	if parentId == "" {
		return nil, fmt.Errorf("parent ID filter required")
	}

	ous, err := appaws.Paginate(ctx, func(token *string) ([]types.OrganizationalUnit, *string, error) {
		output, err := d.client.ListOrganizationalUnitsForParent(ctx, &organizations.ListOrganizationalUnitsForParentInput{
			ParentId:  &parentId,
			NextToken: token,
		})
		if err != nil {
			return nil, nil, apperrors.Wrap(err, "list organizations OUs")
		}
		return output.OrganizationalUnits, output.NextToken, nil
	})
	if err != nil {
		return nil, err
	}

	resources := make([]dao.Resource, len(ous))
	for i, ou := range ous {
		resources[i] = NewOUResource(ou)
	}
	return resources, nil
}

// Get returns a specific OU.
func (d *OUDAO) Get(ctx context.Context, id string) (dao.Resource, error) {
	output, err := d.client.DescribeOrganizationalUnit(ctx, &organizations.DescribeOrganizationalUnitInput{
		OrganizationalUnitId: &id,
	})
	if err != nil {
		return nil, apperrors.Wrap(err, "describe organizations OU")
	}
	return NewOUResource(*output.OrganizationalUnit), nil
}

// Delete deletes an OU.
func (d *OUDAO) Delete(ctx context.Context, id string) error {
	_, err := d.client.DeleteOrganizationalUnit(ctx, &organizations.DeleteOrganizationalUnitInput{
		OrganizationalUnitId: &id,
	})
	if err != nil {
		return apperrors.Wrap(err, "delete organizations OU")
	}
	return nil
}

// OUResource wraps an Organizations OU.
type OUResource struct {
	dao.BaseResource
	OU *types.OrganizationalUnit
}

// NewOUResource creates a new OUResource.
func NewOUResource(ou types.OrganizationalUnit) *OUResource {
	return &OUResource{
		BaseResource: dao.BaseResource{
			ID:  appaws.Str(ou.Id),
			ARN: appaws.Str(ou.Arn),
		},
		OU: &ou,
	}
}

// Name returns the OU name.
func (r *OUResource) Name() string {
	if r.OU != nil && r.OU.Name != nil {
		return *r.OU.Name
	}
	return ""
}
