package recordsets

import (
	"fmt"
	"strings"

	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/render"
)

// RecordSetRenderer renders Route53 record sets with custom columns
type RecordSetRenderer struct {
	render.BaseRenderer
}

// NewRecordSetRenderer creates a new RecordSetRenderer
func NewRecordSetRenderer() render.Renderer {
	return &RecordSetRenderer{
		BaseRenderer: render.BaseRenderer{
			Service:  "route53",
			Resource: "record-sets",
			Cols: []render.Column{
				{
					Name:  "NAME",
					Width: 40,
					Getter: func(r dao.Resource) string {
						if rr, ok := r.(*RecordSetResource); ok {
							return rr.RecordName()
						}
						return r.GetName()
					},
					Priority: 0,
				},
				{
					Name:  "TYPE",
					Width: 8,
					Getter: func(r dao.Resource) string {
						if rr, ok := r.(*RecordSetResource); ok {
							return rr.RecordType()
						}
						return ""
					},
					Priority: 1,
				},
				{
					Name:  "TTL",
					Width: 8,
					Getter: func(r dao.Resource) string {
						if rr, ok := r.(*RecordSetResource); ok {
							if rr.IsAlias() {
								return "Alias"
							}
							return fmt.Sprintf("%d", rr.TTL())
						}
						return ""
					},
					Priority: 2,
				},
				{
					Name:  "VALUE",
					Width: 50,
					Getter: func(r dao.Resource) string {
						if rr, ok := r.(*RecordSetResource); ok {
							if rr.IsAlias() {
								return rr.AliasTarget()
							}
							values := rr.Values()
							if len(values) > 0 {
								if len(values) == 1 {
									return values[0]
								}
								return fmt.Sprintf("%s (+%d more)", values[0], len(values)-1)
							}
						}
						return ""
					},
					Priority: 3,
				},
				{
					Name:  "ROUTING",
					Width: 12,
					Getter: func(r dao.Resource) string {
						if rr, ok := r.(*RecordSetResource); ok {
							if rr.SetIdentifier() != "" {
								if rr.Weight() > 0 {
									return "Weighted"
								}
								if rr.Region() != "" {
									return "Latency"
								}
								if rr.Failover() != "" {
									return "Failover"
								}
								return "Multi-value"
							}
							return "Simple"
						}
						return ""
					},
					Priority: 4,
				},
			},
		},
	}
}

// RenderDetail renders detailed record set information
func (r *RecordSetRenderer) RenderDetail(resource dao.Resource) string {
	rr, ok := resource.(*RecordSetResource)
	if !ok {
		return ""
	}

	d := render.NewDetailBuilder()

	d.Title("Route53 Record Set", rr.RecordName())

	// Basic Info
	d.Section("Basic Information")
	d.Field("Name", rr.RecordName())
	d.Field("Type", rr.RecordType())
	d.Field("Hosted Zone ID", rr.HostedZoneID)

	// Record Value
	if rr.IsAlias() {
		d.Section("Alias Target")
		d.Field("DNS Name", rr.AliasTarget())
		d.Field("Hosted Zone ID", rr.AliasHostedZoneID())
		if rr.Item.AliasTarget != nil {
			d.Field("Evaluate Target Health", fmt.Sprintf("%v", rr.Item.AliasTarget.EvaluateTargetHealth))
		}
	} else {
		d.Section("Record Values")
		d.Field("TTL", fmt.Sprintf("%d seconds", rr.TTL()))
		for i, v := range rr.Values() {
			d.Field(fmt.Sprintf("Value %d", i+1), v)
		}
	}

	// Routing Policy
	if rr.SetIdentifier() != "" {
		d.Section("Routing Policy")
		d.Field("Set Identifier", rr.SetIdentifier())

		if rr.Weight() > 0 {
			d.Field("Policy", "Weighted")
			d.Field("Weight", fmt.Sprintf("%d", rr.Weight()))
		} else if rr.Region() != "" {
			d.Field("Policy", "Latency")
			d.Field("Region", rr.Region())
		} else if rr.Failover() != "" {
			d.Field("Policy", "Failover")
			d.Field("Failover Type", rr.Failover())
		} else {
			d.Field("Policy", "Multi-value Answer")
		}
	}

	// Health Check
	if rr.HealthCheckID() != "" {
		d.Section("Health Check")
		d.Field("Health Check ID", rr.HealthCheckID())
	}

	return d.String()
}

// RenderSummary returns summary fields for the header panel
func (r *RecordSetRenderer) RenderSummary(resource dao.Resource) []render.SummaryField {
	rr, ok := resource.(*RecordSetResource)
	if !ok {
		return nil
	}

	fields := []render.SummaryField{
		{Label: "Name", Value: rr.RecordName()},
		{Label: "Type", Value: rr.RecordType()},
	}

	if rr.IsAlias() {
		fields = append(fields, render.SummaryField{Label: "Alias Target", Value: rr.AliasTarget()})
	} else {
		fields = append(fields, render.SummaryField{Label: "TTL", Value: fmt.Sprintf("%d", rr.TTL())})
		values := rr.Values()
		if len(values) > 0 {
			fields = append(fields, render.SummaryField{Label: "Values", Value: strings.Join(values, ", ")})
		}
	}

	if rr.SetIdentifier() != "" {
		fields = append(fields, render.SummaryField{Label: "Set ID", Value: rr.SetIdentifier()})
	}

	return fields
}
