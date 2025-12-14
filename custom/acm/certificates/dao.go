package certificates

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/acm"
	"github.com/aws/aws-sdk-go-v2/service/acm/types"
	appaws "github.com/clawscli/claws/internal/aws"
	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/log"
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
		return nil, err
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

	// Get certificate details for each summary
	resources := make([]dao.Resource, 0, len(summaries))
	for _, cert := range summaries {
		describeOutput, err := d.client.DescribeCertificate(ctx, &acm.DescribeCertificateInput{
			CertificateArn: cert.CertificateArn,
		})
		if err != nil {
			log.Warn("failed to describe certificate", "arn", appaws.Str(cert.CertificateArn), "error", err)
			continue
		}

		resources = append(resources, NewCertificateResource(describeOutput.Certificate))
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
	Item *types.CertificateDetail
}

// NewCertificateResource creates a new CertificateResource
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

// DomainName returns the primary domain name
func (r *CertificateResource) DomainName() string {
	return appaws.Str(r.Item.DomainName)
}

// Status returns the certificate status
func (r *CertificateResource) Status() string {
	return string(r.Item.Status)
}

// Type returns the certificate type (IMPORTED or AMAZON_ISSUED)
func (r *CertificateResource) Type() string {
	return string(r.Item.Type)
}

// KeyAlgorithm returns the key algorithm
func (r *CertificateResource) KeyAlgorithm() string {
	return string(r.Item.KeyAlgorithm)
}

// Issuer returns the certificate issuer
func (r *CertificateResource) Issuer() string {
	return appaws.Str(r.Item.Issuer)
}

// NotBefore returns the not before date as string
func (r *CertificateResource) NotBefore() string {
	if r.Item.NotBefore != nil {
		return r.Item.NotBefore.Format("2006-01-02")
	}
	return ""
}

// NotAfter returns the not after date as string
func (r *CertificateResource) NotAfter() string {
	if r.Item.NotAfter != nil {
		return r.Item.NotAfter.Format("2006-01-02")
	}
	return ""
}

// CreatedAt returns the creation date as string
func (r *CertificateResource) CreatedAt() string {
	if r.Item.CreatedAt != nil {
		return r.Item.CreatedAt.Format("2006-01-02 15:04:05")
	}
	return ""
}

// IssuedAt returns the issued date as string
func (r *CertificateResource) IssuedAt() string {
	if r.Item.IssuedAt != nil {
		return r.Item.IssuedAt.Format("2006-01-02 15:04:05")
	}
	return ""
}

// RenewalEligibility returns the renewal eligibility
func (r *CertificateResource) RenewalEligibility() string {
	return string(r.Item.RenewalEligibility)
}

// InUseBy returns the resources using this certificate
func (r *CertificateResource) InUseBy() []string {
	return r.Item.InUseBy
}

// SubjectAlternativeNames returns the SANs
func (r *CertificateResource) SubjectAlternativeNames() []string {
	return r.Item.SubjectAlternativeNames
}

// DomainValidationOptions returns the domain validation options
func (r *CertificateResource) DomainValidationOptions() []types.DomainValidation {
	return r.Item.DomainValidationOptions
}

// Serial returns the certificate serial number
func (r *CertificateResource) Serial() string {
	return appaws.Str(r.Item.Serial)
}

// SignatureAlgorithm returns the signature algorithm
func (r *CertificateResource) SignatureAlgorithm() string {
	return appaws.Str(r.Item.SignatureAlgorithm)
}

// Subject returns the certificate subject
func (r *CertificateResource) Subject() string {
	return appaws.Str(r.Item.Subject)
}

// CertificateAuthorityArn returns the private CA ARN (for private certificates)
func (r *CertificateResource) CertificateAuthorityArn() string {
	return appaws.Str(r.Item.CertificateAuthorityArn)
}

// KeyUsages returns the key usages
func (r *CertificateResource) KeyUsages() []string {
	usages := make([]string, len(r.Item.KeyUsages))
	for i, ku := range r.Item.KeyUsages {
		usages[i] = string(ku.Name)
	}
	return usages
}

// ExtendedKeyUsages returns the extended key usages
func (r *CertificateResource) ExtendedKeyUsages() []types.ExtendedKeyUsage {
	return r.Item.ExtendedKeyUsages
}

// CertificateTransparencyLogging returns the certificate transparency logging status
func (r *CertificateResource) CertificateTransparencyLogging() string {
	if r.Item.Options != nil {
		return string(r.Item.Options.CertificateTransparencyLoggingPreference)
	}
	return ""
}

// RenewalSummary returns the renewal summary (for AMAZON_ISSUED certificates)
func (r *CertificateResource) RenewalSummary() *types.RenewalSummary {
	return r.Item.RenewalSummary
}

// FailureReason returns the failure reason (when status is FAILED)
func (r *CertificateResource) FailureReason() string {
	return string(r.Item.FailureReason)
}

// RevocationReason returns the revocation reason (when status is REVOKED)
func (r *CertificateResource) RevocationReason() string {
	return string(r.Item.RevocationReason)
}

// RevokedAt returns the revocation date
func (r *CertificateResource) RevokedAt() string {
	if r.Item.RevokedAt != nil {
		return r.Item.RevokedAt.Format("2006-01-02 15:04:05")
	}
	return ""
}

// ImportedAt returns the import date (for IMPORTED certificates)
func (r *CertificateResource) ImportedAt() string {
	if r.Item.ImportedAt != nil {
		return r.Item.ImportedAt.Format("2006-01-02 15:04:05")
	}
	return ""
}

// ManagedBy returns the service managing the certificate
func (r *CertificateResource) ManagedBy() string {
	return string(r.Item.ManagedBy)
}
