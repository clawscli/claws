package addons

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

// AddonDAO provides data access for EKS add-ons
type AddonDAO struct {
	dao.BaseDAO
	client *eks.Client
}

// NewAddonDAO creates a new AddonDAO
func NewAddonDAO(ctx context.Context) (dao.DAO, error) {
	cfg, err := appaws.NewConfig(ctx)
	if err != nil {
		return nil, apperrors.Wrap(err, "new eks/addons dao")
	}
	return &AddonDAO{
		BaseDAO: dao.NewBaseDAO("eks", "addons"),
		client:  eks.NewFromConfig(cfg),
	}, nil
}

func (d *AddonDAO) List(ctx context.Context) ([]dao.Resource, error) {
	clusterName := dao.GetFilterFromContext(ctx, "ClusterName")
	if clusterName == "" {
		return nil, fmt.Errorf("ClusterName filter required")
	}

	addonNames, err := appaws.Paginate(ctx, func(token *string) ([]string, *string, error) {
		output, err := d.client.ListAddons(ctx, &eks.ListAddonsInput{
			ClusterName: &clusterName,
			NextToken:   token,
		})
		if err != nil {
			return nil, nil, apperrors.Wrap(err, "list addons")
		}
		return output.Addons, output.NextToken, nil
	})
	if err != nil {
		return nil, err
	}

	if len(addonNames) == 0 {
		return nil, nil
	}

	resources := make([]dao.Resource, 0, len(addonNames))
	for _, name := range addonNames {
		output, err := d.client.DescribeAddon(ctx, &eks.DescribeAddonInput{
			ClusterName: &clusterName,
			AddonName:   &name,
		})
		if err != nil {
			if apperrors.IsNotFound(err) {
				continue
			}
			return nil, apperrors.Wrapf(err, "describe addon %s", name)
		}
		if output.Addon != nil {
			resources = append(resources, NewAddonResource(*output.Addon))
		}
	}

	return resources, nil
}

func (d *AddonDAO) Get(ctx context.Context, id string) (dao.Resource, error) {
	clusterName := dao.GetFilterFromContext(ctx, "ClusterName")
	if clusterName == "" {
		return nil, fmt.Errorf("ClusterName filter required")
	}

	output, err := d.client.DescribeAddon(ctx, &eks.DescribeAddonInput{
		ClusterName: &clusterName,
		AddonName:   &id,
	})
	if err != nil {
		return nil, apperrors.Wrapf(err, "describe addon %s", id)
	}

	if output.Addon == nil {
		return nil, fmt.Errorf("addon not found: %s", id)
	}

	return NewAddonResource(*output.Addon), nil
}

func (d *AddonDAO) Delete(ctx context.Context, id string) error {
	clusterName := dao.GetFilterFromContext(ctx, "ClusterName")
	if clusterName == "" {
		return fmt.Errorf("ClusterName filter required")
	}

	_, err := d.client.DeleteAddon(ctx, &eks.DeleteAddonInput{
		ClusterName: &clusterName,
		AddonName:   &id,
	})
	if err != nil {
		if apperrors.IsNotFound(err) {
			return nil
		}
		return apperrors.Wrapf(err, "delete addon %s", id)
	}
	return nil
}

// AddonResource represents an EKS add-on resource
type AddonResource struct {
	dao.BaseResource
	Addon types.Addon
}

// NewAddonResource creates a new AddonResource
func NewAddonResource(addon types.Addon) *AddonResource {
	return &AddonResource{
		BaseResource: dao.BaseResource{
			ID:   appaws.Str(addon.AddonName),
			Name: appaws.Str(addon.AddonName),
			ARN:  appaws.Str(addon.AddonArn),
			Data: addon,
		},
		Addon: addon,
	}
}

// Status returns addon status
func (r *AddonResource) Status() string {
	return string(r.Addon.Status)
}

// Version returns addon version
func (r *AddonResource) Version() string {
	return appaws.Str(r.Addon.AddonVersion)
}

// CreatedAge returns age since creation
func (r *AddonResource) CreatedAge() string {
	return render.FormatAge(appaws.Time(r.Addon.CreatedAt))
}
