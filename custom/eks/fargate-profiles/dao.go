package fargateprofiles

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

// FargateProfileDAO provides data access for EKS Fargate profiles
type FargateProfileDAO struct {
	dao.BaseDAO
	client *eks.Client
}

// NewFargateProfileDAO creates a new FargateProfileDAO
func NewFargateProfileDAO(ctx context.Context) (dao.DAO, error) {
	cfg, err := appaws.NewConfig(ctx)
	if err != nil {
		return nil, apperrors.Wrap(err, "new eks/fargate-profiles dao")
	}
	return &FargateProfileDAO{
		BaseDAO: dao.NewBaseDAO("eks", "fargate-profiles"),
		client:  eks.NewFromConfig(cfg),
	}, nil
}

func (d *FargateProfileDAO) List(ctx context.Context) ([]dao.Resource, error) {
	clusterName := dao.GetFilterFromContext(ctx, "ClusterName")
	if clusterName == "" {
		return nil, fmt.Errorf("ClusterName filter required")
	}

	profileNames, err := appaws.Paginate(ctx, func(token *string) ([]string, *string, error) {
		output, err := d.client.ListFargateProfiles(ctx, &eks.ListFargateProfilesInput{
			ClusterName: &clusterName,
			NextToken:   token,
		})
		if err != nil {
			return nil, nil, apperrors.Wrap(err, "list fargate profiles")
		}
		return output.FargateProfileNames, output.NextToken, nil
	})
	if err != nil {
		return nil, err
	}

	if len(profileNames) == 0 {
		return nil, nil
	}

	resources := make([]dao.Resource, 0, len(profileNames))
	for _, name := range profileNames {
		output, err := d.client.DescribeFargateProfile(ctx, &eks.DescribeFargateProfileInput{
			ClusterName:        &clusterName,
			FargateProfileName: &name,
		})
		if err != nil {
			if apperrors.IsNotFound(err) {
				continue
			}
			return nil, apperrors.Wrapf(err, "describe fargate profile %s", name)
		}
		if output.FargateProfile != nil {
			resources = append(resources, NewFargateProfileResource(*output.FargateProfile))
		}
	}

	return resources, nil
}

func (d *FargateProfileDAO) Get(ctx context.Context, id string) (dao.Resource, error) {
	clusterName := dao.GetFilterFromContext(ctx, "ClusterName")
	if clusterName == "" {
		return nil, fmt.Errorf("ClusterName filter required")
	}

	output, err := d.client.DescribeFargateProfile(ctx, &eks.DescribeFargateProfileInput{
		ClusterName:        &clusterName,
		FargateProfileName: &id,
	})
	if err != nil {
		return nil, apperrors.Wrapf(err, "describe fargate profile %s", id)
	}

	if output.FargateProfile == nil {
		return nil, fmt.Errorf("fargate profile not found: %s", id)
	}

	return NewFargateProfileResource(*output.FargateProfile), nil
}

func (d *FargateProfileDAO) Delete(ctx context.Context, id string) error {
	clusterName := dao.GetFilterFromContext(ctx, "ClusterName")
	if clusterName == "" {
		return fmt.Errorf("ClusterName filter required")
	}

	_, err := d.client.DeleteFargateProfile(ctx, &eks.DeleteFargateProfileInput{
		ClusterName:        &clusterName,
		FargateProfileName: &id,
	})
	if err != nil {
		if apperrors.IsNotFound(err) {
			return nil
		}
		return apperrors.Wrapf(err, "delete fargate profile %s", id)
	}
	return nil
}

// FargateProfileResource represents an EKS Fargate profile resource
type FargateProfileResource struct {
	dao.BaseResource
	FargateProfile types.FargateProfile
}

// NewFargateProfileResource creates a new FargateProfileResource
func NewFargateProfileResource(fp types.FargateProfile) *FargateProfileResource {
	return &FargateProfileResource{
		BaseResource: dao.BaseResource{
			ID:   appaws.Str(fp.FargateProfileName),
			Name: appaws.Str(fp.FargateProfileName),
			ARN:  appaws.Str(fp.FargateProfileArn),
			Data: fp,
		},
		FargateProfile: fp,
	}
}

// Status returns fargate profile status
func (r *FargateProfileResource) Status() string {
	return string(r.FargateProfile.Status)
}

// CreatedAge returns age since creation
func (r *FargateProfileResource) CreatedAge() string {
	return render.FormatAge(appaws.Time(r.FargateProfile.CreatedAt))
}
