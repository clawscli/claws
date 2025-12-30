package certificates

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/acm"
	"github.com/aws/aws-sdk-go-v2/service/acm/types"

	appaws "github.com/clawscli/claws/internal/aws"
	"github.com/clawscli/claws/internal/dao"
)

// CertificateDAO provides data access for ACM certificates
type CertificateDAO struct {
	dao.BaseDAO
	client *acm.Client
}

// NewCertificateDAO creates a new CertificateDAO
func NewCertificateDAO(ctx context.Context) (dao.DAO, error) {
	cfg, err := appaws.NewConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("new acm/certificates dao: %w", err)
	}
	return &CertificateDAO{
		BaseDAO: dao.NewBaseDAO("acm", "certificates"),
		client:  acm.NewFromConfig(cfg),
	}, nil
}

func (d *CertificateDAO) List(ctx context.Context) ([]dao.Resource, error) {
	summaries, err := appaws.Paginate(ctx, func(token *string) ([]types.CertificateSummary, *string, error) {
		output, err := d.client.ListCertificates(ctx, &acm.ListCertificatesInput{
			NextToken: token,
			MaxItems:  appaws.Int32Ptr(100),
			Includes:  &types.Filters{},
		})
		if err != nil {
			return nil, nil, fmt.Errorf("list certificates: %w", err)
		}
		return output.CertificateSummaryList, output.NextToken, nil
	})
	if err != nil {
		return nil, err
	}

	// CertificateSummary contains all fields needed for list view
	// No need for N+1 DescribeCertificate calls
	resources := make([]dao.Resource, 0, len(summaries))
	for _, cert := range summaries {
		resources = append(resources, NewCertificateResourceFromSummary(cert))
	}

	return resources, nil
}

func (d *CertificateDAO) Get(ctx context.Context, id string) (dao.Resource, error) {
	input := &acm.DescribeCertificateInput{
		CertificateArn: &id,
	}

	output, err := d.client.DescribeCertificate(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("describe certificate %s: %w", id, err)
	}

	return NewCertificateResource(output.Certificate), nil
}

func (d *CertificateDAO) Delete(ctx context.Context, id string) error {
	input := &acm.DeleteCertificateInput{
		CertificateArn: &id,
	}

	_, err := d.client.DeleteCertificate(ctx, input)
	if err != nil {
		if appaws.IsNotFound(err) {
			return nil // Already deleted
		}
		if appaws.IsResourceInUse(err) {
			return fmt.Errorf("certificate %s is in use by AWS resources", id)
		}
		return fmt.Errorf("delete certificate %s: %w", id, err)
	}

	return nil
}

// CertificateResource wraps an ACM certificate
type CertificateResource struct {
	dao.BaseResource
	Item    *types.CertificateDetail  // Full details (from Get/DescribeCertificate)
	Summary *types.CertificateSummary // Summary (from List, avoids N+1 calls)
}

// NewCertificateResource creates a new CertificateResource from CertificateDetail
func NewCertificateResource(cert *types.CertificateDetail) *CertificateResource {
	arn := appaws.Str(cert.CertificateArn)
	domain := appaws.Str(cert.DomainName)

	// Convert tags
	tags := make(map[string]string)

	return &CertificateResource{
		BaseResource: dao.BaseResource{
			ID:   arn,
			Name: domain,
			ARN:  arn,
			Tags: tags,
			Data: cert,
		},
		Item: cert,
	}
}

// NewCertificateResourceFromSummary creates a new CertificateResource from CertificateSummary
// Used for list view to avoid N+1 DescribeCertificate calls
func NewCertificateResourceFromSummary(cert types.CertificateSummary) *CertificateResource {
	arn := appaws.Str(cert.CertificateArn)
	domain := appaws.Str(cert.DomainName)

	return &CertificateResource{
		BaseResource: dao.BaseResource{
			ID:   arn,
			Name: domain,
			ARN:  arn,
			Tags: make(map[string]string),
			Data: cert,
		},
		Summary: &cert,
	}
}

