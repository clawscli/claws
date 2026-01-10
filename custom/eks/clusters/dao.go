package clusters

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/eks"
	"github.com/aws/aws-sdk-go-v2/service/eks/types"

	appaws "github.com/clawscli/claws/internal/aws"
	"github.com/clawscli/claws/internal/dao"
	apperrors "github.com/clawscli/claws/internal/errors"
	"github.com/clawscli/claws/internal/render"
)

// ClusterDAO provides data access for EKS clusters
type ClusterDAO struct {
	dao.BaseDAO
	client *eks.Client
}

// NewClusterDAO creates a new ClusterDAO
func NewClusterDAO(ctx context.Context) (dao.DAO, error) {
	cfg, err := appaws.NewConfig(ctx)
	if err != nil {
		return nil, apperrors.Wrap(err, "new eks/clusters dao")
	}
	return &ClusterDAO{
		BaseDAO: dao.NewBaseDAO("eks", "clusters"),
		client:  eks.NewFromConfig(cfg),
	}, nil
}

func (d *ClusterDAO) List(ctx context.Context) ([]dao.Resource, error) {
	// Check for ClusterName filter (for navigation from child resources)
	if clusterName := dao.GetFilterFromContext(ctx, "ClusterName"); clusterName != "" {
		// Direct lookup for specific cluster
		cluster, err := d.Get(ctx, clusterName)
		if err != nil {
			// If not found, return empty list (not an error for filtering)
			if apperrors.IsNotFound(err) {
				return []dao.Resource{}, nil
			}
			return nil, err
		}
		return []dao.Resource{cluster}, nil
	}

	// List cluster names
	clusterNames, err := appaws.Paginate(ctx, func(token *string) ([]string, *string, error) {
		output, err := d.client.ListClusters(ctx, &eks.ListClustersInput{
			NextToken: token,
		})
		if err != nil {
			return nil, nil, apperrors.Wrap(err, "list clusters")
		}
		return output.Clusters, output.NextToken, nil
	})
	if err != nil {
		return nil, err
	}

	if len(clusterNames) == 0 {
		return nil, nil
	}

	// Describe each cluster to get details
	resources := make([]dao.Resource, 0, len(clusterNames))
	for _, name := range clusterNames {
		output, err := d.client.DescribeCluster(ctx, &eks.DescribeClusterInput{
			Name: &name,
		})
		if err != nil {
			if apperrors.IsNotFound(err) {
				continue
			}
			return nil, apperrors.Wrapf(err, "describe cluster %s", name)
		}
		if output.Cluster != nil {
			resources = append(resources, NewClusterResource(*output.Cluster))
		}
	}

	return resources, nil
}

func (d *ClusterDAO) Get(ctx context.Context, id string) (dao.Resource, error) {
	output, err := d.client.DescribeCluster(ctx, &eks.DescribeClusterInput{
		Name: &id,
	})
	if err != nil {
		return nil, apperrors.Wrapf(err, "describe cluster %s", id)
	}

	if output.Cluster == nil {
		return nil, fmt.Errorf("cluster not found: %s", id)
	}

	return NewClusterResource(*output.Cluster), nil
}

func (d *ClusterDAO) Delete(ctx context.Context, id string) error {
	_, err := d.client.DeleteCluster(ctx, &eks.DeleteClusterInput{
		Name: &id,
	})
	if err != nil {
		if apperrors.IsNotFound(err) {
			return nil // Already deleted
		}
		return apperrors.Wrapf(err, "delete cluster %s", id)
	}
	return nil
}

// ClusterResource represents an EKS cluster resource
type ClusterResource struct {
	dao.BaseResource
	Cluster types.Cluster
}

// NewClusterResource creates a new ClusterResource
func NewClusterResource(cluster types.Cluster) *ClusterResource {
	return &ClusterResource{
		BaseResource: dao.BaseResource{
			ID:   appaws.Str(cluster.Name),
			Name: appaws.Str(cluster.Name),
			ARN:  appaws.Str(cluster.Arn),
			Data: cluster,
		},
		Cluster: cluster,
	}
}

// Status returns cluster status
func (r *ClusterResource) Status() string {
	return string(r.Cluster.Status)
}

// Version returns Kubernetes version
func (r *ClusterResource) Version() string {
	return appaws.Str(r.Cluster.Version)
}

// Endpoint returns API server endpoint
func (r *ClusterResource) Endpoint() string {
	return appaws.Str(r.Cluster.Endpoint)
}

// CreatedAge returns age since creation
func (r *ClusterResource) CreatedAge() string {
	return render.FormatAge(appaws.Time(r.Cluster.CreatedAt))
}

// PlatformVersion returns platform version
func (r *ClusterResource) PlatformVersion() string {
	return appaws.Str(r.Cluster.PlatformVersion)
}
