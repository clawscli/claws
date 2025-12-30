package services

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/ecs"
	"github.com/aws/aws-sdk-go-v2/service/ecs/types"

	appaws "github.com/clawscli/claws/internal/aws"
	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/log"
)

// ServiceDAO provides data access for ECS services
type ServiceDAO struct {
	dao.BaseDAO
	client *ecs.Client
}

// NewServiceDAO creates a new ServiceDAO
func NewServiceDAO(ctx context.Context) (dao.DAO, error) {
	cfg, err := appaws.NewConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("new ecs/services dao: %w", err)
	}
	return &ServiceDAO{
		BaseDAO: dao.NewBaseDAO("ecs", "services"),
		client:  ecs.NewFromConfig(cfg),
	}, nil
}

func (d *ServiceDAO) List(ctx context.Context) ([]dao.Resource, error) {
	// Get cluster name from filter context
	clusterName := dao.GetFilterFromContext(ctx, "ClusterName")
	if clusterName == "" {
		// List services from all clusters
		return d.listAllServices(ctx)
	}

	return d.listServicesInCluster(ctx, clusterName)
}

func (d *ServiceDAO) listAllServices(ctx context.Context) ([]dao.Resource, error) {
	// First get all clusters
	clusterArns, err := appaws.Paginate(ctx, func(token *string) ([]string, *string, error) {
		output, err := d.client.ListClusters(ctx, &ecs.ListClustersInput{NextToken: token})
		if err != nil {
			return nil, nil, fmt.Errorf("list clusters: %w", err)
		}
		return output.ClusterArns, output.NextToken, nil
	})
	if err != nil {
		return nil, err
	}

	resources := make([]dao.Resource, 0, len(clusterArns))
	for _, clusterArn := range clusterArns {
		clusterServices, err := d.listServicesInCluster(ctx, clusterArn)
		if err != nil {
			log.Warn("failed to list services in cluster", "cluster", clusterArn, "error", err)
			continue
		}
		resources = append(resources, clusterServices...)
	}

	return resources, nil
}

func (d *ServiceDAO) listServicesInCluster(ctx context.Context, cluster string) ([]dao.Resource, error) {
	serviceArns, err := appaws.Paginate(ctx, func(token *string) ([]string, *string, error) {
		output, err := d.client.ListServices(ctx, &ecs.ListServicesInput{
			Cluster:   &cluster,
			NextToken: token,
		})
		if err != nil {
			return nil, nil, fmt.Errorf("list services: %w", err)
		}
		return output.ServiceArns, output.NextToken, nil
	})
	if err != nil {
		return nil, err
	}

	if len(serviceArns) == 0 {
		return nil, nil
	}

	// Describe services in batches of 10 (API limit)
	resources := make([]dao.Resource, 0, len(serviceArns))
	for i := 0; i < len(serviceArns); i += 10 {
		end := i + 10
		if end > len(serviceArns) {
			end = len(serviceArns)
		}

		descInput := &ecs.DescribeServicesInput{
			Cluster:  &cluster,
			Services: serviceArns[i:end],
		}

		descOutput, err := d.client.DescribeServices(ctx, descInput)
		if err != nil {
			log.Warn("failed to describe services", "cluster", cluster, "error", err)
			continue
		}

		for _, svc := range descOutput.Services {
			resources = append(resources, NewServiceResource(svc))
		}
	}

	return resources, nil
}

func (d *ServiceDAO) Get(ctx context.Context, id string) (dao.Resource, error) {
	clusterName := dao.GetFilterFromContext(ctx, "ClusterName")
	if clusterName == "" {
		return nil, fmt.Errorf("cluster name filter required")
	}

	input := &ecs.DescribeServicesInput{
		Cluster:  &clusterName,
		Services: []string{id},
	}

	output, err := d.client.DescribeServices(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("describe service %s: %w", id, err)
	}

	if len(output.Services) == 0 {
		return nil, fmt.Errorf("service not found: %s", id)
	}

	return NewServiceResource(output.Services[0]), nil
}

