package transitgateways

import (
	"fmt"

	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/render"
)

// TransitGatewayRenderer renders Transit Gateways.
// Ensure TransitGatewayRenderer implements render.Navigator
var _ render.Navigator = (*TransitGatewayRenderer)(nil)

type TransitGatewayRenderer struct {
	render.BaseRenderer
}

// NewTransitGatewayRenderer creates a new TransitGatewayRenderer.
func NewTransitGatewayRenderer() render.Renderer {
	return &TransitGatewayRenderer{
		BaseRenderer: render.BaseRenderer{
			Service:  "vpc",
			Resource: "transit-gateways",
			Cols: []render.Column{
				{Name: "TGW ID", Width: 24, Getter: func(r dao.Resource) string { return r.GetID() }},
				{Name: "NAME", Width: 30, Getter: getName},
				{Name: "STATE", Width: 12, Getter: getState},
				{Name: "OWNER", Width: 14, Getter: getOwner},
				{Name: "ASN", Width: 12, Getter: getAsn},
				{Name: "CREATED", Width: 20, Getter: getCreated},
			},
		},
	}
}

func getName(r dao.Resource) string {
	tgw, ok := r.(*TransitGatewayResource)
	if !ok {
		return ""
	}
	return tgw.Name()
}

func getState(r dao.Resource) string {
	tgw, ok := r.(*TransitGatewayResource)
	if !ok {
		return ""
	}
	return tgw.State()
}

func getOwner(r dao.Resource) string {
	tgw, ok := r.(*TransitGatewayResource)
	if !ok {
		return ""
	}
	return tgw.OwnerId()
}

func getAsn(r dao.Resource) string {
	tgw, ok := r.(*TransitGatewayResource)
	if !ok {
		return ""
	}
	asn := tgw.AmazonSideAsn()
	if asn > 0 {
		return fmt.Sprintf("%d", asn)
	}
	return ""
}

func getCreated(r dao.Resource) string {
	tgw, ok := r.(*TransitGatewayResource)
	if !ok {
		return ""
	}
	if t := tgw.CreationTime(); t != nil {
		return t.Format("2006-01-02 15:04")
	}
	return ""
}

// RenderDetail renders the detail view for a Transit Gateway.
func (r *TransitGatewayRenderer) RenderDetail(resource dao.Resource) string {
	tgw, ok := resource.(*TransitGatewayResource)
	if !ok {
		return ""
	}

	d := render.NewDetailBuilder()

	title := tgw.GetID()
	if name := tgw.Name(); name != "" {
		title = name
	}
	d.Title("Transit Gateway", title)

	// Basic Info
	d.Section("Basic Information")
	d.Field("Transit Gateway ID", tgw.GetID())
	if name := tgw.Name(); name != "" {
		d.Field("Name", name)
	}
	d.Field("ARN", tgw.GetARN())
	d.Field("State", tgw.State())
	d.Field("Owner ID", tgw.OwnerId())
	if desc := tgw.Description(); desc != "" {
		d.Field("Description", desc)
	}

	// Configuration
	d.Section("Configuration")
	if asn := tgw.AmazonSideAsn(); asn > 0 {
		d.Field("Amazon Side ASN", fmt.Sprintf("%d", asn))
	}
	if dns := tgw.DnsSupport(); dns != "" {
		d.Field("DNS Support", dns)
	}
	if vpnEcmp := tgw.VpnEcmpSupport(); vpnEcmp != "" {
		d.Field("VPN ECMP Support", vpnEcmp)
	}
	if multicast := tgw.MulticastSupport(); multicast != "" {
		d.Field("Multicast Support", multicast)
	}
	if autoAccept := tgw.AutoAcceptSharedAttachments(); autoAccept != "" {
		d.Field("Auto Accept Shared Attachments", autoAccept)
	}

	// Route Tables
	d.Section("Route Tables")
	if rt := tgw.DefaultRouteTableId(); rt != "" {
		d.Field("Association Default Route Table", rt)
	}
	if rt := tgw.PropagationDefaultRouteTableId(); rt != "" {
		d.Field("Propagation Default Route Table", rt)
	}
	if assoc := tgw.DefaultRouteTableAssociation(); assoc != "" {
		d.Field("Default Route Table Association", assoc)
	}
	if prop := tgw.DefaultRouteTablePropagation(); prop != "" {
		d.Field("Default Route Table Propagation", prop)
	}

	// Tags
	if tags := tgw.Tags(); len(tags) > 0 {
		d.Section("Tags")
		for k, v := range tags {
			if k != "Name" {
				d.Field(k, v)
			}
		}
	}

	// Timestamps
	if t := tgw.CreationTime(); t != nil {
		d.Section("Timestamps")
		d.Field("Created", t.Format("2006-01-02 15:04:05"))
	}

	return d.String()
}

// RenderSummary renders summary fields for a Transit Gateway.
func (r *TransitGatewayRenderer) RenderSummary(resource dao.Resource) []render.SummaryField {
	tgw, ok := resource.(*TransitGatewayResource)
	if !ok {
		return r.BaseRenderer.RenderSummary(resource)
	}

	fields := []render.SummaryField{
		{Label: "Transit Gateway ID", Value: tgw.GetID()},
		{Label: "State", Value: tgw.State()},
		{Label: "Owner", Value: tgw.OwnerId()},
	}

	if name := tgw.Name(); name != "" {
		fields = append([]render.SummaryField{{Label: "Name", Value: name}}, fields...)
	}

	if asn := tgw.AmazonSideAsn(); asn > 0 {
		fields = append(fields, render.SummaryField{Label: "ASN", Value: fmt.Sprintf("%d", asn)})
	}

	return fields
}

// Navigations returns available navigations from a Transit Gateway.
func (r *TransitGatewayRenderer) Navigations(resource dao.Resource) []render.Navigation {
	tgw, ok := resource.(*TransitGatewayResource)
	if !ok {
		return nil
	}
	return []render.Navigation{
		{
			Key:         "a",
			Label:       "Attachments",
			Service:     "vpc",
			Resource:    "tgw-attachments",
			FilterField: "TransitGatewayId",
			FilterValue: tgw.GetID(),
		},
	}
}
