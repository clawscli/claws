package clusters

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/redshift"
	"github.com/aws/aws-sdk-go-v2/service/redshift/types"

	appaws "github.com/clawscli/claws/internal/aws"
	"github.com/clawscli/claws/internal/dao"
)

// ClusterDAO provides data access for Redshift clusters.
type ClusterDAO struct {
	dao.BaseDAO
	client *redshift.Client
}

// NewClusterDAO creates a new ClusterDAO.
func NewClusterDAO(ctx context.Context) (dao.DAO, error) {
	cfg, err := appaws.NewConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("new redshift/clusters dao: %w", err)
	}
	return &ClusterDAO{
		BaseDAO: dao.NewBaseDAO("redshift", "clusters"),
		client:  redshift.NewFromConfig(cfg),
	}, nil
}

// List returns all Redshift clusters.
func (d *ClusterDAO) List(ctx context.Context) ([]dao.Resource, error) {
	clusters, err := appaws.Paginate(ctx, func(token *string) ([]types.Cluster, *string, error) {
		output, err := d.client.DescribeClusters(ctx, &redshift.DescribeClustersInput{
			Marker: token,
		})
		if err != nil {
			return nil, nil, fmt.Errorf("describe redshift clusters: %w", err)
		}
		return output.Clusters, output.Marker, nil
	})
	if err != nil {
		return nil, err
	}

	resources := make([]dao.Resource, len(clusters))
	for i, cluster := range clusters {
		resources[i] = NewClusterResource(cluster)
	}
	return resources, nil
}

// Get returns a specific cluster.
func (d *ClusterDAO) Get(ctx context.Context, id string) (dao.Resource, error) {
	output, err := d.client.DescribeClusters(ctx, &redshift.DescribeClustersInput{
		ClusterIdentifier: &id,
	})
	if err != nil {
		return nil, fmt.Errorf("describe redshift cluster: %w", err)
	}
	if len(output.Clusters) == 0 {
		return nil, fmt.Errorf("cluster not found: %s", id)
	}
	return NewClusterResource(output.Clusters[0]), nil
}

// Delete deletes a Redshift cluster.
func (d *ClusterDAO) Delete(ctx context.Context, id string) error {
	skipFinalSnapshot := true
	_, err := d.client.DeleteCluster(ctx, &redshift.DeleteClusterInput{
		ClusterIdentifier:        &id,
		SkipFinalClusterSnapshot: &skipFinalSnapshot,
	})
	if err != nil {
		return fmt.Errorf("delete redshift cluster: %w", err)
	}
	return nil
}

// ClusterResource wraps a Redshift cluster.
type ClusterResource struct {
	dao.BaseResource
	Cluster *types.Cluster
}

// NewClusterResource creates a new ClusterResource.
func NewClusterResource(cluster types.Cluster) *ClusterResource {
	return &ClusterResource{
		BaseResource: dao.BaseResource{
			ID:  appaws.Str(cluster.ClusterIdentifier),
			ARN: "",
		},
		Cluster: &cluster,
	}
}

// Status returns the cluster status.
func (r *ClusterResource) Status() string {
	if r.Cluster != nil && r.Cluster.ClusterStatus != nil {
		return *r.Cluster.ClusterStatus
	}
	return ""
}

// NodeType returns the node type.
func (r *ClusterResource) NodeType() string {
	if r.Cluster != nil && r.Cluster.NodeType != nil {
		return *r.Cluster.NodeType
	}
	return ""
}

// NumberOfNodes returns the number of nodes.
func (r *ClusterResource) NumberOfNodes() int32 {
	if r.Cluster != nil && r.Cluster.NumberOfNodes != nil {
		return *r.Cluster.NumberOfNodes
	}
	return 0
}

// DBName returns the database name.
func (r *ClusterResource) DBName() string {
	if r.Cluster != nil && r.Cluster.DBName != nil {
		return *r.Cluster.DBName
	}
	return ""
}

// MasterUsername returns the master username.
func (r *ClusterResource) MasterUsername() string {
	if r.Cluster != nil && r.Cluster.MasterUsername != nil {
		return *r.Cluster.MasterUsername
	}
	return ""
}

// Endpoint returns the cluster endpoint.
func (r *ClusterResource) Endpoint() string {
	if r.Cluster != nil && r.Cluster.Endpoint != nil && r.Cluster.Endpoint.Address != nil {
		return fmt.Sprintf("%s:%d", *r.Cluster.Endpoint.Address, r.Cluster.Endpoint.Port)
	}
	return ""
}

// VpcId returns the VPC ID.
func (r *ClusterResource) VpcId() string {
	if r.Cluster != nil && r.Cluster.VpcId != nil {
		return *r.Cluster.VpcId
	}
	return ""
}

// CreatedAt returns when the cluster was created.
func (r *ClusterResource) CreatedAt() *time.Time {
	if r.Cluster != nil {
		return r.Cluster.ClusterCreateTime
	}
	return nil
}