func (d *ServiceDAO) Delete(ctx context.Context, id string) error {
	clusterName := dao.GetFilterFromContext(ctx, "ClusterName")
	if clusterName == "" {
		return fmt.Errorf("cluster name filter required")
	}

	force := true
	input := &ecs.DeleteServiceInput{
		Cluster: &clusterName,
		Service: &id,
		Force:   &force,
	}

	_, err := d.client.DeleteService(ctx, input)
	if err != nil {
		if appaws.IsNotFound(err) {
			return nil // Already deleted
		}
		return fmt.Errorf("delete service %s: %w", id, err)
	}

	return nil
}

// ServiceResource wraps an ECS service
type ServiceResource struct {
	dao.BaseResource
	Item types.Service
}

// NewServiceResource creates a new ServiceResource
func NewServiceResource(svc types.Service) *ServiceResource {
	return &ServiceResource{
		BaseResource: dao.BaseResource{
			ID:   appaws.Str(svc.ServiceName),
			Name: appaws.Str(svc.ServiceName),
			ARN:  appaws.Str(svc.ServiceArn),
			Data: svc,
		},
		Item: svc,
	}
}

// Status returns the service status
func (r *ServiceResource) Status() string {
	return appaws.Str(r.Item.Status)
}

// DesiredCount returns the desired task count
func (r *ServiceResource) DesiredCount() int32 {
	return r.Item.DesiredCount
}

// RunningCount returns the running task count
func (r *ServiceResource) RunningCount() int32 {
	return r.Item.RunningCount
}

// PendingCount returns the pending task count
func (r *ServiceResource) PendingCount() int32 {
	return r.Item.PendingCount
}

// LaunchType returns the launch type
func (r *ServiceResource) LaunchType() string {
	return string(r.Item.LaunchType)
}

// TaskDefinition returns the task definition ARN
func (r *ServiceResource) TaskDefinition() string {
	return appaws.Str(r.Item.TaskDefinition)
}

// ClusterArn returns the cluster ARN
func (r *ServiceResource) ClusterArn() string {
	return appaws.Str(r.Item.ClusterArn)
}

// Deployments returns the deployments
func (r *ServiceResource) Deployments() []types.Deployment {
	return r.Item.Deployments
}

// LoadBalancers returns the load balancers
func (r *ServiceResource) LoadBalancers() []types.LoadBalancer {
	return r.Item.LoadBalancers
}

// CreatedAt returns the service creation time
func (r *ServiceResource) CreatedAt() string {
	if r.Item.CreatedAt != nil {
		return r.Item.CreatedAt.Format("2006-01-02 15:04:05")
	}
	return ""
}

// CreatedBy returns the principal that created the service
func (r *ServiceResource) CreatedBy() string {
	return appaws.Str(r.Item.CreatedBy)
}

// EnableExecuteCommand returns whether ECS Exec is enabled
func (r *ServiceResource) EnableExecuteCommand() bool {
	return r.Item.EnableExecuteCommand
}

// SchedulingStrategy returns the scheduling strategy (REPLICA or DAEMON)
func (r *ServiceResource) SchedulingStrategy() string {
	return string(r.Item.SchedulingStrategy)
}

// NetworkConfiguration returns the network configuration
func (r *ServiceResource) NetworkConfiguration() *types.NetworkConfiguration {
	return r.Item.NetworkConfiguration
}

// ServiceRegistries returns the service discovery registries
func (r *ServiceResource) ServiceRegistries() []types.ServiceRegistry {
	return r.Item.ServiceRegistries
}

// Events returns the latest service events
func (r *ServiceResource) Events() []types.ServiceEvent {
	return r.Item.Events
}

// HealthCheckGracePeriodSeconds returns the health check grace period
func (r *ServiceResource) HealthCheckGracePeriodSeconds() int32 {
	if r.Item.HealthCheckGracePeriodSeconds != nil {
		return *r.Item.HealthCheckGracePeriodSeconds
	}
	return 0
}

// CapacityProviderStrategy returns the capacity provider strategy
func (r *ServiceResource) CapacityProviderStrategy() []types.CapacityProviderStrategyItem {
	return r.Item.CapacityProviderStrategy
}

// PlatformVersion returns the Fargate platform version
func (r *ServiceResource) PlatformVersion() string {
	return appaws.Str(r.Item.PlatformVersion)
}
