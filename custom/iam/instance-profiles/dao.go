package instanceprofiles

import (
	"context"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/iam/types"

	appaws "github.com/clawscli/claws/internal/aws"
	"github.com/clawscli/claws/internal/dao"
	apperrors "github.com/clawscli/claws/internal/errors"
)

// InstanceProfileDAO provides data access for IAM Instance Profiles
type InstanceProfileDAO struct {
	dao.BaseDAO
	client *iam.Client
}

// NewInstanceProfileDAO creates a new InstanceProfileDAO
func NewInstanceProfileDAO(ctx context.Context) (dao.DAO, error) {
	cfg, err := appaws.NewConfig(ctx)
	if err != nil {
		return nil, apperrors.Wrap(err, "new iam/instanceprofiles dao")
	}
	return &InstanceProfileDAO{
		BaseDAO: dao.NewBaseDAO("iam", "instance-profiles"),
		client:  iam.NewFromConfig(cfg),
	}, nil
}

func (d *InstanceProfileDAO) List(ctx context.Context) ([]dao.Resource, error) {
	var resources []dao.Resource

	paginator := iam.NewListInstanceProfilesPaginator(d.client, &iam.ListInstanceProfilesInput{})
	for paginator.HasMorePages() {
		output, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, apperrors.Wrap(err, "list instance profiles")
		}

		for _, profile := range output.InstanceProfiles {
			resources = append(resources, NewInstanceProfileResource(profile))
		}
	}

	return resources, nil
}

func (d *InstanceProfileDAO) Get(ctx context.Context, id string) (dao.Resource, error) {
	output, err := d.client.GetInstanceProfile(ctx, &iam.GetInstanceProfileInput{
		InstanceProfileName: &id,
	})
	if err != nil {
		return nil, apperrors.Wrapf(err, "get instance profile %s", id)
	}

	return NewInstanceProfileResource(*output.InstanceProfile), nil
}

func (d *InstanceProfileDAO) Delete(ctx context.Context, id string) error {
	_, err := d.client.DeleteInstanceProfile(ctx, &iam.DeleteInstanceProfileInput{
		InstanceProfileName: &id,
	})
	if err != nil {
		return apperrors.Wrapf(err, "delete instance profile %s", id)
	}
	return nil
}

// InstanceProfileResource wraps an IAM Instance Profile
type InstanceProfileResource struct {
	dao.BaseResource
	Item types.InstanceProfile
}

// NewInstanceProfileResource creates a new InstanceProfileResource
func NewInstanceProfileResource(profile types.InstanceProfile) *InstanceProfileResource {
	return &InstanceProfileResource{
		BaseResource: dao.BaseResource{
			ID:   appaws.Str(profile.InstanceProfileId),
			Name: appaws.Str(profile.InstanceProfileName),
			ARN:  appaws.Str(profile.Arn),
			Data: profile,
		},
		Item: profile,
	}
}

// RoleNames returns the names of associated roles
func (r *InstanceProfileResource) RoleNames() []string {
	var names []string
	for _, role := range r.Item.Roles {
		if role.RoleName != nil {
			names = append(names, *role.RoleName)
		}
	}
	return names
}

// RoleNamesString returns comma-separated role names
func (r *InstanceProfileResource) RoleNamesString() string {
	return strings.Join(r.RoleNames(), ", ")
}

// Path returns the IAM path
func (r *InstanceProfileResource) Path() string {
	if r.Item.Path != nil {
		return *r.Item.Path
	}
	return ""
}