// DomainName returns the primary domain name
func (r *CertificateResource) DomainName() string {
	if r.Item != nil {
		return appaws.Str(r.Item.DomainName)
	}
	if r.Summary != nil {
		return appaws.Str(r.Summary.DomainName)
	}
	return ""
}

// Status returns the certificate status
func (r *CertificateResource) Status() string {
	if r.Item != nil {
		return string(r.Item.Status)
	}
	if r.Summary != nil {
		return string(r.Summary.Status)
	}
	return ""
}

// Type returns the certificate type (IMPORTED or AMAZON_ISSUED)
func (r *CertificateResource) Type() string {
	if r.Item != nil {
		return string(r.Item.Type)
	}
	if r.Summary != nil {
		return string(r.Summary.Type)
	}
	return ""
}

// KeyAlgorithm returns the key algorithm
func (r *CertificateResource) KeyAlgorithm() string {
	if r.Item != nil {
		return string(r.Item.KeyAlgorithm)
	}
	if r.Summary != nil {
		return string(r.Summary.KeyAlgorithm)
	}
	return ""
}

// Issuer returns the certificate issuer
func (r *CertificateResource) Issuer() string {
	if r.Item == nil {
		return ""
	}
	return appaws.Str(r.Item.Issuer)
}

// NotBefore returns the not before date as string
func (r *CertificateResource) NotBefore() string {
	if r.Item != nil && r.Item.NotBefore != nil {
		return r.Item.NotBefore.Format("2006-01-02")
	}
	if r.Summary != nil && r.Summary.NotBefore != nil {
		return r.Summary.NotBefore.Format("2006-01-02")
	}
	return ""
}

// NotAfter returns the not after date as string
func (r *CertificateResource) NotAfter() string {
	if r.Item != nil && r.Item.NotAfter != nil {
		return r.Item.NotAfter.Format("2006-01-02")
	}
	if r.Summary != nil && r.Summary.NotAfter != nil {
		return r.Summary.NotAfter.Format("2006-01-02")
	}
	return ""
}

// CreatedAt returns the creation date as string
func (r *CertificateResource) CreatedAt() string {
	if r.Item != nil && r.Item.CreatedAt != nil {
		return r.Item.CreatedAt.Format("2006-01-02 15:04:05")
	}
	if r.Summary != nil && r.Summary.CreatedAt != nil {
		return r.Summary.CreatedAt.Format("2006-01-02 15:04:05")
	}
	return ""
}

// IssuedAt returns the issued date as string
func (r *CertificateResource) IssuedAt() string {
	if r.Item != nil && r.Item.IssuedAt != nil {
		return r.Item.IssuedAt.Format("2006-01-02 15:04:05")
	}
	if r.Summary != nil && r.Summary.IssuedAt != nil {
		return r.Summary.IssuedAt.Format("2006-01-02 15:04:05")
	}
	return ""
}

// RenewalEligibility returns the renewal eligibility
func (r *CertificateResource) RenewalEligibility() string {
	if r.Item != nil {
		return string(r.Item.RenewalEligibility)
	}
	if r.Summary != nil {
		return string(r.Summary.RenewalEligibility)
	}
	return ""
}

// InUseBy returns the resources using this certificate
// Only available from Item (detail view), Summary only has bool flag
func (r *CertificateResource) InUseBy() []string {
	if r.Item != nil {
		return r.Item.InUseBy
	}
	return nil
}

// IsInUse returns whether the certificate is in use (available from Summary)
func (r *CertificateResource) IsInUse() *bool {
	if r.Item != nil {
		inUse := len(r.Item.InUseBy) > 0
		return &inUse
	}
	if r.Summary != nil {
		return r.Summary.InUse
	}
	return nil
}

