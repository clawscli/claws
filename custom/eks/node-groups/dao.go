package nodegroups

import (
	"context"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/eks"
	"github.com/aws/aws-sdk-go-v2/service/eks/types"

	appaws "github.com/clawscli/claws/internal/aws"
	"github.com/clawscli/claws/internal/dao"
	apperrors "github.com/clawscli/claws/internal/errors"
	"github.com/clawscli/claws/internal/render"
)

// NodeGroupDAO provides data access for EKS node groups
type NodeGroupDAO struct {
	dao.BaseDAO
	client *eks.Client
}

// NewNodeGroupDAO creates a new NodeGroupDAO
func NewNodeGroupDAO(ctx context.Context) (dao.DAO, error) {
	cfg, err := appaws.NewConfig(ctx)
	if err != nil {
		return nil, apperrors.Wrap(err, "new eks/node-groups dao")
	}
	return &NodeGroupDAO{
		BaseDAO: dao.NewBaseDAO("eks", "node-groups"),
		client:  eks.NewFromConfig(cfg),
	}, nil
}

func (d *NodeGroupDAO) List(ctx context.Context) ([]dao.Resource, error) {
	// Get cluster name from context filter
	clusterName := dao.GetFilterFromContext(ctx, "ClusterName")
	if clusterName == "" {
		return nil, fmt.Errorf("ClusterName filter required")
	}

	// List node group names
	nodegroupNames, err := appaws.Paginate(ctx, func(token *string) ([]string, *string, error) {
		output, err := d.client.ListNodegroups(ctx, &eks.ListNodegroupsInput{
			ClusterName: &clusterName,
			NextToken:   token,
		})
		if err != nil {
			return nil, nil, apperrors.Wrap(err, "list node groups")
		}
		return output.Nodegroups, output.NextToken, nil
	})
	if err != nil {
		return nil, err
	}

	if len(nodegroupNames) == 0 {
		return nil, nil
	}

	// Describe each node group to get details
	resources := make([]dao.Resource, 0, len(nodegroupNames))
	for _, name := range nodegroupNames {
		output, err := d.client.DescribeNodegroup(ctx, &eks.DescribeNodegroupInput{
			ClusterName:   &clusterName,
			NodegroupName: &name,
		})
		if err != nil {
			if apperrors.IsNotFound(err) {
				continue
			}
			return nil, apperrors.Wrapf(err, "describe node group %s", name)
		}
		if output.Nodegroup != nil {
			resources = append(resources, NewNodeGroupResource(*output.Nodegroup))
		}
	}

	return resources, nil
}

func (d *NodeGroupDAO) Get(ctx context.Context, id string) (dao.Resource, error) {
	// Get cluster name from context filter
	clusterName := dao.GetFilterFromContext(ctx, "ClusterName")
	if clusterName == "" {
		return nil, fmt.Errorf("ClusterName filter required")
	}

	output, err := d.client.DescribeNodegroup(ctx, &eks.DescribeNodegroupInput{
		ClusterName:   &clusterName,
		NodegroupName: &id,
	})
	if err != nil {
		return nil, apperrors.Wrapf(err, "describe node group %s", id)
	}

	if output.Nodegroup == nil {
		return nil, fmt.Errorf("node group not found: %s", id)
	}

	return NewNodeGroupResource(*output.Nodegroup), nil
}

func (d *NodeGroupDAO) Delete(ctx context.Context, id string) error {
	// Get cluster name from context filter
	clusterName := dao.GetFilterFromContext(ctx, "ClusterName")
	if clusterName == "" {
		return fmt.Errorf("ClusterName filter required")
	}

	_, err := d.client.DeleteNodegroup(ctx, &eks.DeleteNodegroupInput{
		ClusterName:   &clusterName,
		NodegroupName: &id,
	})
	if err != nil {
		if apperrors.IsNotFound(err) {
			return nil // Already deleted
		}
		return apperrors.Wrapf(err, "delete node group %s", id)
	}
	return nil
}

// NodeGroupResource represents an EKS node group resource
type NodeGroupResource struct {
	dao.BaseResource
	NodeGroup types.Nodegroup
}

// NewNodeGroupResource creates a new NodeGroupResource
func NewNodeGroupResource(ng types.Nodegroup) *NodeGroupResource {
	return &NodeGroupResource{
		BaseResource: dao.BaseResource{
			ID:   appaws.Str(ng.NodegroupName),
			Name: appaws.Str(ng.NodegroupName),
			ARN:  appaws.Str(ng.NodegroupArn),
			Data: ng,
		},
		NodeGroup: ng,
	}
}

// Status returns node group status
func (r *NodeGroupResource) Status() string {
	return string(r.NodeGroup.Status)
}

// Version returns Kubernetes version
func (r *NodeGroupResource) Version() string {
	return appaws.Str(r.NodeGroup.Version)
}

// InstanceTypes returns comma-separated instance types
func (r *NodeGroupResource) InstanceTypes() string {
	if len(r.NodeGroup.InstanceTypes) == 0 {
		return ""
	}
	return strings.Join(r.NodeGroup.InstanceTypes, ", ")
}

// DesiredSize returns desired node count
func (r *NodeGroupResource) DesiredSize() int32 {
	if r.NodeGroup.ScalingConfig == nil {
		return 0
	}
	return appaws.Int32(r.NodeGroup.ScalingConfig.DesiredSize)
}

// MinSize returns minimum node count
func (r *NodeGroupResource) MinSize() int32 {
	if r.NodeGroup.ScalingConfig == nil {
		return 0
	}
	return appaws.Int32(r.NodeGroup.ScalingConfig.MinSize)
}

// MaxSize returns maximum node count
func (r *NodeGroupResource) MaxSize() int32 {
	if r.NodeGroup.ScalingConfig == nil {
		return 0
	}
	return appaws.Int32(r.NodeGroup.ScalingConfig.MaxSize)
}

// CreatedAge returns age since creation
func (r *NodeGroupResource) CreatedAge() string {
	return render.FormatAge(appaws.Time(r.NodeGroup.CreatedAt))
}
