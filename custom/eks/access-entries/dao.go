package accessentries

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

// AccessEntryDAO provides data access for EKS access entries
type AccessEntryDAO struct {
	dao.BaseDAO
	client *eks.Client
}

// NewAccessEntryDAO creates a new AccessEntryDAO
func NewAccessEntryDAO(ctx context.Context) (dao.DAO, error) {
	cfg, err := appaws.NewConfig(ctx)
	if err != nil {
		return nil, apperrors.Wrap(err, "new eks/access-entries dao")
	}
	return &AccessEntryDAO{
		BaseDAO: dao.NewBaseDAO("eks", "access-entries"),
		client:  eks.NewFromConfig(cfg),
	}, nil
}

func (d *AccessEntryDAO) List(ctx context.Context) ([]dao.Resource, error) {
	clusterName := dao.GetFilterFromContext(ctx, "ClusterName")
	if clusterName == "" {
		return nil, fmt.Errorf("ClusterName filter required")
	}

	principalArns, err := appaws.Paginate(ctx, func(token *string) ([]string, *string, error) {
		output, err := d.client.ListAccessEntries(ctx, &eks.ListAccessEntriesInput{
			ClusterName: &clusterName,
			NextToken:   token,
		})
		if err != nil {
			return nil, nil, apperrors.Wrap(err, "list access entries")
		}
		return output.AccessEntries, output.NextToken, nil
	})
	if err != nil {
		return nil, err
	}

	if len(principalArns) == 0 {
		return nil, nil
	}

	resources := make([]dao.Resource, 0, len(principalArns))
	for _, arn := range principalArns {
		output, err := d.client.DescribeAccessEntry(ctx, &eks.DescribeAccessEntryInput{
			ClusterName:  &clusterName,
			PrincipalArn: &arn,
		})
		if err != nil {
			if apperrors.IsNotFound(err) {
				continue
			}
			return nil, apperrors.Wrapf(err, "describe access entry %s", arn)
		}
		if output.AccessEntry != nil {
			resources = append(resources, NewAccessEntryResource(*output.AccessEntry))
		}
	}

	return resources, nil
}

func (d *AccessEntryDAO) Get(ctx context.Context, id string) (dao.Resource, error) {
	clusterName := dao.GetFilterFromContext(ctx, "ClusterName")
	if clusterName == "" {
		return nil, fmt.Errorf("ClusterName filter required")
	}

	output, err := d.client.DescribeAccessEntry(ctx, &eks.DescribeAccessEntryInput{
		ClusterName:  &clusterName,
		PrincipalArn: &id,
	})
	if err != nil {
		return nil, apperrors.Wrapf(err, "describe access entry %s", id)
	}

	if output.AccessEntry == nil {
		return nil, fmt.Errorf("access entry not found: %s", id)
	}

	return NewAccessEntryResource(*output.AccessEntry), nil
}

func (d *AccessEntryDAO) Delete(ctx context.Context, id string) error {
	clusterName := dao.GetFilterFromContext(ctx, "ClusterName")
	if clusterName == "" {
		return fmt.Errorf("ClusterName filter required")
	}

	_, err := d.client.DeleteAccessEntry(ctx, &eks.DeleteAccessEntryInput{
		ClusterName:  &clusterName,
		PrincipalArn: &id,
	})
	if err != nil {
		if apperrors.IsNotFound(err) {
			return nil
		}
		return apperrors.Wrapf(err, "delete access entry %s", id)
	}
	return nil
}

// AccessEntryResource represents an EKS access entry resource
type AccessEntryResource struct {
	dao.BaseResource
	AccessEntry types.AccessEntry
}

// NewAccessEntryResource creates a new AccessEntryResource
func NewAccessEntryResource(ae types.AccessEntry) *AccessEntryResource {
	principalArn := appaws.Str(ae.PrincipalArn)
	return &AccessEntryResource{
		BaseResource: dao.BaseResource{
			ID:   principalArn,
			Name: principalArn,
			ARN:  appaws.Str(ae.AccessEntryArn),
			Data: ae,
		},
		AccessEntry: ae,
	}
}

// Type returns access entry type
func (r *AccessEntryResource) Type() string {
	return appaws.Str(r.AccessEntry.Type)
}

// Username returns Kubernetes username
func (r *AccessEntryResource) Username() string {
	return appaws.Str(r.AccessEntry.Username)
}

// CreatedAge returns age since creation
func (r *AccessEntryResource) CreatedAge() string {
	return render.FormatAge(appaws.Time(r.AccessEntry.CreatedAt))
}
