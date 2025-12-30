package clusters

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/ecs"
	"github.com/aws/aws-sdk-go-v2/service/ecs/types"

	appaws "github.com/clawscli/claws/internal/aws"
	"github.com/clawscli/claws/internal/dao"
)

// ClusterDAO provides data access for ECS clusters
type ClusterDAO struct {
	dao.BaseDAO
	client *ecs.Client
}

// NewClusterDAO creates a new ClusterDAO
func NewClusterDAO(ctx context.Context) (dao.DAO, error) {
	cfg, err := appaws.NewConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("new ecs/clusters dao: %w", err)
	}
	return &ClusterDAO{
		BaseDAO: dao.NewBaseDAO("ecs", "clusters"),
		client:  ecs.NewFromConfig(cfg),
	}, nil
}

func (d *ClusterDAO) List(ctx context.Context) ([]dao.Resource, error) {
	// First, list all cluster ARNs
	clusterArns, err := appaws.Paginate(ctx, func(token *string) ([]string, *string, error) {
		output, err := d.client.ListClusters(ctx, &ecs.ListClustersInput{
			NextToken: token,
		})
		if err != nil {
			return nil, nil, fmt.Errorf("list clusters: %w", err)
		}
		return output.ClusterArns, output.NextToken, nil
	})
	if err != nil {
		return nil, err
	}

	if len(clusterArns) == 0 {
		return nil, nil
	}

	// Describe clusters to get details
	descInput := &ecs.DescribeClustersInput{
		Clusters: clusterArns,
		Include:  []types.ClusterField{types.ClusterFieldStatistics, types.ClusterFieldSettings},
	}

	descOutput, err := d.client.DescribeClusters(ctx, descInput)
	if err != nil {
		return nil, fmt.Errorf("describe clusters: %w", err)
	}

	resources := make([]dao.Resource, 0, len(descOutput.Clusters))
	for _, cluster := range descOutput.Clusters {
		resources = append(resources, NewClusterResource(cluster))
	}

	return resources, nil
}

func (d *ClusterDAO) Get(ctx context.Context, id string) (dao.Resource, error) {
	input := &ecs.DescribeClustersInput{
		Clusters: []string{id},
		Include:  []types.ClusterField{types.ClusterFieldStatistics, types.ClusterFieldSettings},
	}

	output, err := d.client.DescribeClusters(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("describe cluster %s: %w", id, err)
	}

	if len(output.Clusters) == 0 {
		return nil, fmt.Errorf("cluster not found: %s", id)
	}

	return NewClusterResource(output.Clusters[0]), nil
}

func (d *ClusterDAO) Delete(ctx context.Context, id string) error {
	input := &ecs.DeleteClusterInput{
		Cluster: &id,
	}

	_, err := d.client.DeleteCluster(ctx, input)
	if err != nil {
		if appaws.IsNotFound(err) {
			return nil // Already deleted
		}
		if appaws.IsResourceInUse(err) {
			return fmt.Errorf("cluster %s has active services or tasks", id)
		}
		return fmt.Errorf("delete cluster %s: %w", id, err)
	}

	return nil
}

// ClusterResource wraps an ECS cluster
type ClusterResource struct {
	dao.BaseResource
	Item types.Cluster
}

// NewClusterResource creates a new ClusterResource
func NewClusterResource(cluster types.Cluster) *ClusterResource {
	return &ClusterResource{
		BaseResource: dao.BaseResource{
			ID:   appaws.Str(cluster.ClusterName),
			Name: appaws.Str(cluster.ClusterName),
			ARN:  appaws.Str(cluster.ClusterArn),
			Data: cluster,
		},
		Item: cluster,
	}
}

// Status returns the cluster status
func (r *ClusterResource) Status() string {
	return appaws.Str(r.Item.Status)
}

// RunningTasksCount returns the number of running tasks
func (r *ClusterResource) RunningTasksCount() int32 {
	return r.Item.RunningTasksCount
}

// PendingTasksCount returns the number of pending tasks
func (r *ClusterResource) PendingTasksCount() int32 {
	return r.Item.PendingTasksCount
}

// ActiveServicesCount returns the number of active services
func (r *ClusterResource) ActiveServicesCount() int32 {
	return r.Item.ActiveServicesCount
}

// RegisteredContainerInstancesCount returns the number of container instances
func (r *ClusterResource) RegisteredContainerInstancesCount() int32 {
	return r.Item.RegisteredContainerInstancesCount
}

// CapacityProviders returns the capacity providers
func (r *ClusterResource) CapacityProviders() []string {
	return r.Item.CapacityProviders
}

// Settings returns the cluster settings
func (r *ClusterResource) Settings() []types.ClusterSetting {
	return r.Item.Settings
}
