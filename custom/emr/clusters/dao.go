package clusters

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/emr"
	"github.com/aws/aws-sdk-go-v2/service/emr/types"

	appaws "github.com/clawscli/claws/internal/aws"
	"github.com/clawscli/claws/internal/dao"
	apperrors "github.com/clawscli/claws/internal/errors"
)

// ClusterDAO provides data access for EMR clusters.
type ClusterDAO struct {
	dao.BaseDAO
	client *emr.Client
}

// NewClusterDAO creates a new ClusterDAO.
func NewClusterDAO(ctx context.Context) (dao.DAO, error) {
	cfg, err := appaws.NewConfig(ctx)
	if err != nil {
		return nil, apperrors.Wrap(err, "new "+ServiceResourcePath+" dao")
	}
	return &ClusterDAO{
		BaseDAO: dao.NewBaseDAO("emr", "clusters"),
		client:  emr.NewFromConfig(cfg),
	}, nil
}

// List returns all EMR clusters.
func (d *ClusterDAO) List(ctx context.Context) ([]dao.Resource, error) {
	clusters, err := appaws.Paginate(ctx, func(token *string) ([]types.ClusterSummary, *string, error) {
		output, err := d.client.ListClusters(ctx, &emr.ListClustersInput{
			Marker: token,
		})
		if err != nil {
			return nil, nil, apperrors.Wrap(err, "list emr clusters")
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
	output, err := d.client.DescribeCluster(ctx, &emr.DescribeClusterInput{
		ClusterId: &id,
	})
	if err != nil {
		return nil, apperrors.Wrap(err, "describe emr cluster")
	}

	cluster := output.Cluster
	apps := make([]string, len(cluster.Applications))
	for i, app := range cluster.Applications {
		if app.Name != nil {
			apps[i] = *app.Name
		}
	}
	scaleDown := ""
	if cluster.ScaleDownBehavior != "" {
		scaleDown = string(cluster.ScaleDownBehavior)
	}
	visibleToAll := false
	if cluster.VisibleToAllUsers != nil {
		visibleToAll = *cluster.VisibleToAllUsers
	}

	return &ClusterResource{
		BaseResource: dao.BaseResource{
			ID:  appaws.Str(cluster.Id),
			ARN: appaws.Str(cluster.ClusterArn),
		},
		Cluster: &types.ClusterSummary{
			Id:   cluster.Id,
			Name: cluster.Name,
			Status: &types.ClusterStatus{
				State: cluster.Status.State,
			},
			NormalizedInstanceHours: cluster.NormalizedInstanceHours,
		},
		ReleaseLabel:        appaws.Str(cluster.ReleaseLabel),
		LogUri:              appaws.Str(cluster.LogUri),
		ServiceRole:         appaws.Str(cluster.ServiceRole),
		AutoScalingRole:     appaws.Str(cluster.AutoScalingRole),
		Applications:        apps,
		Ec2InstanceAttrs:    cluster.Ec2InstanceAttributes,
		VisibleToAllUsers:   visibleToAll,
		StateChangeReason:   cluster.Status.StateChangeReason,
		Tags:                cluster.Tags,
		ScaleDownBehavior:   scaleDown,
		MasterPublicDnsName: appaws.Str(cluster.MasterPublicDnsName),
	}, nil
}

// Delete terminates an EMR cluster.
func (d *ClusterDAO) Delete(ctx context.Context, id string) error {
	_, err := d.client.TerminateJobFlows(ctx, &emr.TerminateJobFlowsInput{
		JobFlowIds: []string{id},
	})
	if err != nil {
		return apperrors.Wrap(err, "terminate emr cluster")
	}
	return nil
}

// ClusterResource wraps an EMR cluster.
type ClusterResource struct {
	dao.BaseResource
	Cluster             *types.ClusterSummary
	ReleaseLabel        string
	LogUri              string
	ServiceRole         string
	AutoScalingRole     string
	Applications        []string
	Ec2InstanceAttrs    *types.Ec2InstanceAttributes
	VisibleToAllUsers   bool
	StateChangeReason   *types.ClusterStateChangeReason
	Tags                []types.Tag
	ScaleDownBehavior   string
	MasterPublicDnsName string
}

// NewClusterResource creates a new ClusterResource.
func NewClusterResource(cluster types.ClusterSummary) *ClusterResource {
	return &ClusterResource{
		BaseResource: dao.BaseResource{
			ID:  appaws.Str(cluster.Id),
			ARN: appaws.Str(cluster.ClusterArn),
		},
		Cluster: &cluster,
	}
}

// Name returns the cluster name.
func (r *ClusterResource) Name() string {
	if r.Cluster != nil && r.Cluster.Name != nil {
		return *r.Cluster.Name
	}
	return ""
}

// State returns the cluster state.
func (r *ClusterResource) State() string {
	if r.Cluster != nil && r.Cluster.Status != nil {
		return string(r.Cluster.Status.State)
	}
	return ""
}

// NormalizedInstanceHours returns the normalized instance hours.
func (r *ClusterResource) NormalizedInstanceHours() int32 {
	if r.Cluster != nil && r.Cluster.NormalizedInstanceHours != nil {
		return *r.Cluster.NormalizedInstanceHours
	}
	return 0
}

// GetLogUri returns the log URI.
func (r *ClusterResource) GetLogUri() string {
	return r.LogUri
}

// GetServiceRole returns the service role.
func (r *ClusterResource) GetServiceRole() string {
	return r.ServiceRole
}

// GetAutoScalingRole returns the auto scaling role.
func (r *ClusterResource) GetAutoScalingRole() string {
	return r.AutoScalingRole
}

// GetApplications returns the applications.
func (r *ClusterResource) GetApplications() []string {
	return r.Applications
}

// GetEc2InstanceAttrs returns EC2 instance attributes.
func (r *ClusterResource) GetEc2InstanceAttrs() *types.Ec2InstanceAttributes {
	return r.Ec2InstanceAttrs
}

// GetVisibleToAllUsers returns visibility setting.
func (r *ClusterResource) GetVisibleToAllUsers() bool {
	return r.VisibleToAllUsers
}

// GetStateChangeReason returns state change reason.
func (r *ClusterResource) GetStateChangeReason() *types.ClusterStateChangeReason {
	return r.StateChangeReason
}

// GetClusterTags returns the cluster tags.
func (r *ClusterResource) GetClusterTags() []types.Tag {
	return r.Tags
}

// GetScaleDownBehavior returns scale down behavior.
func (r *ClusterResource) GetScaleDownBehavior() string {
	return r.ScaleDownBehavior
}

// GetMasterPublicDnsName returns the master public DNS name.
func (r *ClusterResource) GetMasterPublicDnsName() string {
	return r.MasterPublicDnsName
}
