package targets

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2"
	"github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2/types"

	appaws "github.com/clawscli/claws/internal/aws"
	"github.com/clawscli/claws/internal/dao"
)

// TargetDAO provides data access for ELBv2 Targets (Target Health)
type TargetDAO struct {
	dao.BaseDAO
	client *elasticloadbalancingv2.Client
}

// NewTargetDAO creates a new TargetDAO
func NewTargetDAO(ctx context.Context) (dao.DAO, error) {
	cfg, err := appaws.NewConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("new elbv2/targets dao: %w", err)
	}
	return &TargetDAO{
		BaseDAO: dao.NewBaseDAO("elbv2", "targets"),
		client:  elasticloadbalancingv2.NewFromConfig(cfg),
	}, nil
}

// List returns all targets for a target group (requires TargetGroupArn filter)
func (d *TargetDAO) List(ctx context.Context) ([]dao.Resource, error) {
	tgArn := dao.GetFilterFromContext(ctx, "TargetGroupArn")
	if tgArn == "" {
		return nil, fmt.Errorf("TargetGroupArn filter required - navigate from a target group")
	}

	output, err := d.client.DescribeTargetHealth(ctx, &elasticloadbalancingv2.DescribeTargetHealthInput{
		TargetGroupArn: &tgArn,
	})
	if err != nil {
		return nil, fmt.Errorf("describe target health: %w", err)
	}

	var resources []dao.Resource
	for _, th := range output.TargetHealthDescriptions {
		resources = append(resources, NewTargetResource(th, tgArn))
	}

	return resources, nil
}

// Get returns a specific target - not supported for targets
func (d *TargetDAO) Get(ctx context.Context, id string) (dao.Resource, error) {
	return nil, fmt.Errorf("get not supported for targets - use list from target group")
}

// Delete deregisters a target from the target group
func (d *TargetDAO) Delete(ctx context.Context, id string) error {
	return fmt.Errorf("delete not supported for targets - use deregister action")
}

// TargetResource wraps an ELBv2 Target Health Description
type TargetResource struct {
	dao.BaseResource
	Item           types.TargetHealthDescription
	TargetGroupArn string
}

// NewTargetResource creates a new TargetResource
func NewTargetResource(th types.TargetHealthDescription, tgArn string) *TargetResource {
	targetId := ""
	port := int32(0)
	if th.Target != nil {
		if th.Target.Id != nil {
			targetId = *th.Target.Id
		}
		if th.Target.Port != nil {
			port = *th.Target.Port
		}
	}

	// Create unique ID from target ID and port
	id := fmt.Sprintf("%s:%d", targetId, port)

	return &TargetResource{
		BaseResource: dao.BaseResource{
			ID:   id,
			Name: targetId,
			Tags: make(map[string]string),
			Data: th,
		},
		Item:           th,
		TargetGroupArn: tgArn,
	}
}

// TargetId returns the target ID (instance ID, IP, or Lambda ARN)
func (r *TargetResource) TargetId() string {
	if r.Item.Target != nil && r.Item.Target.Id != nil {
		return *r.Item.Target.Id
	}
	return ""
}

// Port returns the target port
func (r *TargetResource) Port() int32 {
	if r.Item.Target != nil && r.Item.Target.Port != nil {
		return *r.Item.Target.Port
	}
	return 0
}

// AvailabilityZone returns the availability zone
func (r *TargetResource) AvailabilityZone() string {
	if r.Item.Target != nil && r.Item.Target.AvailabilityZone != nil {
		return *r.Item.Target.AvailabilityZone
	}
	return ""
}

// HealthCheckPort returns the port used for health checks
func (r *TargetResource) HealthCheckPort() string {
	if r.Item.HealthCheckPort != nil {
		return *r.Item.HealthCheckPort
	}
	return ""
}

// HealthState returns the health state (healthy, unhealthy, draining, etc.)
func (r *TargetResource) HealthState() string {
	if r.Item.TargetHealth != nil {
		return string(r.Item.TargetHealth.State)
	}
	return ""
}

// HealthReason returns the reason for the health state
func (r *TargetResource) HealthReason() string {
	if r.Item.TargetHealth != nil {
		return string(r.Item.TargetHealth.Reason)
	}
	return ""
}

// HealthDescription returns the health state description
func (r *TargetResource) HealthDescription() string {
	if r.Item.TargetHealth != nil && r.Item.TargetHealth.Description != nil {
		return *r.Item.TargetHealth.Description
	}
	return ""
}
