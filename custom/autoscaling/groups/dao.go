package groups

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/autoscaling"
	"github.com/aws/aws-sdk-go-v2/service/autoscaling/types"

	appaws "github.com/clawscli/claws/internal/aws"
	"github.com/clawscli/claws/internal/dao"
	apperrors "github.com/clawscli/claws/internal/errors"
)

// AutoScalingGroupDAO provides data access for Auto Scaling Groups
type AutoScalingGroupDAO struct {
	dao.BaseDAO
	client *autoscaling.Client
}

// NewAutoScalingGroupDAO creates a new AutoScalingGroupDAO
func NewAutoScalingGroupDAO(ctx context.Context) (dao.DAO, error) {
	cfg, err := appaws.NewConfig(ctx)
	if err != nil {
		return nil, apperrors.Wrap(err, "new "+ServiceResourcePath+" dao")
	}
	return &AutoScalingGroupDAO{
		BaseDAO: dao.NewBaseDAO("autoscaling", "groups"),
		client:  autoscaling.NewFromConfig(cfg),
	}, nil
}

// List returns all Auto Scaling Groups
func (d *AutoScalingGroupDAO) List(ctx context.Context) ([]dao.Resource, error) {
	asgs, err := appaws.Paginate(ctx, func(token *string) ([]types.AutoScalingGroup, *string, error) {
		output, err := d.client.DescribeAutoScalingGroups(ctx, &autoscaling.DescribeAutoScalingGroupsInput{
			NextToken: token,
		})
		if err != nil {
			return nil, nil, apperrors.Wrap(err, "list auto scaling groups")
		}
		return output.AutoScalingGroups, output.NextToken, nil
	})
	if err != nil {
		return nil, err
	}

	resources := make([]dao.Resource, len(asgs))
	for i, asg := range asgs {
		resources[i] = NewAutoScalingGroupResource(asg)
	}

	return resources, nil
}

// Get returns a specific Auto Scaling Group
func (d *AutoScalingGroupDAO) Get(ctx context.Context, id string) (dao.Resource, error) {
	output, err := d.client.DescribeAutoScalingGroups(ctx, &autoscaling.DescribeAutoScalingGroupsInput{
		AutoScalingGroupNames: []string{id},
	})
	if err != nil {
		return nil, apperrors.Wrapf(err, "get auto scaling group %s", id)
	}

	if len(output.AutoScalingGroups) == 0 {
		return nil, fmt.Errorf("auto scaling group not found: %s", id)
	}

	return NewAutoScalingGroupResource(output.AutoScalingGroups[0]), nil
}

// Delete deletes an Auto Scaling Group
func (d *AutoScalingGroupDAO) Delete(ctx context.Context, id string) error {
	forceDelete := true
	_, err := d.client.DeleteAutoScalingGroup(ctx, &autoscaling.DeleteAutoScalingGroupInput{
		AutoScalingGroupName: &id,
		ForceDelete:          &forceDelete,
	})
	if err != nil {
		return apperrors.Wrapf(err, "delete auto scaling group %s", id)
	}
	return nil
}

// AutoScalingGroupResource wraps an Auto Scaling Group
type AutoScalingGroupResource struct {
	dao.BaseResource
	Item types.AutoScalingGroup
}

// NewAutoScalingGroupResource creates a new AutoScalingGroupResource
func NewAutoScalingGroupResource(asg types.AutoScalingGroup) *AutoScalingGroupResource {
	name := appaws.Str(asg.AutoScalingGroupName)

	// Extract tags
	tags := make(map[string]string)
	for _, tag := range asg.Tags {
		if tag.Key != nil && tag.Value != nil {
			tags[*tag.Key] = *tag.Value
		}
	}

	return &AutoScalingGroupResource{
		BaseResource: dao.BaseResource{
			ID:   name,
			Name: name,
			ARN:  appaws.Str(asg.AutoScalingGroupARN),
			Tags: tags,
			Data: asg,
		},
		Item: asg,
	}
}

// AutoScalingGroupName returns the ASG name
func (r *AutoScalingGroupResource) AutoScalingGroupName() string {
	if r.Item.AutoScalingGroupName != nil {
		return *r.Item.AutoScalingGroupName
	}
	return ""
}

// AutoScalingGroupARN returns the ASG ARN
func (r *AutoScalingGroupResource) AutoScalingGroupARN() string {
	if r.Item.AutoScalingGroupARN != nil {
		return *r.Item.AutoScalingGroupARN
	}
	return ""
}

// MinSize returns the minimum size
func (r *AutoScalingGroupResource) MinSize() int32 {
	if r.Item.MinSize != nil {
		return *r.Item.MinSize
	}
	return 0
}

