package virtualinterfaces

import (
	"fmt"

	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/render"
)

// VirtualInterfaceRenderer renders Direct Connect virtual interfaces.
type VirtualInterfaceRenderer struct {
	render.BaseRenderer
}

// NewVirtualInterfaceRenderer creates a new VirtualInterfaceRenderer.
func NewVirtualInterfaceRenderer() render.Renderer {
	return &VirtualInterfaceRenderer{
		BaseRenderer: render.BaseRenderer{
			Service:  "directconnect",
			Resource: "virtual-interfaces",
			Cols: []render.Column{
				{Name: "VI ID", Width: 20, Getter: func(r dao.Resource) string { return r.GetID() }},
				{Name: "NAME", Width: 25, Getter: getName},
				{Name: "TYPE", Width: 10, Getter: getType},
				{Name: "STATE", Width: 12, Getter: getState},
				{Name: "VLAN", Width: 8, Getter: getVlan},
				{Name: "LOCATION", Width: 15, Getter: getLocation},
			},
		},
	}
}

func getName(r dao.Resource) string {
	vi, ok := r.(*VirtualInterfaceResource)
	if !ok {
		return ""
	}
	return vi.VirtualInterfaceName()
}

func getType(r dao.Resource) string {
	vi, ok := r.(*VirtualInterfaceResource)
	if !ok {
		return ""
	}
	return vi.VirtualInterfaceType()
}

func getState(r dao.Resource) string {
	vi, ok := r.(*VirtualInterfaceResource)
	if !ok {
		return ""
	}
	return vi.VirtualInterfaceState()
}

func getVlan(r dao.Resource) string {
	vi, ok := r.(*VirtualInterfaceResource)
	if !ok {
		return ""
	}
	vlan := vi.Vlan()
	if vlan > 0 {
		return fmt.Sprintf("%d", vlan)
	}
	return ""
}

func getLocation(r dao.Resource) string {
	vi, ok := r.(*VirtualInterfaceResource)
	if !ok {
		return ""
	}
	return vi.Location()
}

// RenderDetail renders the detail view for a virtual interface.
func (r *VirtualInterfaceRenderer) RenderDetail(resource dao.Resource) string {
	vi, ok := resource.(*VirtualInterfaceResource)
	if !ok {
		return ""
	}

	d := render.NewDetailBuilder()

	title := vi.GetID()
	if name := vi.VirtualInterfaceName(); name != "" {
		title = name
	}
	d.Title("Direct Connect Virtual Interface", title)

	// Basic Info
	d.Section("Basic Information")
	d.Field("Virtual Interface ID", vi.GetID())
	if name := vi.VirtualInterfaceName(); name != "" {
		d.Field("Name", name)
	}
	d.Field("Type", vi.VirtualInterfaceType())
	d.Field("State", vi.VirtualInterfaceState())
	d.Field("Owner Account", vi.OwnerAccount())

	// Connection
	d.Section("Connection")
	d.Field("Connection ID", vi.ConnectionId())
	d.Field("Location", vi.Location())
	d.Field("Region", vi.Region())

	// Routing
	d.Section("Routing Configuration")
	if vlan := vi.Vlan(); vlan > 0 {
		d.Field("VLAN", fmt.Sprintf("%d", vlan))
	}
	if asn := vi.Asn(); asn > 0 {
		d.Field("Customer ASN", fmt.Sprintf("%d", asn))
	}
	if asn := vi.AmazonSideAsn(); asn > 0 {
		d.Field("Amazon Side ASN", fmt.Sprintf("%d", asn))
	}
	if addr := vi.AmazonAddress(); addr != "" {
		d.Field("Amazon Address", addr)
	}
	if addr := vi.CustomerAddress(); addr != "" {
		d.Field("Customer Address", addr)
	}

	// Gateway
	if vgw := vi.VirtualGatewayId(); vgw != "" {
		d.Section("Gateway")
		d.Field("Virtual Gateway ID", vgw)
	}
	if dxgw := vi.DirectConnectGatewayId(); dxgw != "" {
		if vi.VirtualGatewayId() == "" {
			d.Section("Gateway")
		}
		d.Field("Direct Connect Gateway ID", dxgw)
	}

	// Features
	d.Section("Features")
	if af := vi.AddressFamily(); af != "" {
		d.Field("Address Family", af)
	}
	if mtu := vi.Mtu(); mtu > 0 {
		d.Field("MTU", fmt.Sprintf("%d", mtu))
	}
	if vi.JumboFrameCapable() {
		d.Field("Jumbo Frame Capable", "Yes")
	}
	if vi.SiteLinkEnabled() {
		d.Field("SiteLink", "Enabled")
	}

	// BGP Peers
	if peers := vi.BgpPeers(); len(peers) > 0 {
		d.Section("BGP Peers")
		for i, peer := range peers {
			if i >= 5 {
				d.Field("", fmt.Sprintf("... and %d more", len(peers)-5))
				break
			}
			peerInfo := fmt.Sprintf("ASN: %d, State: %s", peer.Asn, peer.BgpPeerState)
			d.Field(fmt.Sprintf("Peer %d", i+1), peerInfo)
		}
	}

	// Route Prefixes
	if prefixes := vi.RouteFilterPrefixes(); len(prefixes) > 0 {
		d.Section("Route Filter Prefixes")
		for i, prefix := range prefixes {
			if i >= 10 {
				d.Field("", fmt.Sprintf("... and %d more", len(prefixes)-10))
				break
			}
			d.Field(fmt.Sprintf("Prefix %d", i+1), prefix)
		}
	}

	// Tags
	if tags := vi.Tags(); len(tags) > 0 {
		d.Section("Tags")
		for k, v := range tags {
			d.Field(k, v)
		}
	}

	return d.String()
}

// RenderSummary renders summary fields for a virtual interface.
func (r *VirtualInterfaceRenderer) RenderSummary(resource dao.Resource) []render.SummaryField {
	vi, ok := resource.(*VirtualInterfaceResource)
	if !ok {
		return r.BaseRenderer.RenderSummary(resource)
	}

	fields := []render.SummaryField{
		{Label: "Virtual Interface ID", Value: vi.GetID()},
		{Label: "Type", Value: vi.VirtualInterfaceType()},
		{Label: "State", Value: vi.VirtualInterfaceState()},
		{Label: "Location", Value: vi.Location()},
	}

	if name := vi.VirtualInterfaceName(); name != "" {
		fields = append([]render.SummaryField{{Label: "Name", Value: name}}, fields...)
	}

	if vlan := vi.Vlan(); vlan > 0 {
		fields = append(fields, render.SummaryField{Label: "VLAN", Value: fmt.Sprintf("%d", vlan)})
	}

	return fields
}
