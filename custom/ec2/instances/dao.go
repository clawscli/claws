package instances

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/aws-sdk-go-v2/service/iam"

	appaws "github.com/clawscli/claws/internal/aws"
	"github.com/clawscli/claws/internal/dao"
	apperrors "github.com/clawscli/claws/internal/errors"
)

// InstanceDAO provides data access for EC2 instances
type InstanceDAO struct {
	dao.BaseDAO
	client    *ec2.Client
	iamClient *iam.Client
}

// NewInstanceDAO creates a new InstanceDAO
func NewInstanceDAO(ctx context.Context) (dao.DAO, error) {
	cfg, err := appaws.NewConfig(ctx)
	if err != nil {
		return nil, apperrors.Wrap(err, "new "+ServiceResourcePath+" dao")
	}
	return &InstanceDAO{
		BaseDAO:   dao.NewBaseDAO("ec2", "instances"),
		client:    ec2.NewFromConfig(cfg),
		iamClient: iam.NewFromConfig(cfg),
	}, nil
}

func (d *InstanceDAO) List(ctx context.Context) ([]dao.Resource, error) {
	input := &ec2.DescribeInstancesInput{}
	paginator := ec2.NewDescribeInstancesPaginator(d.client, input)

	// Cache for instance profile -> role name mapping
	roleCache := make(map[string]string)

	var resources []dao.Resource
	for paginator.HasMorePages() {
		output, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, apperrors.Wrap(err, "describe instances")
		}

		for _, reservation := range output.Reservations {
			for _, instance := range reservation.Instances {
				roleName := d.getRoleNameFromInstance(ctx, instance, roleCache)
				resources = append(resources, NewInstanceResourceWithRole(instance, roleName))
			}
		}
	}

	return resources, nil
}

func (d *InstanceDAO) Get(ctx context.Context, id string) (dao.Resource, error) {
	input := &ec2.DescribeInstancesInput{
		InstanceIds: []string{id},
	}

	output, err := d.client.DescribeInstances(ctx, input)
	if err != nil {
		return nil, apperrors.Wrapf(err, "describe instance %s", id)
	}

	if len(output.Reservations) == 0 || len(output.Reservations[0].Instances) == 0 {
		return nil, fmt.Errorf("instance not found: %s", id)
	}

	instance := output.Reservations[0].Instances[0]
	roleName := d.getRoleNameFromInstance(ctx, instance, nil)
	return NewInstanceResourceWithRole(instance, roleName), nil
}

func (d *InstanceDAO) Delete(ctx context.Context, id string) error {
	input := &ec2.TerminateInstancesInput{
		InstanceIds: []string{id},
	}

	_, err := d.client.TerminateInstances(ctx, input)
	if err != nil {
		if apperrors.IsNotFound(err) {
			return nil // Already terminated
		}
		return apperrors.Wrapf(err, "terminate instance %s", id)
	}

	return nil
}

// getRoleNameFromInstance extracts the IAM role name from an instance's instance profile
func (d *InstanceDAO) getRoleNameFromInstance(ctx context.Context, instance types.Instance, cache map[string]string) string {
	if instance.IamInstanceProfile == nil || instance.IamInstanceProfile.Arn == nil {
		return ""
	}

	profileArn := *instance.IamInstanceProfile.Arn

	// Check cache first
	if cache != nil {
		if roleName, ok := cache[profileArn]; ok {
			return roleName
		}
	}

	// Extract instance profile name from ARN
	profileName := appaws.ExtractResourceName(profileArn)
	if profileName == "" {
		return ""
	}

	// Get instance profile to find the role
	output, err := d.iamClient.GetInstanceProfile(ctx, &iam.GetInstanceProfileInput{
		InstanceProfileName: &profileName,
	})
	if err != nil {
		return ""
	}

	roleName := ""
	if output.InstanceProfile != nil && len(output.InstanceProfile.Roles) > 0 {
		if output.InstanceProfile.Roles[0].RoleName != nil {
			roleName = *output.InstanceProfile.Roles[0].RoleName
		}
	}

	// Cache the result
	if cache != nil {
		cache[profileArn] = roleName
	}

	return roleName
}

// InstanceResource wraps an EC2 instance
type InstanceResource struct {
	dao.BaseResource
	Item     types.Instance
	RoleName string
}

// NewInstanceResourceWithRole creates a new InstanceResource with IAM role name
func NewInstanceResourceWithRole(instance types.Instance, roleName string) *InstanceResource {
	name := appaws.EC2NameTag(instance.Tags)

	return &InstanceResource{
		BaseResource: dao.BaseResource{
			ID:   appaws.Str(instance.InstanceId),
			Name: name,
			Tags: appaws.TagsToMap(instance.Tags),
			Data: instance,
		},
		Item:     instance,
		RoleName: roleName,
	}
}

// GetRoleName returns the IAM role name
func (r *InstanceResource) GetRoleName() string {
	return r.RoleName
}

