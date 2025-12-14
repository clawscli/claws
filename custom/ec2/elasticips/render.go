package elasticips

import (
	appaws "github.com/clawscli/claws/internal/aws"
	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/render"
)

// ElasticIPRenderer renders Elastic IPs
type ElasticIPRenderer struct {
	render.BaseRenderer
}

// NewElasticIPRenderer creates a new ElasticIPRenderer
func NewElasticIPRenderer() render.Renderer {
	return &ElasticIPRenderer{
		BaseRenderer: render.BaseRenderer{
			Service:  "ec2",
			Resource: "elastic-ips",
			Cols: []render.Column{
				{
					Name:  "NAME",
					Width: 25,
					Getter: func(r dao.Resource) string {
						return r.GetName()
					},
					Priority: 0,
				},
				{
					Name:  "ALLOCATION ID",
					Width: 26,
					Getter: func(r dao.Resource) string {
						return r.GetID()
					},
					Priority: 1,
				},
				{
					Name:  "PUBLIC IP",
					Width: 16,
					Getter: func(r dao.Resource) string {
						if v, ok := r.(*ElasticIPResource); ok {
							return v.PublicIP()
						}
						return ""
					},
					Priority: 2,
				},
				{
					Name:  "INSTANCE",
					Width: 20,
					Getter: func(r dao.Resource) string {
						if v, ok := r.(*ElasticIPResource); ok {
							return v.InstanceId()
						}
						return ""
					},
					Priority: 3,
				},
				{
					Name:  "PRIVATE IP",
					Width: 16,
					Getter: func(r dao.Resource) string {
						if v, ok := r.(*ElasticIPResource); ok {
							return v.PrivateIP()
						}
						return ""
					},
					Priority: 4,
				},
				{
					Name:  "DOMAIN",
					Width: 8,
					Getter: func(r dao.Resource) string {
						if v, ok := r.(*ElasticIPResource); ok {
							return v.Domain()
						}
						return ""
					},
					Priority: 5,
				},
				{
					Name:  "ENI",
					Width: 22,
					Getter: func(r dao.Resource) string {
						if v, ok := r.(*ElasticIPResource); ok {
							return v.NetworkInterfaceId()
						}
						return ""
					},
					Priority: 6,
				},
				render.TagsColumn(25, 7),
			},
		},
	}
}

// RenderDetail renders detailed Elastic IP information
func (r *ElasticIPRenderer) RenderDetail(resource dao.Resource) string {
	v, ok := resource.(*ElasticIPResource)
	if !ok {
		return ""
	}

	d := render.NewDetailBuilder()

	d.Title("Elastic IP", v.GetName())

	// Basic Info
	d.Section("Basic Information")
	d.Field("Allocation ID", v.GetID())
	d.Field("Public IP", v.PublicIP())
	d.Field("Domain", v.Domain())

	// Association
	d.Section("Association")
	if assocId := v.AssociationId(); assocId != "" {
		d.Field("Association ID", assocId)
		d.Field("Instance ID", v.InstanceId())
		d.Field("Private IP", v.PrivateIP())
		d.Field("Network Interface", v.NetworkInterfaceId())
		d.Field("ENI Owner", v.NetworkInterfaceOwnerId())
	} else {
		d.DimIndent("(not associated)")
	}

	// Public IP Pool
	if v.Item.PublicIpv4Pool != nil {
		d.Section("IP Pool")
		d.Field("IPv4 Pool", *v.Item.PublicIpv4Pool)
	}

	// Carrier IP (for Wavelength)
	if v.Item.CarrierIp != nil {
		d.Section("Carrier")
		d.Field("Carrier IP", *v.Item.CarrierIp)
	}

	// Customer Owned IP
	if v.Item.CustomerOwnedIp != nil {
		d.Section("Customer Owned")
		d.Field("Customer Owned IP", *v.Item.CustomerOwnedIp)
		d.FieldIf("Customer Owned Pool", v.Item.CustomerOwnedIpv4Pool)
	}

	// Tags
	d.Tags(appaws.TagsToMap(v.Item.Tags))

	return d.String()
}

// RenderSummary returns summary fields for the header panel
func (r *ElasticIPRenderer) RenderSummary(resource dao.Resource) []render.SummaryField {
	v, ok := resource.(*ElasticIPResource)
	if !ok {
		return nil
	}

	fields := []render.SummaryField{
		{Label: "Allocation ID", Value: v.GetID()},
		{Label: "Public IP", Value: v.PublicIP()},
		{Label: "Domain", Value: v.Domain()},
	}

	if name := v.GetName(); name != "" {
		fields = append(fields, render.SummaryField{Label: "Name", Value: name})
	}

	if instanceId := v.InstanceId(); instanceId != "" {
		fields = append(fields, render.SummaryField{Label: "Instance", Value: instanceId})
		fields = append(fields, render.SummaryField{Label: "Private IP", Value: v.PrivateIP()})
	}

	if eni := v.NetworkInterfaceId(); eni != "" {
		fields = append(fields, render.SummaryField{Label: "ENI", Value: eni})
	}

	return fields
}
