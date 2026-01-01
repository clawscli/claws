package loadbalancers

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2"
	"github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2/types"

	appaws "github.com/clawscli/claws/internal/aws"
	"github.com/clawscli/claws/internal/dao"
	apperrors "github.com/clawscli/claws/internal/errors"
)

// LoadBalancerDAO provides data access for ELBv2 Load Balancers
type LoadBalancerDAO struct {
	dao.BaseDAO
	client *elasticloadbalancingv2.Client
}

// NewLoadBalancerDAO creates a new LoadBalancerDAO
func NewLoadBalancerDAO(ctx context.Context) (dao.DAO, error) {
	cfg, err := appaws.NewConfig(ctx)
	if err != nil {
		return nil, apperrors.Wrap(err, "new elbv2/loadbalancers dao")
	}
	return &LoadBalancerDAO{
		BaseDAO: dao.NewBaseDAO("elbv2", "load-balancers"),
		client:  elasticloadbalancingv2.NewFromConfig(cfg),
	}, nil
}

// List returns all load balancers (optionally filtered by ARN or name)
func (d *LoadBalancerDAO) List(ctx context.Context) ([]dao.Resource, error) {
	// Check for filters
	lbArn := dao.GetFilterFromContext(ctx, "LoadBalancerArn")
	lbName := dao.GetFilterFromContext(ctx, "LoadBalancerName")

	// If filtering by ARN or name, no pagination needed
	if lbArn != "" || lbName != "" {
		input := &elasticloadbalancingv2.DescribeLoadBalancersInput{}
		if lbArn != "" {
			input.LoadBalancerArns = []string{lbArn}
		}
		if lbName != "" {
			input.Names = []string{lbName}
		}

		output, err := d.client.DescribeLoadBalancers(ctx, input)
		if err != nil {
			return nil, apperrors.Wrap(err, "list load balancers")
		}

		resources := make([]dao.Resource, 0, len(output.LoadBalancers))
		for _, lb := range output.LoadBalancers {
			resources = append(resources, NewLoadBalancerResource(lb))
		}
		return resources, nil
	}

	// No filter - paginate through all
	loadBalancers, err := appaws.Paginate(ctx, func(token *string) ([]types.LoadBalancer, *string, error) {
		output, err := d.client.DescribeLoadBalancers(ctx, &elasticloadbalancingv2.DescribeLoadBalancersInput{
			Marker: token,
		})
		if err != nil {
			return nil, nil, apperrors.Wrap(err, "list load balancers")
		}
		return output.LoadBalancers, output.NextMarker, nil
	})
	if err != nil {
		return nil, err
	}

	resources := make([]dao.Resource, 0, len(loadBalancers))
	for _, lb := range loadBalancers {
		resources = append(resources, NewLoadBalancerResource(lb))
	}
	return resources, nil
}

// Get returns a specific load balancer
func (d *LoadBalancerDAO) Get(ctx context.Context, id string) (dao.Resource, error) {
	output, err := d.client.DescribeLoadBalancers(ctx, &elasticloadbalancingv2.DescribeLoadBalancersInput{
		LoadBalancerArns: []string{id},
	})
	if err != nil {
		return nil, apperrors.Wrapf(err, "get load balancer %s", id)
	}

	if len(output.LoadBalancers) == 0 {
		return nil, fmt.Errorf("load balancer not found: %s", id)
	}

	return NewLoadBalancerResource(output.LoadBalancers[0]), nil
}

// Delete deletes a load balancer
func (d *LoadBalancerDAO) Delete(ctx context.Context, id string) error {
	_, err := d.client.DeleteLoadBalancer(ctx, &elasticloadbalancingv2.DeleteLoadBalancerInput{
		LoadBalancerArn: &id,
	})
	if err != nil {
		if apperrors.IsNotFound(err) {
			return nil // Already deleted
		}
		if apperrors.IsResourceInUse(err) {
			return apperrors.Wrapf(err, "load balancer %s is in use", id)
		}
		return apperrors.Wrapf(err, "delete load balancer %s", id)
	}
	return nil
}

// LoadBalancerResource wraps an ELBv2 Load Balancer
type LoadBalancerResource struct {
	dao.BaseResource
	Item types.LoadBalancer
}

// NewLoadBalancerResource creates a new LoadBalancerResource
func NewLoadBalancerResource(lb types.LoadBalancer) *LoadBalancerResource {
	name := appaws.Str(lb.LoadBalancerName)
	arn := appaws.Str(lb.LoadBalancerArn)

	return &LoadBalancerResource{
		BaseResource: dao.BaseResource{
			ID:   arn,
			Name: name,
			ARN:  arn,
			Tags: make(map[string]string),
			Data: lb,
		},
		Item: lb,
	}
}

// LoadBalancerName returns the load balancer name
func (r *LoadBalancerResource) LoadBalancerName() string {
	if r.Item.LoadBalancerName != nil {
		return *r.Item.LoadBalancerName
	}
	return ""
}

// LoadBalancerArn returns the load balancer ARN
func (r *LoadBalancerResource) LoadBalancerArn() string {
	if r.Item.LoadBalancerArn != nil {
		return *r.Item.LoadBalancerArn
	}
	return ""
}

// Type returns the load balancer type (application, network, gateway)
func (r *LoadBalancerResource) Type() string {
	return string(r.Item.Type)
}

// Scheme returns the scheme (internet-facing, internal)
func (r *LoadBalancerResource) Scheme() string {
	return string(r.Item.Scheme)
}

// State returns the load balancer state
func (r *LoadBalancerResource) State() string {
	if r.Item.State != nil {
		return string(r.Item.State.Code)
	}
	return ""
}

// StateReason returns the state reason if any
func (r *LoadBalancerResource) StateReason() string {
	if r.Item.State != nil && r.Item.State.Reason != nil {
		return *r.Item.State.Reason
	}
	return ""
}

// DNSName returns the DNS name
func (r *LoadBalancerResource) DNSName() string {
	if r.Item.DNSName != nil {
		return *r.Item.DNSName
	}
	return ""
}

// VpcId returns the VPC ID
func (r *LoadBalancerResource) VpcId() string {
	if r.Item.VpcId != nil {
		return *r.Item.VpcId
	}
	return ""
}

// CreatedTime returns the creation time
func (r *LoadBalancerResource) CreatedTime() time.Time {
	if r.Item.CreatedTime != nil {
		return *r.Item.CreatedTime
	}
	return time.Time{}
}

// IpAddressType returns the IP address type
func (r *LoadBalancerResource) IpAddressType() string {
	return string(r.Item.IpAddressType)
}

// CanonicalHostedZoneId returns the Route53 hosted zone ID
func (r *LoadBalancerResource) CanonicalHostedZoneId() string {
	if r.Item.CanonicalHostedZoneId != nil {
		return *r.Item.CanonicalHostedZoneId
	}
	return ""
}

// AvailabilityZones returns the availability zones
func (r *LoadBalancerResource) AvailabilityZones() []string {
	var zones []string
	for _, az := range r.Item.AvailabilityZones {
		if az.ZoneName != nil {
			zones = append(zones, *az.ZoneName)
		}
	}
	return zones
}

// SecurityGroups returns the security group IDs
func (r *LoadBalancerResource) SecurityGroups() []string {
	return r.Item.SecurityGroups
}