// State returns the instance state
func (r *InstanceResource) State() string {
	if r.Item.State != nil && r.Item.State.Name != "" {
		return string(r.Item.State.Name)
	}
	return "unknown"
}

// InstanceType returns the instance type
func (r *InstanceResource) InstanceType() string {
	return string(r.Item.InstanceType)
}

// PrivateIP returns the private IP address
func (r *InstanceResource) PrivateIP() string {
	if r.Item.PrivateIpAddress != nil {
		return *r.Item.PrivateIpAddress
	}
	return ""
}

// PublicIP returns the public IP address
func (r *InstanceResource) PublicIP() string {
	if r.Item.PublicIpAddress != nil {
		return *r.Item.PublicIpAddress
	}
	return ""
}

// AZ returns the availability zone
func (r *InstanceResource) AZ() string {
	if r.Item.Placement != nil && r.Item.Placement.AvailabilityZone != nil {
		return *r.Item.Placement.AvailabilityZone
	}
	return ""
}

// EbsOptimized returns whether EBS optimization is enabled
func (r *InstanceResource) EbsOptimized() bool {
	if r.Item.EbsOptimized != nil {
		return *r.Item.EbsOptimized
	}
	return false
}

// SourceDestCheck returns whether source/destination check is enabled
func (r *InstanceResource) SourceDestCheck() bool {
	if r.Item.SourceDestCheck != nil {
		return *r.Item.SourceDestCheck
	}
	return true // default is enabled
}

// StateReason returns the reason for the current state
func (r *InstanceResource) StateReason() string {
	if r.Item.StateReason != nil && r.Item.StateReason.Message != nil {
		return *r.Item.StateReason.Message
	}
	return ""
}

// StateReasonCode returns the state reason code
func (r *InstanceResource) StateReasonCode() string {
	if r.Item.StateReason != nil && r.Item.StateReason.Code != nil {
		return *r.Item.StateReason.Code
	}
	return ""
}

// InstanceLifecycle returns the lifecycle (spot or empty for on-demand)
func (r *InstanceResource) InstanceLifecycle() string {
	return string(r.Item.InstanceLifecycle)
}

// CpuCoreCount returns the number of CPU cores
func (r *InstanceResource) CpuCoreCount() int32 {
	if r.Item.CpuOptions != nil && r.Item.CpuOptions.CoreCount != nil {
		return *r.Item.CpuOptions.CoreCount
	}
	return 0
}

// CpuThreadsPerCore returns the threads per core
func (r *InstanceResource) CpuThreadsPerCore() int32 {
	if r.Item.CpuOptions != nil && r.Item.CpuOptions.ThreadsPerCore != nil {
		return *r.Item.CpuOptions.ThreadsPerCore
	}
	return 0
}

// HibernationEnabled returns whether hibernation is enabled
func (r *InstanceResource) HibernationEnabled() bool {
	if r.Item.HibernationOptions != nil && r.Item.HibernationOptions.Configured != nil {
		return *r.Item.HibernationOptions.Configured
	}
	return false
}

// MonitoringState returns the monitoring state
func (r *InstanceResource) MonitoringState() string {
	if r.Item.Monitoring != nil {
		return string(r.Item.Monitoring.State)
	}
	return ""
}

// Tenancy returns the instance tenancy
func (r *InstanceResource) Tenancy() string {
	if r.Item.Placement != nil {
		return string(r.Item.Placement.Tenancy)
	}
	return ""
}

// RootDeviceType returns the root device type (ebs or instance-store)
func (r *InstanceResource) RootDeviceType() string {
	return string(r.Item.RootDeviceType)
}

// RootDeviceName returns the root device name
func (r *InstanceResource) RootDeviceName() string {
	return appaws.Str(r.Item.RootDeviceName)
}

// VirtualizationType returns the virtualization type (hvm or paravirtual)
func (r *InstanceResource) VirtualizationType() string {
	return string(r.Item.VirtualizationType)
}

// Hypervisor returns the hypervisor type
func (r *InstanceResource) Hypervisor() string {
	return string(r.Item.Hypervisor)
}

// MetadataHttpTokens returns the IMDS token requirement
func (r *InstanceResource) MetadataHttpTokens() string {
	if r.Item.MetadataOptions != nil {
		return string(r.Item.MetadataOptions.HttpTokens)
	}
	return ""
}

// MetadataHttpEndpoint returns the IMDS endpoint state
func (r *InstanceResource) MetadataHttpEndpoint() string {
	if r.Item.MetadataOptions != nil {
		return string(r.Item.MetadataOptions.HttpEndpoint)
	}
	return ""
}

// EnclaveEnabled returns whether Nitro Enclave is enabled
func (r *InstanceResource) EnclaveEnabled() bool {
	if r.Item.EnclaveOptions != nil && r.Item.EnclaveOptions.Enabled != nil {
		return *r.Item.EnclaveOptions.Enabled
	}
	return false
}
