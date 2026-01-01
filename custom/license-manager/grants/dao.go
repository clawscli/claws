package grants

import (
	"context"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/licensemanager"
	"github.com/aws/aws-sdk-go-v2/service/licensemanager/types"

	appaws "github.com/clawscli/claws/internal/aws"
	"github.com/clawscli/claws/internal/dao"
	apperrors "github.com/clawscli/claws/internal/errors"
)

// GrantDAO provides data access for License Manager grants.
type GrantDAO struct {
	dao.BaseDAO
	client *licensemanager.Client
}

// NewGrantDAO creates a new GrantDAO.
func NewGrantDAO(ctx context.Context) (dao.DAO, error) {
	cfg, err := appaws.NewConfig(ctx)
	if err != nil {
		return nil, apperrors.Wrap(err, "new licensemanager/grants dao")
	}
	return &GrantDAO{
		BaseDAO: dao.NewBaseDAO("license-manager", "grants"),
		client:  licensemanager.NewFromConfig(cfg),
	}, nil
}

// List returns grants for the specified license.
func (d *GrantDAO) List(ctx context.Context) ([]dao.Resource, error) {
	licenseArn := dao.GetFilterFromContext(ctx, "LicenseArn")
	if licenseArn == "" {
		return nil, fmt.Errorf("license ARN filter required")
	}

	grants, err := appaws.Paginate(ctx, func(token *string) ([]types.Grant, *string, error) {
		output, err := d.client.ListDistributedGrants(ctx, &licensemanager.ListDistributedGrantsInput{
			Filters: []types.Filter{
				{
					Name:   stringPtr("LicenseArn"),
					Values: []string{licenseArn},
				},
			},
			NextToken: token,
		})
		if err != nil {
			return nil, nil, apperrors.Wrap(err, "list license grants")
		}
		return output.Grants, output.NextToken, nil
	})
	if err != nil {
		return nil, err
	}

	resources := make([]dao.Resource, len(grants))
	for i, grant := range grants {
		resources[i] = NewGrantResource(grant)
	}
	return resources, nil
}

func stringPtr(s string) *string {
	return &s
}

// Get returns a specific grant.
func (d *GrantDAO) Get(ctx context.Context, arn string) (dao.Resource, error) {
	output, err := d.client.GetGrant(ctx, &licensemanager.GetGrantInput{
		GrantArn: &arn,
	})
	if err != nil {
		return nil, apperrors.Wrap(err, "get grant")
	}
	return NewGrantResource(*output.Grant), nil
}

// Delete deletes a grant.
func (d *GrantDAO) Delete(ctx context.Context, arn string) error {
	// Get the grant version first
	output, err := d.client.GetGrant(ctx, &licensemanager.GetGrantInput{
		GrantArn: &arn,
	})
	if err != nil {
		return apperrors.Wrap(err, "get grant for delete")
	}

	_, err = d.client.DeleteGrant(ctx, &licensemanager.DeleteGrantInput{
		GrantArn: &arn,
		Version:  output.Grant.Version,
	})
	if err != nil {
		return apperrors.Wrap(err, "delete grant")
	}
	return nil
}

// GrantResource wraps a License Manager grant.
type GrantResource struct {
	dao.BaseResource
	Grant *types.Grant
}

// NewGrantResource creates a new GrantResource.
func NewGrantResource(grant types.Grant) *GrantResource {
	arn := appaws.Str(grant.GrantArn)
	// Extract ID from ARN
	id := arn
	if idx := strings.LastIndex(arn, "/"); idx != -1 {
		id = arn[idx+1:]
	}
	return &GrantResource{
		BaseResource: dao.BaseResource{
			ID:  id,
			ARN: arn,
		},
		Grant: &grant,
	}
}

// Name returns the grant name.
func (r *GrantResource) Name() string {
	if r.Grant != nil && r.Grant.GrantName != nil {
		return *r.Grant.GrantName
	}
	return ""
}

// GranteePrincipal returns the grantee principal.
func (r *GrantResource) GranteePrincipal() string {
	if r.Grant != nil && r.Grant.GranteePrincipalArn != nil {
		return *r.Grant.GranteePrincipalArn
	}
	return ""
}

// Status returns the grant status.
func (r *GrantResource) Status() string {
	if r.Grant != nil {
		return string(r.Grant.GrantStatus)
	}
	return ""
}

// ParentArn returns the parent ARN.
func (r *GrantResource) ParentArn() string {
	if r.Grant != nil && r.Grant.ParentArn != nil {
		return *r.Grant.ParentArn
	}
	return ""
}
