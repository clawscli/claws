package clusters

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/elasticache"
	"github.com/aws/aws-sdk-go-v2/service/elasticache/types"

	appaws "github.com/clawscli/claws/internal/aws"
	"github.com/clawscli/claws/internal/dao"
	apperrors "github.com/clawscli/claws/internal/errors"
)

// ClusterDAO provides data access for ElastiCache clusters
type ClusterDAO struct {
	dao.BaseDAO
	client *elasticache.Client
}

// NewClusterDAO creates a new ClusterDAO
func NewClusterDAO(ctx context.Context) (dao.DAO, error) {
	cfg, err := appaws.NewConfig(ctx)
	if err != nil {
		return nil, apperrors.Wrap(err, "new "+ServiceResourcePath+" dao")
	}
	return &ClusterDAO{
		BaseDAO: dao.NewBaseDAO("elasticache", "clusters"),
		client:  elasticache.NewFromConfig(cfg),
	}, nil
}

// List returns all ElastiCache clusters
func (d *ClusterDAO) List(ctx context.Context) ([]dao.Resource, error) {
	clusters, err := appaws.Paginate(ctx, func(token *string) ([]types.CacheCluster, *string, error) {
		output, err := d.client.DescribeCacheClusters(ctx, &elasticache.DescribeCacheClustersInput{
			Marker:            token,
			MaxRecords:        appaws.Int32Ptr(100),
			ShowCacheNodeInfo: appaws.BoolPtr(true),
		})
		if err != nil {
			return nil, nil, apperrors.Wrap(err, "describe cache clusters")
		}
		return output.CacheClusters, output.Marker, nil
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

// Get returns a specific ElastiCache cluster by ID
func (d *ClusterDAO) Get(ctx context.Context, id string) (dao.Resource, error) {
	input := &elasticache.DescribeCacheClustersInput{
		CacheClusterId:    &id,
		ShowCacheNodeInfo: appaws.BoolPtr(true),
	}

	output, err := d.client.DescribeCacheClusters(ctx, input)
	if err != nil {
		return nil, apperrors.Wrapf(err, "describe cache cluster %s", id)
	}

	if len(output.CacheClusters) == 0 {
		return nil, fmt.Errorf("cache cluster %s not found", id)
	}

	return NewClusterResource(output.CacheClusters[0]), nil
}

// Delete deletes an ElastiCache cluster
func (d *ClusterDAO) Delete(ctx context.Context, id string) error {
	input := &elasticache.DeleteCacheClusterInput{
		CacheClusterId: &id,
	}

	_, err := d.client.DeleteCacheCluster(ctx, input)
	if err != nil {
		return apperrors.Wrapf(err, "delete cache cluster %s", id)
	}

	return nil
}

// ClusterResource represents an ElastiCache cluster
type ClusterResource struct {
	dao.BaseResource
	Item types.CacheCluster
}

// NewClusterResource creates a new ClusterResource
func NewClusterResource(item types.CacheCluster) *ClusterResource {
	clusterId := appaws.Str(item.CacheClusterId)
	arn := appaws.Str(item.ARN)

	return &ClusterResource{
		BaseResource: dao.BaseResource{
			ID:   clusterId,
			Name: clusterId,
			ARN:  arn,
			Tags: make(map[string]string),
			Data: item,
		},
		Item: item,
	}
}

// ClusterId returns the cluster ID
func (r *ClusterResource) ClusterId() string {
	return appaws.Str(r.Item.CacheClusterId)
}

// Status returns the cluster status
func (r *ClusterResource) Status() string {
	return appaws.Str(r.Item.CacheClusterStatus)
}

// Engine returns the cache engine (redis/memcached)
func (r *ClusterResource) Engine() string {
	return appaws.Str(r.Item.Engine)
}

// EngineVersion returns the engine version
func (r *ClusterResource) EngineVersion() string {
	return appaws.Str(r.Item.EngineVersion)
}

// NodeType returns the cache node type
func (r *ClusterResource) NodeType() string {
	return appaws.Str(r.Item.CacheNodeType)
}

// NumNodes returns the number of cache nodes
func (r *ClusterResource) NumNodes() int32 {
	if r.Item.NumCacheNodes != nil {
		return *r.Item.NumCacheNodes
	}
	return 0
}

// AvailabilityZone returns the preferred availability zone
func (r *ClusterResource) AvailabilityZone() string {
	return appaws.Str(r.Item.PreferredAvailabilityZone)
}

// Endpoint returns the configuration endpoint (for memcached) or primary endpoint
func (r *ClusterResource) Endpoint() string {
	if r.Item.ConfigurationEndpoint != nil {
		return fmt.Sprintf("%s:%d",
			appaws.Str(r.Item.ConfigurationEndpoint.Address),
			appaws.Int32(r.Item.ConfigurationEndpoint.Port))
	}
	// For Redis, get the first cache node endpoint
	if len(r.Item.CacheNodes) > 0 && r.Item.CacheNodes[0].Endpoint != nil {
		return fmt.Sprintf("%s:%d",
			appaws.Str(r.Item.CacheNodes[0].Endpoint.Address),
			appaws.Int32(r.Item.CacheNodes[0].Endpoint.Port))
	}
	return ""
}

// ReplicationGroupId returns the replication group ID (for Redis)
func (r *ClusterResource) ReplicationGroupId() string {
	return appaws.Str(r.Item.ReplicationGroupId)
}

// CreatedAt returns the creation time as a formatted string
func (r *ClusterResource) CreatedAt() string {
	if r.Item.CacheClusterCreateTime != nil {
		return r.Item.CacheClusterCreateTime.Format("2006-01-02 15:04:05")
	}
	return ""
}

// SecurityGroups returns the security group IDs
func (r *ClusterResource) SecurityGroups() []string {
	var sgs []string
	for _, sg := range r.Item.SecurityGroups {
		sgs = append(sgs, appaws.Str(sg.SecurityGroupId))
	}
	return sgs
}

// SubnetGroupName returns the cache subnet group name
func (r *ClusterResource) SubnetGroupName() string {
	return appaws.Str(r.Item.CacheSubnetGroupName)
}

// ParameterGroupName returns the cache parameter group name
func (r *ClusterResource) ParameterGroupName() string {
	if r.Item.CacheParameterGroup != nil {
		return appaws.Str(r.Item.CacheParameterGroup.CacheParameterGroupName)
	}
	return ""
}

// MaintenanceWindow returns the preferred maintenance window
func (r *ClusterResource) MaintenanceWindow() string {
	return appaws.Str(r.Item.PreferredMaintenanceWindow)
}

// SnapshotWindow returns the snapshot window (for Redis)
func (r *ClusterResource) SnapshotWindow() string {
	return appaws.Str(r.Item.SnapshotWindow)
}

// SnapshotRetentionLimit returns the snapshot retention limit
func (r *ClusterResource) SnapshotRetentionLimit() int32 {
	if r.Item.SnapshotRetentionLimit != nil {
		return *r.Item.SnapshotRetentionLimit
	}
	return 0
}

// AutoMinorVersionUpgrade returns whether auto minor version upgrade is enabled
func (r *ClusterResource) AutoMinorVersionUpgrade() bool {
	if r.Item.AutoMinorVersionUpgrade != nil {
		return *r.Item.AutoMinorVersionUpgrade
	}
	return false
}

// TransitEncryptionEnabled returns whether transit encryption is enabled
func (r *ClusterResource) TransitEncryptionEnabled() bool {
	if r.Item.TransitEncryptionEnabled != nil {
		return *r.Item.TransitEncryptionEnabled
	}
	return false
}

// AtRestEncryptionEnabled returns whether at-rest encryption is enabled
func (r *ClusterResource) AtRestEncryptionEnabled() bool {
	if r.Item.AtRestEncryptionEnabled != nil {
		return *r.Item.AtRestEncryptionEnabled
	}
	return false
}
