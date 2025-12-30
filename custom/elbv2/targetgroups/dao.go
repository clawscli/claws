package targetgroups

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2"
	"github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2/types"

	appaws "github.com/clawscli/claws/internal/aws"
	"github.com/clawscli/claws/internal/dao"
	apperrors "github.com/clawscli/claws/internal/errors"
)

// TargetGroupDAO provides data access for ELBv2 Target Groups
type TargetGroupDAO struct {
	dao.BaseDAO
	client *elasticloadbalancingv2.Client
}

// NewTargetGroupDAO creates a new TargetGroupDAO
func NewTargetGroupDAO(ctx context.Context) (dao.DAO, error) {
	cfg, err := appaws.NewConfig(ctx)
	if err != nil {
		return nil, apperrors.Wrap(err, "new elbv2/targetgroups dao")
	}
	return &TargetGroupDAO{
		BaseDAO: dao.NewBaseDAO("elbv2", "target-groups"),
		client:  elasticloadbalancingv2.NewFromConfig(cfg),
	}, nil
}

// List returns all target groups (optionally filtered by load balancer ARN or target group ARN)
func (d *TargetGroupDAO) List(ctx context.Context) ([]dao.Resource, error) {
	// Check for filters
	lbArn := dao.GetFilterFromContext(ctx, "LoadBalancerArn")
	tgArn := dao.GetFilterFromContext(ctx, "TargetGroupArn")

	// If filtering by target group ARN, no pagination needed
	if tgArn != "" {
		output, err := d.client.DescribeTargetGroups(ctx, &elasticloadbalancingv2.DescribeTargetGroupsInput{
			TargetGroupArns: []string{tgArn},
		})
		if err != nil {
			return nil, apperrors.Wrap(err, "list target groups")
		}
		resources := make([]dao.Resource, 0, len(output.TargetGroups))
		for _, tg := range output.TargetGroups {
			resources = append(resources, NewTargetGroupResource(tg))
		}
		return resources, nil
	}

	// Paginate through results
	targetGroups, err := appaws.Paginate(ctx, func(token *string) ([]types.TargetGroup, *string, error) {
		input := &elasticloadbalancingv2.DescribeTargetGroupsInput{
			Marker: token,
		}
		if lbArn != "" {
			input.LoadBalancerArn = &lbArn
		}
		output, err := d.client.DescribeTargetGroups(ctx, input)
		if err != nil {
			return nil, nil, apperrors.Wrap(err, "list target groups")
		}
		return output.TargetGroups, output.NextMarker, nil
	})
	if err != nil {
		return nil, err
	}

	resources := make([]dao.Resource, 0, len(targetGroups))
	for _, tg := range targetGroups {
		resources = append(resources, NewTargetGroupResource(tg))
	}
	return resources, nil
}

// Get returns a specific target group
func (d *TargetGroupDAO) Get(ctx context.Context, id string) (dao.Resource, error) {
	output, err := d.client.DescribeTargetGroups(ctx, &elasticloadbalancingv2.DescribeTargetGroupsInput{
		TargetGroupArns: []string{id},
	})
	if err != nil {
		return nil, apperrors.Wrapf(err, "get target group %s", id)
	}

	if len(output.TargetGroups) == 0 {
		return nil, fmt.Errorf("target group not found: %s", id)
	}

	return NewTargetGroupResource(output.TargetGroups[0]), nil
}

// Delete deletes a target group
func (d *TargetGroupDAO) Delete(ctx context.Context, id string) error {
	_, err := d.client.DeleteTargetGroup(ctx, &elasticloadbalancingv2.DeleteTargetGroupInput{
		TargetGroupArn: &id,
	})
	if err != nil {
		if apperrors.IsNotFound(err) {
			return nil // Already deleted
		}
		if apperrors.IsResourceInUse(err) {
			return apperrors.Wrapf(err, "target group %s is in use (attached to load balancer)", id)
		}
		return apperrors.Wrapf(err, "delete target group %s", id)
	}
	return nil
}

// TargetGroupResource wraps an ELBv2 Target Group
type TargetGroupResource struct {
	dao.BaseResource
	Item types.TargetGroup
}

// NewTargetGroupResource creates a new TargetGroupResource
func NewTargetGroupResource(tg types.TargetGroup) *TargetGroupResource {
	name := appaws.Str(tg.TargetGroupName)
	arn := appaws.Str(tg.TargetGroupArn)

	return &TargetGroupResource{
		BaseResource: dao.BaseResource{
			ID:   arn,
			Name: name,
			ARN:  arn,
			Tags: make(map[string]string),
			Data: tg,
		},
		Item: tg,
	}
}

