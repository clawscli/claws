package distributions

import (
	"fmt"
	"strings"

	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/render"
	"github.com/clawscli/claws/internal/ui"
)

// DistributionRenderer renders CloudFront distributions
// Ensure DistributionRenderer implements render.Navigator
var _ render.Navigator = (*DistributionRenderer)(nil)

type DistributionRenderer struct {
	render.BaseRenderer
}

// NewDistributionRenderer creates a new DistributionRenderer
func NewDistributionRenderer() *DistributionRenderer {
	return &DistributionRenderer{
		BaseRenderer: render.BaseRenderer{
			Service:  "cloudfront",
			Resource: "distributions",
			Cols: []render.Column{
				{Name: "ID", Width: 16, Getter: getDistributionId},
				{Name: "DOMAIN", Width: 32, Getter: getDomainName},
				{Name: "STATUS", Width: 12, Getter: getStatus},
				{Name: "ORIGIN", Width: 25, Getter: getDefaultOrigin},
				{Name: "ALIASES", Width: 8, Getter: getAliasCount},
				{Name: "AGE", Width: 10, Getter: getAge},
			},
		},
	}
}

func getDistributionId(r dao.Resource) string {
	if dist, ok := r.(*DistributionResource); ok {
		return dist.DistributionId()
	}
	return ""
}

func getDomainName(r dao.Resource) string {
	if dist, ok := r.(*DistributionResource); ok {
		return dist.DomainName()
	}
	return ""
}

func getStatus(r dao.Resource) string {
	if dist, ok := r.(*DistributionResource); ok {
		status := dist.Status()
		if !dist.Enabled() {
			return status + " (Off)"
		}
		return status
	}
	return ""
}

func getDefaultOrigin(r dao.Resource) string {
	if dist, ok := r.(*DistributionResource); ok {
		origin := dist.DefaultOrigin()
		if len(origin) > 23 {
			return origin[:23] + "..."
		}
		return origin
	}
	return ""
}

func getAliasCount(r dao.Resource) string {
	if dist, ok := r.(*DistributionResource); ok {
		count := dist.AliasCount()
		if count > 0 {
			return fmt.Sprintf("%d", count)
		}
		return "-"
	}
	return ""
}

func getAge(r dao.Resource) string {
	if dist, ok := r.(*DistributionResource); ok {
		if dist.Item.LastModifiedTime != nil {
			return render.FormatAge(*dist.Item.LastModifiedTime)
		}
	}
	return "-"
}

