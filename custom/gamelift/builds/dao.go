package builds

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/gamelift"
	"github.com/aws/aws-sdk-go-v2/service/gamelift/types"

	appaws "github.com/clawscli/claws/internal/aws"
	"github.com/clawscli/claws/internal/dao"
	apperrors "github.com/clawscli/claws/internal/errors"
)

// BuildDAO provides data access for GameLift builds.
type BuildDAO struct {
	dao.BaseDAO
	client *gamelift.Client
}

// NewBuildDAO creates a new BuildDAO.
func NewBuildDAO(ctx context.Context) (dao.DAO, error) {
	cfg, err := appaws.NewConfig(ctx)
	if err != nil {
		return nil, apperrors.Wrap(err, "new "+ServiceResourcePath+" dao")
	}
	return &BuildDAO{
		BaseDAO: dao.NewBaseDAO("gamelift", "builds"),
		client:  gamelift.NewFromConfig(cfg),
	}, nil
}

// List returns all GameLift builds.
func (d *BuildDAO) List(ctx context.Context) ([]dao.Resource, error) {
	builds, err := appaws.Paginate(ctx, func(token *string) ([]types.Build, *string, error) {
		output, err := d.client.ListBuilds(ctx, &gamelift.ListBuildsInput{
			NextToken: token,
		})
		if err != nil {
			return nil, nil, apperrors.Wrap(err, "list gamelift builds")
		}
		return output.Builds, output.NextToken, nil
	})
	if err != nil {
		return nil, err
	}

	resources := make([]dao.Resource, len(builds))
	for i, build := range builds {
		resources[i] = NewBuildResource(build)
	}
	return resources, nil
}

// Get returns a specific GameLift build by ID.
func (d *BuildDAO) Get(ctx context.Context, id string) (dao.Resource, error) {
	output, err := d.client.DescribeBuild(ctx, &gamelift.DescribeBuildInput{
		BuildId: &id,
	})
	if err != nil {
		return nil, apperrors.Wrapf(err, "describe gamelift build %s", id)
	}
	if output.Build == nil {
		return nil, fmt.Errorf("gamelift build %s not found", id)
	}
	return NewBuildResource(*output.Build), nil
}

// Delete deletes a GameLift build by ID.
func (d *BuildDAO) Delete(ctx context.Context, id string) error {
	_, err := d.client.DeleteBuild(ctx, &gamelift.DeleteBuildInput{
		BuildId: &id,
	})
	if err != nil {
		return apperrors.Wrapf(err, "delete gamelift build %s", id)
	}
	return nil
}

// BuildResource wraps a GameLift build.
type BuildResource struct {
	dao.BaseResource
	Build types.Build
}

// NewBuildResource creates a new BuildResource.
func NewBuildResource(build types.Build) *BuildResource {
	return &BuildResource{
		BaseResource: dao.BaseResource{
			ID:   appaws.Str(build.BuildId),
			Name: appaws.Str(build.Name),
			ARN:  appaws.Str(build.BuildArn),
			Data: build,
		},
		Build: build,
	}
}

// Status returns the build status.
func (r *BuildResource) Status() string {
	return string(r.Build.Status)
}

// Version returns the build version.
func (r *BuildResource) Version() string {
	return appaws.Str(r.Build.Version)
}

// OperatingSystem returns the OS.
func (r *BuildResource) OperatingSystem() string {
	return string(r.Build.OperatingSystem)
}

// SizeOnDisk returns the size in bytes.
func (r *BuildResource) SizeOnDisk() int64 {
	return appaws.Int64(r.Build.SizeOnDisk)
}

// CreationTime returns when the build was created.
func (r *BuildResource) CreationTime() *time.Time {
	return r.Build.CreationTime
}

// ServerSdkVersion returns the server SDK version.
func (r *BuildResource) ServerSdkVersion() string {
	return appaws.Str(r.Build.ServerSdkVersion)
}
