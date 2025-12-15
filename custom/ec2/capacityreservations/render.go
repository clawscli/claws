package capacityreservations

import (
	"fmt"
	"time"

	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/render"
)

// CapacityReservationRenderer renders EC2 Capacity Reservations
type CapacityReservationRenderer struct {
	render.BaseRenderer
}

// NewCapacityReservationRenderer creates a new CapacityReservationRenderer
func NewCapacityReservationRenderer() render.Renderer {
	return &CapacityReservationRenderer{
		BaseRenderer: render.BaseRenderer{
			Service:  "ec2",
			Resource: "capacity-reservations",
			Cols: []render.Column{
				{
					Name:  "ID",
					Width: 24,
					Getter: func(r dao.Resource) string {
						return r.GetID()
					},
					Priority: 0,
				},
				{
					Name:  "STATE",
					Width: 10,
					Getter: func(r dao.Resource) string {
						if cr, ok := r.(*CapacityReservationResource); ok {
							return cr.State()
						}
						return ""
					},
					Priority: 1,
				},
				{
					Name:  "TYPE",
					Width: 13,
					Getter: func(r dao.Resource) string {
						if cr, ok := r.(*CapacityReservationResource); ok {
							return cr.InstanceType()
						}
						return ""
					},
					Priority: 2,
				},
				{
					Name:  "AZ",
					Width: 14,
					Getter: func(r dao.Resource) string {
						if cr, ok := r.(*CapacityReservationResource); ok {
							return cr.AvailabilityZone()
						}
						return ""
					},
					Priority: 3,
				},
				{
					Name:  "USED/TOTAL",
					Width: 11,
					Getter: func(r dao.Resource) string {
						if cr, ok := r.(*CapacityReservationResource); ok {
							return fmt.Sprintf("%d/%d", cr.UsedInstanceCount(), cr.TotalInstanceCount())
						}
						return ""
					},
					Priority: 4,
				},
				{
					Name:  "MATCH",
					Width: 10,
					Getter: func(r dao.Resource) string {
						if cr, ok := r.(*CapacityReservationResource); ok {
							return cr.InstanceMatchCriteria()
						}
						return ""
					},
					Priority: 5,
				},
				{
					Name:  "END TYPE",
					Width: 10,
					Getter: func(r dao.Resource) string {
						if cr, ok := r.(*CapacityReservationResource); ok {
							return cr.EndDateType()
						}
						return ""
					},
					Priority: 6,
				},
				{
					Name:  "AGE",
					Width: 8,
					Getter: func(r dao.Resource) string {
						if cr, ok := r.(*CapacityReservationResource); ok {
							if cr.CreateDate() != nil {
								return render.FormatAge(*cr.CreateDate())
							}
						}
						return ""
					},
					Priority: 7,
				},
			},
		},
	}
}

// RenderDetail renders detailed Capacity Reservation information
func (r *CapacityReservationRenderer) RenderDetail(resource dao.Resource) string {
	cr, ok := resource.(*CapacityReservationResource)
	if !ok {
		return ""
	}

	d := render.NewDetailBuilder()

	d.Title("Capacity Reservation", cr.GetID())

	// Basic Info
	d.Section("Basic Information")
	d.Field("Capacity Reservation ID", cr.GetID())
	d.Field("ARN", cr.ARN())
	d.FieldStyled("State", cr.State(), render.StateColorer()(cr.State()))
	d.Field("Owner ID", cr.OwnerID())

	// Instance Configuration
	d.Section("Instance Configuration")
	d.Field("Instance Type", cr.InstanceType())
	d.Field("Platform", cr.InstancePlatform())
	d.Field("Availability Zone", cr.AvailabilityZone())
	d.Field("Tenancy", cr.Tenancy())
	if cr.EbsOptimized() {
		d.Field("EBS Optimized", "Yes")
	}
	if cr.EphemeralStorage() {
		d.Field("Ephemeral Storage", "Yes")
	}

	// Capacity
	d.Section("Capacity")
	d.Field("Total Instances", fmt.Sprintf("%d", cr.TotalInstanceCount()))
	d.Field("Available Instances", fmt.Sprintf("%d", cr.AvailableInstanceCount()))
	d.Field("Used Instances", fmt.Sprintf("%d", cr.UsedInstanceCount()))
	utilization := float64(0)
	if cr.TotalInstanceCount() > 0 {
		utilization = float64(cr.UsedInstanceCount()) / float64(cr.TotalInstanceCount()) * 100
	}
	d.Field("Utilization", fmt.Sprintf("%.1f%%", utilization))

	// Matching
	d.Section("Matching")
	d.Field("Instance Match Criteria", cr.InstanceMatchCriteria())

	// Capacity Allocations
	if len(cr.Item.CapacityAllocations) > 0 {
		d.Section("Capacity Allocations")
		for _, alloc := range cr.Item.CapacityAllocations {
			allocType := string(alloc.AllocationType)
			count := int32(0)
			if alloc.Count != nil {
				count = *alloc.Count
			}
			d.Field(allocType, fmt.Sprintf("%d instances", count))
		}
	}

	// Duration
	d.Section("Duration")
	d.Field("End Date Type", cr.EndDateType())
	if start := cr.StartDate(); start != nil {
		d.Field("Start Date", start.Format(time.RFC3339))
	}
	if end := cr.EndDate(); end != nil {
		d.Field("End Date", end.Format(time.RFC3339))
	}
	if create := cr.CreateDate(); create != nil {
		d.Field("Created", create.Format(time.RFC3339))
		d.Field("Age", render.FormatAge(*create))
	}

	// Tags
	d.Tags(cr.GetTags())

	return d.String()
}

// RenderSummary returns summary fields for the header panel
func (r *CapacityReservationRenderer) RenderSummary(resource dao.Resource) []render.SummaryField {
	cr, ok := resource.(*CapacityReservationResource)
	if !ok {
		return nil
	}

	stateStyle := render.StateColorer()(cr.State())

	fields := []render.SummaryField{
		{Label: "ID", Value: cr.GetID()},
		{Label: "State", Value: cr.State(), Style: stateStyle},
		{Label: "Type", Value: cr.InstanceType()},
		{Label: "AZ", Value: cr.AvailabilityZone()},
		{Label: "Total", Value: fmt.Sprintf("%d", cr.TotalInstanceCount())},
		{Label: "Available", Value: fmt.Sprintf("%d", cr.AvailableInstanceCount())},
		{Label: "Match Criteria", Value: cr.InstanceMatchCriteria()},
		{Label: "End Type", Value: cr.EndDateType()},
	}

	return fields
}
