package internetgateways

import (
	appaws "github.com/clawscli/claws/internal/aws"
	"github.com/clawscli/claws/internal/config"
	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/render"
)

// Ensure InternetGatewayRenderer implements render.Navigator
var _ render.Navigator = (*InternetGatewayRenderer)(nil)

// InternetGatewayRenderer renders Internet Gateways
type InternetGatewayRenderer struct {
	render.BaseRenderer
}

// NewInternetGatewayRenderer creates a new InternetGatewayRenderer
func NewInternetGatewayRenderer() render.Renderer {
	return &InternetGatewayRenderer{
		BaseRenderer: render.BaseRenderer{
			Service:  "vpc",
			Resource: "internet-gateways",
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
					Name:  "IGW ID",
					Width: 24,
					Getter: func(r dao.Resource) string {
						return r.GetID()
					},
					Priority: 1,
				},
				{
					Name:  "STATE",
					Width: 12,
					Getter: func(r dao.Resource) string {
						if igwr, ok := r.(*InternetGatewayResource); ok {
							return igwr.AttachmentState()
						}
						return ""
					},
					Priority: 2,
				},
				{
					Name:  "VPC ID",
					Width: 22,
					Getter: func(r dao.Resource) string {
						if igwr, ok := r.(*InternetGatewayResource); ok {
							return igwr.AttachedVpcId()
						}
						return ""
					},
					Priority: 3,
				},
				{
					Name:  "OWNER",
					Width: 14,
					Getter: func(r dao.Resource) string {
						if igwr, ok := r.(*InternetGatewayResource); ok {
							return config.Global().MaskAccountID(igwr.OwnerId())
						}
						return ""
					},
					Priority: 4,
				},
			},
		},
	}
}

// RenderDetail renders detailed internet gateway information
func (r *InternetGatewayRenderer) RenderDetail(resource dao.Resource) string {
	igwr, ok := resource.(*InternetGatewayResource)
	if !ok {
		return ""
	}

	d := render.NewDetailBuilder()

	d.Title("Internet Gateway", igwr.GetName())

	// Basic Info
	d.Section("Basic Information")
	d.Field("Internet Gateway ID", igwr.GetID())
	d.FieldStyled("State", igwr.AttachmentState(), render.StateColorer()(igwr.AttachmentState()))
	if igwr.Item.OwnerId != nil {
		d.Field("Owner ID", config.Global().MaskAccountID(*igwr.Item.OwnerId))
	}

	// Attachments
	if len(igwr.Item.Attachments) > 0 {
		d.Section("VPC Attachments")
		for _, attach := range igwr.Item.Attachments {
			d.Field("VPC ID", appaws.Str(attach.VpcId))
			d.Field("State", string(attach.State))
		}
	} else {
		d.Section("VPC Attachments")
		d.Field("Status", "Not attached")
	}

	// Tags
	d.Tags(appaws.TagsToMap(igwr.Item.Tags))

	return d.String()
}

// RenderSummary returns summary fields for the header panel
func (r *InternetGatewayRenderer) RenderSummary(resource dao.Resource) []render.SummaryField {
	igwr, ok := resource.(*InternetGatewayResource)
	if !ok {
		return nil
	}

	stateStyle := render.StateColorer()(igwr.AttachmentState())

	fields := []render.SummaryField{
		{Label: "IGW ID", Value: igwr.GetID()},
		{Label: "Name", Value: igwr.GetName()},
		{Label: "State", Value: igwr.AttachmentState(), Style: stateStyle},
		{Label: "VPC ID", Value: igwr.AttachedVpcId()},
	}

	if igwr.Item.OwnerId != nil {
		fields = append(fields, render.SummaryField{Label: "Owner", Value: config.Global().MaskAccountID(*igwr.Item.OwnerId)})
	}

	return fields
}

// Navigations returns navigation shortcuts for Internet Gateway resources
func (r *InternetGatewayRenderer) Navigations(resource dao.Resource) []render.Navigation {
	igwr, ok := resource.(*InternetGatewayResource)
	if !ok {
		return nil
	}

	vpcId := igwr.AttachedVpcId()
	if vpcId == "" {
		return nil
	}

	return []render.Navigation{
		{Key: "v", Label: "VPC", Service: "vpc", Resource: "vpcs", FilterField: "VpcId", FilterValue: vpcId},
	}
}
