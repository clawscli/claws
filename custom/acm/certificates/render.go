package certificates

import (
	"fmt"
	"strings"
	"time"

	appaws "github.com/clawscli/claws/internal/aws"
	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/render"
)

// CertificateRenderer renders ACM certificates
// Ensure CertificateRenderer implements render.Navigator
var _ render.Navigator = (*CertificateRenderer)(nil)

type CertificateRenderer struct {
	render.BaseRenderer
}

// NewCertificateRenderer creates a new CertificateRenderer
func NewCertificateRenderer() *CertificateRenderer {
	return &CertificateRenderer{
		BaseRenderer: render.BaseRenderer{
			Service:  "acm",
			Resource: "certificates",
			Cols: []render.Column{
				{Name: "DOMAIN", Width: 40, Getter: getDomain},
				{Name: "STATUS", Width: 14, Getter: getStatus},
				{Name: "TYPE", Width: 14, Getter: getType},
				{Name: "EXPIRES", Width: 12, Getter: getExpires},
				{Name: "IN USE", Width: 8, Getter: getInUse},
			},
		},
	}
}

func getDomain(r dao.Resource) string {
	if cert, ok := r.(*CertificateResource); ok {
		domain := cert.DomainName()
		if len(domain) > 38 {
			return domain[:38] + "..."
		}
		return domain
	}
	return ""
}

func getStatus(r dao.Resource) string {
	if cert, ok := r.(*CertificateResource); ok {
		return cert.Status()
	}
	return ""
}

func getType(r dao.Resource) string {
	if cert, ok := r.(*CertificateResource); ok {
		return cert.Type()
	}
	return ""
}

func getExpires(r dao.Resource) string {
	if cert, ok := r.(*CertificateResource); ok {
		return cert.NotAfter()
	}
	return "-"
}

func getInUse(r dao.Resource) string {
	if cert, ok := r.(*CertificateResource); ok {
		count := len(cert.InUseBy())
		if count > 0 {
			return fmt.Sprintf("%d", count)
		}
		return "-"
	}
	return "-"
}