// SubjectAlternativeNames returns the SANs
func (r *CertificateResource) SubjectAlternativeNames() []string {
	if r.Item != nil {
		return r.Item.SubjectAlternativeNames
	}
	if r.Summary != nil {
		return r.Summary.SubjectAlternativeNameSummaries
	}
	return nil
}

// DomainValidationOptions returns the domain validation options
func (r *CertificateResource) DomainValidationOptions() []types.DomainValidation {
	if r.Item == nil {
		return nil
	}
	return r.Item.DomainValidationOptions
}

// Serial returns the certificate serial number
func (r *CertificateResource) Serial() string {
	if r.Item == nil {
		return ""
	}
	return appaws.Str(r.Item.Serial)
}

// SignatureAlgorithm returns the signature algorithm
func (r *CertificateResource) SignatureAlgorithm() string {
	if r.Item == nil {
		return ""
	}
	return appaws.Str(r.Item.SignatureAlgorithm)
}

// Subject returns the certificate subject
func (r *CertificateResource) Subject() string {
	if r.Item == nil {
		return ""
	}
	return appaws.Str(r.Item.Subject)
}

// CertificateAuthorityArn returns the private CA ARN (for private certificates)
func (r *CertificateResource) CertificateAuthorityArn() string {
	if r.Item == nil {
		return ""
	}
	return appaws.Str(r.Item.CertificateAuthorityArn)
}

// KeyUsages returns the key usages
func (r *CertificateResource) KeyUsages() []string {
	if r.Item == nil {
		return nil
	}
	usages := make([]string, len(r.Item.KeyUsages))
	for i, ku := range r.Item.KeyUsages {
		usages[i] = string(ku.Name)
	}
	return usages
}

// ExtendedKeyUsages returns the extended key usages
func (r *CertificateResource) ExtendedKeyUsages() []types.ExtendedKeyUsage {
	if r.Item == nil {
		return nil
	}
	return r.Item.ExtendedKeyUsages
}

// CertificateTransparencyLogging returns the certificate transparency logging status
func (r *CertificateResource) CertificateTransparencyLogging() string {
	if r.Item != nil && r.Item.Options != nil {
		return string(r.Item.Options.CertificateTransparencyLoggingPreference)
	}
	return ""
}

// RenewalSummary returns the renewal summary (for AMAZON_ISSUED certificates)
func (r *CertificateResource) RenewalSummary() *types.RenewalSummary {
	if r.Item == nil {
		return nil
	}
	return r.Item.RenewalSummary
}

// FailureReason returns the failure reason (when status is FAILED)
func (r *CertificateResource) FailureReason() string {
	if r.Item == nil {
		return ""
	}
	return string(r.Item.FailureReason)
}

// RevocationReason returns the revocation reason (when status is REVOKED)
func (r *CertificateResource) RevocationReason() string {
	if r.Item == nil {
		return ""
	}
	return string(r.Item.RevocationReason)
}

// RevokedAt returns the revocation date
func (r *CertificateResource) RevokedAt() string {
	if r.Item != nil && r.Item.RevokedAt != nil {
		return r.Item.RevokedAt.Format("2006-01-02 15:04:05")
	}
	if r.Summary != nil && r.Summary.RevokedAt != nil {
		return r.Summary.RevokedAt.Format("2006-01-02 15:04:05")
	}
	return ""
}

// ImportedAt returns the import date (for IMPORTED certificates)
func (r *CertificateResource) ImportedAt() string {
	if r.Item != nil && r.Item.ImportedAt != nil {
		return r.Item.ImportedAt.Format("2006-01-02 15:04:05")
	}
	if r.Summary != nil && r.Summary.ImportedAt != nil {
		return r.Summary.ImportedAt.Format("2006-01-02 15:04:05")
	}
	return ""
}

// ManagedBy returns the service managing the certificate
func (r *CertificateResource) ManagedBy() string {
	if r.Item != nil {
		return string(r.Item.ManagedBy)
	}
	if r.Summary != nil {
		return string(r.Summary.ManagedBy)
	}
	return ""
}