// RenderDetail renders detailed distribution information
func (r *DistributionRenderer) RenderDetail(resource dao.Resource) string {
	dist, ok := resource.(*DistributionResource)
	if !ok {
		return ""
	}

	d := render.NewDetailBuilder()

	d.Title("CloudFront Distribution", dist.DistributionId())

	// Basic Info
	d.Section("Basic Information")
	d.Field("Distribution ID", dist.DistributionId())
	d.Field("ARN", dist.GetARN())
	d.Field("Domain Name", dist.DomainName())
	d.Field("Status", dist.Status())
	d.Field("Enabled", formatBool(dist.Enabled()))
	if comment := dist.Comment(); comment != "" {
		d.Field("Comment", comment)
	}

	// Aliases
	if aliases := dist.Aliases(); len(aliases) > 0 {
		d.Section("Alternate Domain Names (CNAMEs)")
		d.Field("Aliases", strings.Join(aliases, ", "))
	}

	// Origins
	d.Section("Origins")
	origins := dist.Origins()
	d.Field("Origin Count", fmt.Sprintf("%d", dist.OriginCount()))
	if len(origins) > 0 {
		for i, origin := range origins {
			d.Field(fmt.Sprintf("Origin %d", i+1), origin)
		}
	}

	// Cache Behavior
	d.Section("Cache Behavior")
	d.Field("Viewer Protocol Policy", dist.DefaultCacheBehaviorViewerProtocolPolicy())
	if dist.CacheBehaviorCount > 0 {
		d.Field("Additional Cache Behaviors", fmt.Sprintf("%d", dist.CacheBehaviorCount))
	}
	if dist.CustomErrorResponses > 0 {
		d.Field("Custom Error Responses", fmt.Sprintf("%d", dist.CustomErrorResponses))
	}
	if dist.DefaultRootObject != "" {
		d.Field("Default Root Object", dist.DefaultRootObject)
	}

	// SSL/TLS Certificate
	if dist.ViewerCertificate != nil {
		vc := dist.ViewerCertificate
		d.Section("SSL/TLS Certificate")
		if vc.CloudFrontDefaultCertificate != nil && *vc.CloudFrontDefaultCertificate {
			d.Field("Certificate", "CloudFront Default (*.cloudfront.net)")
		} else if vc.ACMCertificateArn != nil {
			d.Field("ACM Certificate", *vc.ACMCertificateArn)
		} else if vc.IAMCertificateId != nil {
			d.Field("IAM Certificate", *vc.IAMCertificateId)
		}
		if vc.MinimumProtocolVersion != "" {
			d.Field("Minimum Protocol Version", string(vc.MinimumProtocolVersion))
		}
		if vc.SSLSupportMethod != "" {
			d.Field("SSL Support Method", string(vc.SSLSupportMethod))
		}
	}

	// Settings
	d.Section("Settings")
	d.Field("Price Class", dist.PriceClass())
	d.Field("HTTP Version", dist.HttpVersion())
	if dist.IsIPV6Enabled {
		d.FieldStyled("IPv6", "Enabled", ui.SuccessStyle())
	} else {
		d.Field("IPv6", "Disabled")
	}

	// Access Logging
	if dist.Logging != nil && dist.Logging.Enabled != nil && *dist.Logging.Enabled {
		d.Section("Access Logging")
		d.FieldStyled("Status", "Enabled", ui.SuccessStyle())
		if dist.Logging.Bucket != nil {
			d.Field("Bucket", *dist.Logging.Bucket)
		}
		if dist.Logging.Prefix != nil && *dist.Logging.Prefix != "" {
			d.Field("Prefix", *dist.Logging.Prefix)
		}
	}

	// Geo Restriction
	if dist.GeoRestriction != nil && dist.GeoRestriction.RestrictionType != "none" {
		d.Section("Geo Restriction")
		d.Field("Type", string(dist.GeoRestriction.RestrictionType))
		if dist.GeoRestriction.Quantity != nil && *dist.GeoRestriction.Quantity > 0 {
			d.Field("Locations", strings.Join(dist.GeoRestriction.Items, ", "))
		}
	}

	// Security
	if webAclId := dist.WebACLId(); webAclId != "" {
		d.Section("Security")
		d.Field("WAF Web ACL", webAclId)
	}

	// Status
	if batches := dist.InProgressInvalidationBatches(); batches > 0 {
		d.Section("In Progress")
		d.Field("Invalidation Batches", fmt.Sprintf("%d", batches))
	}

	// Timestamps
	d.Section("Timestamps")
	if lastMod := dist.LastModifiedTime(); lastMod != "" {
		d.Field("Last Modified", lastMod)
	}

	return d.String()
}

func formatBool(b bool) string {
	if b {
		return "Yes"
	}
	return "No"
}

// RenderSummary returns summary fields for the header panel
func (r *DistributionRenderer) RenderSummary(resource dao.Resource) []render.SummaryField {
	dist, ok := resource.(*DistributionResource)
	if !ok {
		return r.BaseRenderer.RenderSummary(resource)
	}

	fields := []render.SummaryField{
		{Label: "Distribution ID", Value: dist.DistributionId()},
		{Label: "ARN", Value: dist.GetARN()},
		{Label: "Domain Name", Value: dist.DomainName()},
		{Label: "Status", Value: dist.Status()},
		{Label: "Enabled", Value: formatBool(dist.Enabled())},
	}

	if aliases := dist.Aliases(); len(aliases) > 0 {
		fields = append(fields, render.SummaryField{
			Label: "Aliases",
			Value: strings.Join(aliases, ", "),
		})
	}

	fields = append(fields,
		render.SummaryField{Label: "Origins", Value: fmt.Sprintf("%d", dist.OriginCount())},
		render.SummaryField{Label: "Price Class", Value: dist.PriceClass()},
	)

	if lastMod := dist.LastModifiedTime(); lastMod != "" {
		fields = append(fields, render.SummaryField{Label: "Last Modified", Value: lastMod})
	}

	return fields
}

// Navigations returns navigation shortcuts
func (r *DistributionRenderer) Navigations(resource dao.Resource) []render.Navigation {
	// No navigations for now
	return nil
}