// TargetGroupName returns the target group name
func (r *TargetGroupResource) TargetGroupName() string {
	if r.Item.TargetGroupName != nil {
		return *r.Item.TargetGroupName
	}
	return ""
}

// TargetGroupArn returns the target group ARN
func (r *TargetGroupResource) TargetGroupArn() string {
	if r.Item.TargetGroupArn != nil {
		return *r.Item.TargetGroupArn
	}
	return ""
}

// Protocol returns the protocol
func (r *TargetGroupResource) Protocol() string {
	return string(r.Item.Protocol)
}

// Port returns the port
func (r *TargetGroupResource) Port() int32 {
	if r.Item.Port != nil {
		return *r.Item.Port
	}
	return 0
}

// ProtocolPort returns protocol:port string
func (r *TargetGroupResource) ProtocolPort() string {
	if r.Item.Port != nil {
		return fmt.Sprintf("%s:%d", r.Protocol(), *r.Item.Port)
	}
	return r.Protocol()
}

// TargetType returns the target type (instance, ip, lambda, alb)
func (r *TargetGroupResource) TargetType() string {
	return string(r.Item.TargetType)
}

// VpcId returns the VPC ID
func (r *TargetGroupResource) VpcId() string {
	if r.Item.VpcId != nil {
		return *r.Item.VpcId
	}
	return ""
}

// HealthCheckEnabled returns whether health checks are enabled
func (r *TargetGroupResource) HealthCheckEnabled() bool {
	if r.Item.HealthCheckEnabled != nil {
		return *r.Item.HealthCheckEnabled
	}
	return false
}

// HealthCheckProtocol returns the health check protocol
func (r *TargetGroupResource) HealthCheckProtocol() string {
	return string(r.Item.HealthCheckProtocol)
}

// HealthCheckPort returns the health check port
func (r *TargetGroupResource) HealthCheckPort() string {
	if r.Item.HealthCheckPort != nil {
		return *r.Item.HealthCheckPort
	}
	return ""
}

// HealthCheckPath returns the health check path
func (r *TargetGroupResource) HealthCheckPath() string {
	if r.Item.HealthCheckPath != nil {
		return *r.Item.HealthCheckPath
	}
	return ""
}

// HealthCheckIntervalSeconds returns the health check interval
func (r *TargetGroupResource) HealthCheckIntervalSeconds() int32 {
	if r.Item.HealthCheckIntervalSeconds != nil {
		return *r.Item.HealthCheckIntervalSeconds
	}
	return 0
}

// HealthCheckTimeoutSeconds returns the health check timeout
func (r *TargetGroupResource) HealthCheckTimeoutSeconds() int32 {
	if r.Item.HealthCheckTimeoutSeconds != nil {
		return *r.Item.HealthCheckTimeoutSeconds
	}
	return 0
}

// HealthyThresholdCount returns the healthy threshold
func (r *TargetGroupResource) HealthyThresholdCount() int32 {
	if r.Item.HealthyThresholdCount != nil {
		return *r.Item.HealthyThresholdCount
	}
	return 0
}

// UnhealthyThresholdCount returns the unhealthy threshold
func (r *TargetGroupResource) UnhealthyThresholdCount() int32 {
	if r.Item.UnhealthyThresholdCount != nil {
		return *r.Item.UnhealthyThresholdCount
	}
	return 0
}

// Matcher returns the health check matcher (HTTP codes)
func (r *TargetGroupResource) Matcher() string {
	if r.Item.Matcher != nil && r.Item.Matcher.HttpCode != nil {
		return *r.Item.Matcher.HttpCode
	}
	return ""
}

// LoadBalancerArns returns the associated load balancer ARNs
func (r *TargetGroupResource) LoadBalancerArns() []string {
	return r.Item.LoadBalancerArns
}

// IpAddressType returns the IP address type
func (r *TargetGroupResource) IpAddressType() string {
	return string(r.Item.IpAddressType)
}

// ProtocolVersion returns the protocol version (HTTP1, HTTP2, GRPC)
func (r *TargetGroupResource) ProtocolVersion() string {
	if r.Item.ProtocolVersion != nil {
		return *r.Item.ProtocolVersion
	}
	return ""
}
