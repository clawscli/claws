package loadbalancers

import (
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2/types"
)

func TestNewLoadBalancerResource(t *testing.T) {
	createdTime := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
	lb := types.LoadBalancer{
		LoadBalancerName:      aws.String("my-alb"),
		LoadBalancerArn:       aws.String("arn:aws:elasticloadbalancing:us-east-1:123456789012:loadbalancer/app/my-alb/1234567890123456"),
		DNSName:               aws.String("my-alb-123456.us-east-1.elb.amazonaws.com"),
		Type:                  types.LoadBalancerTypeEnumApplication,
		Scheme:                types.LoadBalancerSchemeEnumInternetFacing,
		VpcId:                 aws.String("vpc-12345678"),
		CreatedTime:           &createdTime,
		IpAddressType:         types.IpAddressTypeIpv4,
		CanonicalHostedZoneId: aws.String("Z35SXDOTRQ7X7K"),
		State: &types.LoadBalancerState{
			Code:   types.LoadBalancerStateEnumActive,
			Reason: nil,
		},
		AvailabilityZones: []types.AvailabilityZone{
			{ZoneName: aws.String("us-east-1a")},
			{ZoneName: aws.String("us-east-1b")},
		},
		SecurityGroups: []string{"sg-12345678", "sg-87654321"},
	}

	resource := NewLoadBalancerResource(lb)

	tests := []struct {
		name     string
		got      interface{}
		expected interface{}
	}{
		{"GetID", resource.GetID(), "arn:aws:elasticloadbalancing:us-east-1:123456789012:loadbalancer/app/my-alb/1234567890123456"},
		{"GetName", resource.GetName(), "my-alb"},
		{"LoadBalancerName", resource.LoadBalancerName(), "my-alb"},
		{"LoadBalancerArn", resource.LoadBalancerArn(), "arn:aws:elasticloadbalancing:us-east-1:123456789012:loadbalancer/app/my-alb/1234567890123456"},
		{"Type", resource.Type(), "application"},
		{"Scheme", resource.Scheme(), "internet-facing"},
		{"State", resource.State(), "active"},
		{"DNSName", resource.DNSName(), "my-alb-123456.us-east-1.elb.amazonaws.com"},
		{"VpcId", resource.VpcId(), "vpc-12345678"},
		{"IpAddressType", resource.IpAddressType(), "ipv4"},
		{"CanonicalHostedZoneId", resource.CanonicalHostedZoneId(), "Z35SXDOTRQ7X7K"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.expected {
				t.Errorf("%s = %v, want %v", tt.name, tt.got, tt.expected)
			}
		})
	}

	// Test CreatedTime
	if !resource.CreatedTime().Equal(createdTime) {
		t.Errorf("CreatedTime() = %v, want %v", resource.CreatedTime(), createdTime)
	}

	// Test AvailabilityZones
	zones := resource.AvailabilityZones()
	if len(zones) != 2 {
		t.Errorf("AvailabilityZones length = %d, want 2", len(zones))
	}
	if zones[0] != "us-east-1a" {
		t.Errorf("AvailabilityZones[0] = %q, want %q", zones[0], "us-east-1a")
	}

	// Test SecurityGroups
	sgs := resource.SecurityGroups()
	if len(sgs) != 2 {
		t.Errorf("SecurityGroups length = %d, want 2", len(sgs))
	}
}

func TestLoadBalancerResource_MinimalLB(t *testing.T) {
	lb := types.LoadBalancer{
		LoadBalancerName: aws.String("minimal-lb"),
	}

	resource := NewLoadBalancerResource(lb)

	tests := []struct {
		name     string
		got      interface{}
		expected interface{}
	}{
		{"GetName", resource.GetName(), "minimal-lb"},
		{"LoadBalancerArn", resource.LoadBalancerArn(), ""},
		{"Type", resource.Type(), ""},
		{"Scheme", resource.Scheme(), ""},
		{"State", resource.State(), ""},
		{"DNSName", resource.DNSName(), ""},
		{"VpcId", resource.VpcId(), ""},
		{"CanonicalHostedZoneId", resource.CanonicalHostedZoneId(), ""},
		{"StateReason", resource.StateReason(), ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.expected {
				t.Errorf("%s = %v, want %v", tt.name, tt.got, tt.expected)
			}
		})
	}

	// CreatedTime should be zero
	if !resource.CreatedTime().IsZero() {
		t.Errorf("CreatedTime() should be zero for nil creation time")
	}

	// Empty slices
	if len(resource.AvailabilityZones()) != 0 {
		t.Errorf("AvailabilityZones should be empty")
	}
	if len(resource.SecurityGroups()) != 0 {
		t.Errorf("SecurityGroups should be empty")
	}
}

func TestLoadBalancerResource_TypeVariations(t *testing.T) {
	lbTypes := []struct {
		lbType   types.LoadBalancerTypeEnum
		expected string
	}{
		{types.LoadBalancerTypeEnumApplication, "application"},
		{types.LoadBalancerTypeEnumNetwork, "network"},
		{types.LoadBalancerTypeEnumGateway, "gateway"},
	}

	for _, tc := range lbTypes {
		t.Run(string(tc.lbType), func(t *testing.T) {
			lb := types.LoadBalancer{
				LoadBalancerName: aws.String("test"),
				Type:             tc.lbType,
			}
			resource := NewLoadBalancerResource(lb)
			if got := resource.Type(); got != tc.expected {
				t.Errorf("Type() = %q, want %q", got, tc.expected)
			}
		})
	}
}

func TestLoadBalancerResource_SchemeVariations(t *testing.T) {
	schemes := []struct {
		scheme   types.LoadBalancerSchemeEnum
		expected string
	}{
		{types.LoadBalancerSchemeEnumInternetFacing, "internet-facing"},
		{types.LoadBalancerSchemeEnumInternal, "internal"},
	}

	for _, tc := range schemes {
		t.Run(string(tc.scheme), func(t *testing.T) {
			lb := types.LoadBalancer{
				LoadBalancerName: aws.String("test"),
				Scheme:           tc.scheme,
			}
			resource := NewLoadBalancerResource(lb)
			if got := resource.Scheme(); got != tc.expected {
				t.Errorf("Scheme() = %q, want %q", got, tc.expected)
			}
		})
	}
}

func TestLoadBalancerResource_StateVariations(t *testing.T) {
	states := []struct {
		name     string
		state    *types.LoadBalancerState
		expected string
		reason   string
	}{
		{"nil state", nil, "", ""},
		{"active", &types.LoadBalancerState{Code: types.LoadBalancerStateEnumActive}, "active", ""},
		{"provisioning", &types.LoadBalancerState{Code: types.LoadBalancerStateEnumProvisioning}, "provisioning", ""},
		{"active_impaired", &types.LoadBalancerState{Code: types.LoadBalancerStateEnumActiveImpaired}, "active_impaired", ""},
		{"failed", &types.LoadBalancerState{Code: types.LoadBalancerStateEnumFailed, Reason: aws.String("quota exceeded")}, "failed", "quota exceeded"},
	}

	for _, tc := range states {
		t.Run(tc.name, func(t *testing.T) {
			lb := types.LoadBalancer{
				LoadBalancerName: aws.String("test"),
				State:            tc.state,
			}
			resource := NewLoadBalancerResource(lb)
			if got := resource.State(); got != tc.expected {
				t.Errorf("State() = %q, want %q", got, tc.expected)
			}
			if got := resource.StateReason(); got != tc.reason {
				t.Errorf("StateReason() = %q, want %q", got, tc.reason)
			}
		})
	}
}
