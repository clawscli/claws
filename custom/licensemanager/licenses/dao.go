package licenses

import (
	"context"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/licensemanager"
	"github.com/aws/aws-sdk-go-v2/service/licensemanager/types"

	appaws "github.com/clawscli/claws/internal/aws"
	"github.com/clawscli/claws/internal/dao"
)

// LicenseDAO provides data access for License Manager licenses.
type LicenseDAO struct {
	dao.BaseDAO
	client *licensemanager.Client
}

// NewLicenseDAO creates a new LicenseDAO.
func NewLicenseDAO(ctx context.Context) (dao.DAO, error) {
	cfg, err := appaws.NewConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("new licensemanager/licenses dao: %w", err)
	}
	return &LicenseDAO{
		BaseDAO: dao.NewBaseDAO("license-manager", "licenses"),
		client:  licensemanager.NewFromConfig(cfg),
	}, nil
}

// List returns all licenses.
func (d *LicenseDAO) List(ctx context.Context) ([]dao.Resource, error) {
	licenses, err := appaws.Paginate(ctx, func(token *string) ([]types.License, *string, error) {
		output, err := d.client.ListLicenses(ctx, &licensemanager.ListLicensesInput{
			NextToken: token,
		})
		if err != nil {
			return nil, nil, fmt.Errorf("list licenses: %w", err)
		}
		return output.Licenses, output.NextToken, nil
	})
	if err != nil {
		return nil, err
	}

	resources := make([]dao.Resource, len(licenses))
	for i, license := range licenses {
		resources[i] = NewLicenseResource(license)
	}
	return resources, nil
}

// Get returns a specific license.
func (d *LicenseDAO) Get(ctx context.Context, arn string) (dao.Resource, error) {
	output, err := d.client.GetLicense(ctx, &licensemanager.GetLicenseInput{
		LicenseArn: &arn,
	})
	if err != nil {
		return nil, fmt.Errorf("get license: %w", err)
	}
	return NewLicenseResource(*output.License), nil
}

// Delete deletes a license.
func (d *LicenseDAO) Delete(ctx context.Context, arn string) error {
	// Get the license version first
	output, err := d.client.GetLicense(ctx, &licensemanager.GetLicenseInput{
		LicenseArn: &arn,
	})
	if err != nil {
		return fmt.Errorf("get license for delete: %w", err)
	}

	_, err = d.client.DeleteLicense(ctx, &licensemanager.DeleteLicenseInput{
		LicenseArn:    &arn,
		SourceVersion: output.License.Version,
	})
	if err != nil {
		return fmt.Errorf("delete license: %w", err)
	}
	return nil
}

// LicenseResource wraps a License Manager license.
type LicenseResource struct {
	dao.BaseResource
	License *types.License
}

// NewLicenseResource creates a new LicenseResource.
func NewLicenseResource(license types.License) *LicenseResource {
	arn := appaws.Str(license.LicenseArn)
	// Extract ID from ARN
	id := arn
	if idx := strings.LastIndex(arn, "/"); idx != -1 {
		id = arn[idx+1:]
	}
	return &LicenseResource{
		BaseResource: dao.BaseResource{
			ID:  id,
			ARN: arn,
		},
		License: &license,
	}
}

// Name returns the license name.
func (r *LicenseResource) Name() string {
	if r.License != nil && r.License.LicenseName != nil {
		return *r.License.LicenseName
	}
	return ""
}

// ProductName returns the product name.
func (r *LicenseResource) ProductName() string {
	if r.License != nil && r.License.ProductName != nil {
		return *r.License.ProductName
	}
	return ""
}

// Status returns the license status.
func (r *LicenseResource) Status() string {
	if r.License != nil {
		return string(r.License.Status)
	}
	return ""
}

// Issuer returns the license issuer.
func (r *LicenseResource) Issuer() string {
	if r.License != nil && r.License.Issuer != nil && r.License.Issuer.Name != nil {
		return *r.License.Issuer.Name
	}
	return ""
}

// Beneficiary returns the license beneficiary.
func (r *LicenseResource) Beneficiary() string {
	if r.License != nil && r.License.Beneficiary != nil {
		return *r.License.Beneficiary
	}
	return ""
}
