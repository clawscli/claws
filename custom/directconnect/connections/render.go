package connections

import (
	"fmt"

	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/render"
)

// ConnectionRenderer renders Direct Connect connections.
// Ensure ConnectionRenderer implements render.Navigator
var _ render.Navigator = (*ConnectionRenderer)(nil)

type ConnectionRenderer struct {
	render.BaseRenderer
}

// NewConnectionRenderer creates a new ConnectionRenderer.
func NewConnectionRenderer() render.Renderer {
	return &ConnectionRenderer{
		BaseRenderer: render.BaseRenderer{
			Service:  "directconnect",
			Resource: "connections",
			Cols: []render.Column{
				{Name: "CONNECTION ID", Width: 22, Getter: func(r dao.Resource) string { return r.GetID() }},
				{Name: "NAME", Width: 30, Getter: getName},
				{Name: "STATE", Width: 12, Getter: getState},
				{Name: "LOCATION", Width: 15, Getter: getLocation},
				{Name: "BANDWIDTH", Width: 12, Getter: getBandwidth},
				{Name: "VLAN", Width: 8, Getter: getVlan},
			},
		},
	}
}

func getName(r dao.Resource) string {
	conn, ok := r.(*ConnectionResource)
	if !ok {
		return ""
	}
	return conn.ConnectionName()
}

func getState(r dao.Resource) string {
	conn, ok := r.(*ConnectionResource)
	if !ok {
		return ""
	}
	return conn.ConnectionState()
}

func getLocation(r dao.Resource) string {
	conn, ok := r.(*ConnectionResource)
	if !ok {
		return ""
	}
	return conn.Location()
}

func getBandwidth(r dao.Resource) string {
	conn, ok := r.(*ConnectionResource)
	if !ok {
		return ""
	}
	return conn.Bandwidth()
}

func getVlan(r dao.Resource) string {
	conn, ok := r.(*ConnectionResource)
	if !ok {
		return ""
	}
	vlan := conn.Vlan()
	if vlan > 0 {
		return fmt.Sprintf("%d", vlan)
	}
	return ""
}

// RenderDetail renders the detail view for a Direct Connect connection.
func (r *ConnectionRenderer) RenderDetail(resource dao.Resource) string {
	conn, ok := resource.(*ConnectionResource)
	if !ok {
		return ""
	}

	d := render.NewDetailBuilder()

	title := conn.GetID()
	if name := conn.ConnectionName(); name != "" {
		title = name
	}
	d.Title("Direct Connect Connection", title)

	// Basic Info
	d.Section("Basic Information")
	d.Field("Connection ID", conn.GetID())
	if name := conn.ConnectionName(); name != "" {
		d.Field("Name", name)
	}
	d.Field("State", conn.ConnectionState())
	d.Field("Owner Account", conn.OwnerAccount())

	// Location & Bandwidth
	d.Section("Connection Details")
	d.Field("Location", conn.Location())
	d.Field("Region", conn.Region())
	d.Field("Bandwidth", conn.Bandwidth())
	if vlan := conn.Vlan(); vlan > 0 {
		d.Field("VLAN", fmt.Sprintf("%d", vlan))
	}
	if lagId := conn.LagId(); lagId != "" {
		d.Field("LAG ID", lagId)
	}

	// AWS Devices
	d.Section("AWS Devices")
	if devV2 := conn.AwsDeviceV2(); devV2 != "" {
		d.Field("AWS Device", devV2)
	}
	if logicalDev := conn.AwsLogicalDeviceId(); logicalDev != "" {
		d.Field("AWS Logical Device ID", logicalDev)
	}

	// Partner / Provider
	if partner := conn.PartnerName(); partner != "" {
		d.Section("Partner")
		d.Field("Partner Name", partner)
	}
	if provider := conn.ProviderName(); provider != "" {
		d.Field("Provider Name", provider)
	}

	// Features & Capabilities
	d.Section("Features & Capabilities")
	d.Field("Logical Redundancy", conn.HasLogicalRedundancy())
	if conn.JumboFrameCapable() {
		d.Field("Jumbo Frame", "Capable")
	}
	if conn.MacSecCapable() {
		d.Field("MACSec", "Capable")
	}
	if enc := conn.EncryptionMode(); enc != "" {
		d.Field("Encryption Mode", enc)
	}
	if status := conn.PortEncryptionStatus(); status != "" {
		d.Field("Port Encryption Status", status)
	}

	// Tags
	if tags := conn.Tags(); len(tags) > 0 {
		d.Section("Tags")
		for k, v := range tags {
			d.Field(k, v)
		}
	}

	return d.String()
}

// RenderSummary renders summary fields for a Direct Connect connection.
func (r *ConnectionRenderer) RenderSummary(resource dao.Resource) []render.SummaryField {
	conn, ok := resource.(*ConnectionResource)
	if !ok {
		return r.BaseRenderer.RenderSummary(resource)
	}

	fields := []render.SummaryField{
		{Label: "Connection ID", Value: conn.GetID()},
		{Label: "State", Value: conn.ConnectionState()},
		{Label: "Location", Value: conn.Location()},
		{Label: "Bandwidth", Value: conn.Bandwidth()},
	}

	if name := conn.ConnectionName(); name != "" {
		fields = append([]render.SummaryField{{Label: "Name", Value: name}}, fields...)
	}

	return fields
}

// Navigations returns available navigations from a Direct Connect connection.
func (r *ConnectionRenderer) Navigations(resource dao.Resource) []render.Navigation {
	conn, ok := resource.(*ConnectionResource)
	if !ok {
		return nil
	}
	return []render.Navigation{
		{
			Key:         "v",
			Label:       "Virtual Interfaces",
			Service:     "directconnect",
			Resource:    "virtual-interfaces",
			FilterField: "ConnectionId",
			FilterValue: conn.GetID(),
		},
	}
}
