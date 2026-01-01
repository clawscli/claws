package hostedzones

import (
	"fmt"
	"strings"

	appaws "github.com/clawscli/claws/internal/aws"
	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/render"
)

// Ensure HostedZoneRenderer implements render.Navigator
var _ render.Navigator = (*HostedZoneRenderer)(nil)

// HostedZoneRenderer renders Route53 hosted zones with custom columns
type HostedZoneRenderer struct {
	render.BaseRenderer
}

// NewHostedZoneRenderer creates a new HostedZoneRenderer
func NewHostedZoneRenderer() render.Renderer {
	return &HostedZoneRenderer{
		BaseRenderer: render.BaseRenderer{
			Service:  "route53",
			Resource: "hosted-zones",
			Cols: []render.Column{
				{
					Name:  "DOMAIN",
					Width: 40,
					Getter: func(r dao.Resource) string {
						if hr, ok := r.(*HostedZoneResource); ok {
							return hr.DomainName()
						}
						return r.GetName()
					},
					Priority: 0,
				},
				{
					Name:  "TYPE",
					Width: 8,
					Getter: func(r dao.Resource) string {
						if hr, ok := r.(*HostedZoneResource); ok {
							if hr.IsPrivate() {
								return "Private"
							}
							return "Public"
						}
						return ""
					},
					Priority: 1,
				},
				{
					Name:  "RECORDS",
					Width: 8,
					Getter: func(r dao.Resource) string {
						if hr, ok := r.(*HostedZoneResource); ok {
							return fmt.Sprintf("%d", hr.RecordSetCount)
						}
						return ""
					},
					Priority: 2,
				},
				{
					Name:  "ZONE ID",
					Width: 24,
					Getter: func(r dao.Resource) string {
						if hr, ok := r.(*HostedZoneResource); ok {
							return hr.ZoneID()
						}
						return ""
					},
					Priority: 3,
				},
				{
					Name:  "COMMENT",
					Width: 30,
					Getter: func(r dao.Resource) string {
						if hr, ok := r.(*HostedZoneResource); ok {
							return hr.Comment()
						}
						return ""
					},
					Priority: 4,
				},
			},
		},
	}
}

// RenderDetail renders detailed hosted zone information
func (r *HostedZoneRenderer) RenderDetail(resource dao.Resource) string {
	hr, ok := resource.(*HostedZoneResource)
	if !ok {
		return ""
	}

	d := render.NewDetailBuilder()

	d.Title("Route53 Hosted Zone", hr.DomainName())

	// Basic Info
	d.Section("Basic Information")
	d.Field("Domain Name", hr.DomainName())
	d.Field("Zone ID", hr.ZoneID())

	zoneType := "Public"
	if hr.IsPrivate() {
		zoneType = "Private"
	}
	d.Field("Type", zoneType)

	if hr.Comment() != "" {
		d.Field("Comment", hr.Comment())
	}

	d.Field("Record Set Count", fmt.Sprintf("%d", hr.RecordSetCount))

	// Name Servers (for public zones)
	if len(hr.NameServers()) > 0 {
		d.Section("Name Servers")
		for i, ns := range hr.NameServers() {
			d.Field(fmt.Sprintf("NS %d", i+1), ns)
		}
	}

	// Associated VPCs (for private zones)
	if len(hr.VPCs) > 0 {
		d.Section("Associated VPCs")
		for i, vpc := range hr.VPCs {
			vpcID := appaws.Str(vpc.VPCId)
			region := string(vpc.VPCRegion)
			d.Field(fmt.Sprintf("VPC %d", i+1), fmt.Sprintf("%s (%s)", vpcID, region))
		}
	}

	// Caller Reference
	if hr.CallerReference() != "" {
		d.Section("Metadata")
		d.Field("Caller Reference", hr.CallerReference())
	}

	return d.String()
}

// RenderSummary returns summary fields for the header panel
func (r *HostedZoneRenderer) RenderSummary(resource dao.Resource) []render.SummaryField {
	hr, ok := resource.(*HostedZoneResource)
	if !ok {
		return nil
	}

	zoneType := "Public"
	if hr.IsPrivate() {
		zoneType = "Private"
	}

	fields := []render.SummaryField{
		{Label: "Domain", Value: hr.DomainName()},
		{Label: "Zone ID", Value: hr.ZoneID()},
		{Label: "Type", Value: zoneType},
		{Label: "Records", Value: fmt.Sprintf("%d", hr.RecordSetCount)},
	}

	if hr.Comment() != "" {
		fields = append(fields, render.SummaryField{Label: "Comment", Value: hr.Comment()})
	}

	if len(hr.NameServers()) > 0 {
		fields = append(fields, render.SummaryField{
			Label: "Name Servers",
			Value: strings.Join(hr.NameServers(), ", "),
		})
	}

	return fields
}

// Navigations returns navigation shortcuts for hosted zones
func (r *HostedZoneRenderer) Navigations(resource dao.Resource) []render.Navigation {
	hr, ok := resource.(*HostedZoneResource)
	if !ok {
		return nil
	}

	var navs []render.Navigation

	// Record sets navigation
	navs = append(navs, render.Navigation{
		Key: "r", Label: "Record Sets", Service: "route53", Resource: "record-sets",
		FilterField: "HostedZoneId", FilterValue: hr.ZoneID(),
	})

	return navs
}