// RenderDetail renders detailed certificate information
func (r *CertificateRenderer) RenderDetail(resource dao.Resource) string {
	cert, ok := resource.(*CertificateResource)
	if !ok {
		return ""
	}

	d := render.NewDetailBuilder()

	d.Title("Certificate", cert.DomainName())

	// Basic Info
	d.Section("Basic Information")
	d.Field("Domain Name", cert.DomainName())
	d.Field("ARN", cert.GetARN())
	d.Field("Status", cert.Status())
	d.Field("Type", cert.Type())
	if subject := cert.Subject(); subject != "" {
		d.Field("Subject", subject)
	}
	if serial := cert.Serial(); serial != "" {
		d.Field("Serial Number", serial)
	}

	// Certificate Details
	d.Section("Certificate Details")
	d.Field("Key Algorithm", cert.KeyAlgorithm())
	if sigAlg := cert.SignatureAlgorithm(); sigAlg != "" {
		d.Field("Signature Algorithm", sigAlg)
	}
	if issuer := cert.Issuer(); issuer != "" {
		d.Field("Issuer", issuer)
	}
	if caArn := cert.CertificateAuthorityArn(); caArn != "" {
		d.Field("Certificate Authority", caArn)
	}
	d.Field("Renewal Eligibility", cert.RenewalEligibility())
	if managedBy := cert.ManagedBy(); managedBy != "" {
		d.Field("Managed By", managedBy)
	}

	// Transparency Logging
	if ctLogging := cert.CertificateTransparencyLogging(); ctLogging != "" {
		d.Field("Transparency Logging", ctLogging)
	}

	// Key Usages
	if keyUsages := cert.KeyUsages(); len(keyUsages) > 0 {
		d.Section("Key Usages")
		d.Field("Usages", strings.Join(keyUsages, ", "))
	}

	// Extended Key Usages
	if extUsages := cert.ExtendedKeyUsages(); len(extUsages) > 0 {
		d.Section("Extended Key Usages")
		for _, eu := range extUsages {
			name := string(eu.Name)
			if eu.OID != nil && *eu.OID != "" {
				d.Field(name, *eu.OID)
			} else {
				d.Field("", name)
			}
		}
	}

	// Validity
	d.Section("Validity")
	if notBefore := cert.NotBefore(); notBefore != "" {
		d.Field("Valid From", notBefore)
	}
	if notAfter := cert.NotAfter(); notAfter != "" {
		d.Field("Valid Until", notAfter)
		// Calculate days until expiry
		if cert.Item.NotAfter != nil {
			daysLeft := int(time.Until(*cert.Item.NotAfter).Hours() / 24)
			if daysLeft >= 0 {
				d.Field("Days Until Expiry", fmt.Sprintf("%d days", daysLeft))
			} else {
				d.Field("Days Until Expiry", "EXPIRED")
			}
		}
	}

	// Subject Alternative Names
	if sans := cert.SubjectAlternativeNames(); len(sans) > 0 {
		d.Section("Subject Alternative Names")
		for _, san := range sans {
			d.Field("", san)
		}
	}

	// In Use By
	if inUseBy := cert.InUseBy(); len(inUseBy) > 0 {
		d.Section("In Use By")
		for _, res := range inUseBy {
			// Extract resource name from ARN
			parts := strings.Split(res, "/")
			name := parts[len(parts)-1]
			d.Field("", name)
		}
	}

	// Domain Validation (for AMAZON_ISSUED certificates)
	if dvOptions := cert.DomainValidationOptions(); len(dvOptions) > 0 {
		d.Section("Domain Validation")
		for _, dv := range dvOptions {
			d.Field(appaws.Str(dv.DomainName), string(dv.ValidationStatus))
		}
	}

	// Renewal Summary (for AMAZON_ISSUED certificates)
	if renewal := cert.RenewalSummary(); renewal != nil {
		d.Section("Renewal Information")
		d.Field("Renewal Status", string(renewal.RenewalStatus))
		if renewal.UpdatedAt != nil {
			d.Field("Last Updated", renewal.UpdatedAt.Format("2006-01-02 15:04:05"))
		}
		if renewal.RenewalStatusReason != "" {
			d.Field("Status Reason", string(renewal.RenewalStatusReason))
		}
	}

	// Failure Information (for FAILED certificates)
	if failureReason := cert.FailureReason(); failureReason != "" {
		d.Section("Failure Information")
		d.Field("Failure Reason", failureReason)
	}

	// Revocation Information (for REVOKED certificates)
	if revokedAt := cert.RevokedAt(); revokedAt != "" {
		d.Section("Revocation Information")
		d.Field("Revoked At", revokedAt)
		if reason := cert.RevocationReason(); reason != "" {
			d.Field("Revocation Reason", reason)
		}
	}

	// Timestamps
	d.Section("Timestamps")
	if created := cert.CreatedAt(); created != "" {
		d.Field("Created", created)
	}
	if issued := cert.IssuedAt(); issued != "" {
		d.Field("Issued", issued)
	}
	if imported := cert.ImportedAt(); imported != "" {
		d.Field("Imported", imported)
	}

	return d.String()
}

// RenderSummary returns summary fields for the header panel
func (r *CertificateRenderer) RenderSummary(resource dao.Resource) []render.SummaryField {
	cert, ok := resource.(*CertificateResource)
	if !ok {
		return r.BaseRenderer.RenderSummary(resource)
	}

	fields := []render.SummaryField{
		{Label: "Domain", Value: cert.DomainName()},
		{Label: "ARN", Value: cert.GetARN()},
		{Label: "Status", Value: cert.Status()},
		{Label: "Type", Value: cert.Type()},
	}

	if notAfter := cert.NotAfter(); notAfter != "" {
		fields = append(fields, render.SummaryField{Label: "Expires", Value: notAfter})
	}

	if count := len(cert.InUseBy()); count > 0 {
		fields = append(fields, render.SummaryField{Label: "In Use By", Value: fmt.Sprintf("%d resources", count)})
	}

	return fields
}

// Navigations returns navigation shortcuts
func (r *CertificateRenderer) Navigations(resource dao.Resource) []render.Navigation {
	// No navigations for now - could add links to ELBs/CloudFront using this cert
	return nil
}