// MaxSize returns the maximum size
func (r *AutoScalingGroupResource) MaxSize() int32 {
	if r.Item.MaxSize != nil {
		return *r.Item.MaxSize
	}
	return 0
}

// DesiredCapacity returns the desired capacity
func (r *AutoScalingGroupResource) DesiredCapacity() int32 {
	if r.Item.DesiredCapacity != nil {
		return *r.Item.DesiredCapacity
	}
	return 0
}

// InstanceCount returns the number of running instances
func (r *AutoScalingGroupResource) InstanceCount() int {
	return len(r.Item.Instances)
}

// HealthyInstanceCount returns the number of healthy instances
func (r *AutoScalingGroupResource) HealthyInstanceCount() int {
	count := 0
	for _, inst := range r.Item.Instances {
		if inst.HealthStatus != nil && *inst.HealthStatus == "Healthy" {
			count++
		}
	}
	return count
}

// HealthCheckType returns the health check type (EC2, ELB)
func (r *AutoScalingGroupResource) HealthCheckType() string {
	if r.Item.HealthCheckType != nil {
		return *r.Item.HealthCheckType
	}
	return ""
}

// HealthCheckGracePeriod returns the health check grace period
func (r *AutoScalingGroupResource) HealthCheckGracePeriod() int32 {
	if r.Item.HealthCheckGracePeriod != nil {
		return *r.Item.HealthCheckGracePeriod
	}
	return 0
}

// LaunchConfigurationName returns the launch configuration name
func (r *AutoScalingGroupResource) LaunchConfigurationName() string {
	if r.Item.LaunchConfigurationName != nil {
		return *r.Item.LaunchConfigurationName
	}
	return ""
}

// LaunchTemplateId returns the launch template ID
func (r *AutoScalingGroupResource) LaunchTemplateId() string {
	if r.Item.LaunchTemplate != nil && r.Item.LaunchTemplate.LaunchTemplateId != nil {
		return *r.Item.LaunchTemplate.LaunchTemplateId
	}
	return ""
}

// LaunchTemplateName returns the launch template name
func (r *AutoScalingGroupResource) LaunchTemplateName() string {
	if r.Item.LaunchTemplate != nil && r.Item.LaunchTemplate.LaunchTemplateName != nil {
		return *r.Item.LaunchTemplate.LaunchTemplateName
	}
	return ""
}

// LaunchTemplateVersion returns the launch template version
func (r *AutoScalingGroupResource) LaunchTemplateVersion() string {
	if r.Item.LaunchTemplate != nil && r.Item.LaunchTemplate.Version != nil {
		return *r.Item.LaunchTemplate.Version
	}
	return ""
}

// AvailabilityZones returns the availability zones
func (r *AutoScalingGroupResource) AvailabilityZones() []string {
	return r.Item.AvailabilityZones
}

// VPCZoneIdentifier returns the VPC subnet IDs
func (r *AutoScalingGroupResource) VPCZoneIdentifier() string {
	if r.Item.VPCZoneIdentifier != nil {
		return *r.Item.VPCZoneIdentifier
	}
	return ""
}

// TargetGroupARNs returns the target group ARNs
func (r *AutoScalingGroupResource) TargetGroupARNs() []string {
	return r.Item.TargetGroupARNs
}

// LoadBalancerNames returns the classic load balancer names
func (r *AutoScalingGroupResource) LoadBalancerNames() []string {
	return r.Item.LoadBalancerNames
}

// DefaultCooldown returns the default cooldown period
func (r *AutoScalingGroupResource) DefaultCooldown() int32 {
	if r.Item.DefaultCooldown != nil {
		return *r.Item.DefaultCooldown
	}
	return 0
}

// CreatedTime returns the creation time
func (r *AutoScalingGroupResource) CreatedTime() time.Time {
	if r.Item.CreatedTime != nil {
		return *r.Item.CreatedTime
	}
	return time.Time{}
}

// Status returns the ASG status
func (r *AutoScalingGroupResource) Status() string {
	if r.Item.Status != nil {
		return *r.Item.Status
	}
	return ""
}

// ServiceLinkedRoleARN returns the service-linked role ARN
func (r *AutoScalingGroupResource) ServiceLinkedRoleARN() string {
	if r.Item.ServiceLinkedRoleARN != nil {
		return *r.Item.ServiceLinkedRoleARN
	}
	return ""
}

// TerminationPolicies returns the termination policies
func (r *AutoScalingGroupResource) TerminationPolicies() []string {
	return r.Item.TerminationPolicies
}

// NewInstancesProtectedFromScaleIn returns whether new instances are protected
func (r *AutoScalingGroupResource) NewInstancesProtectedFromScaleIn() bool {
	if r.Item.NewInstancesProtectedFromScaleIn != nil {
		return *r.Item.NewInstancesProtectedFromScaleIn
	}
	return false
}
