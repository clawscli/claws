package jobdefinitions

import (
	"context"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/batch"
	"github.com/aws/aws-sdk-go-v2/service/batch/types"

	appaws "github.com/clawscli/claws/internal/aws"
	"github.com/clawscli/claws/internal/dao"
	apperrors "github.com/clawscli/claws/internal/errors"
)

// JobDefinitionDAO provides data access for Batch job definitions.
type JobDefinitionDAO struct {
	dao.BaseDAO
	client *batch.Client
}

// NewJobDefinitionDAO creates a new JobDefinitionDAO.
func NewJobDefinitionDAO(ctx context.Context) (dao.DAO, error) {
	cfg, err := appaws.NewConfig(ctx)
	if err != nil {
		return nil, apperrors.Wrap(err, "new batch/job-definitions dao")
	}
	return &JobDefinitionDAO{
		BaseDAO: dao.NewBaseDAO("batch", "job-definitions"),
		client:  batch.NewFromConfig(cfg),
	}, nil
}

// List returns all Batch job definitions.
func (d *JobDefinitionDAO) List(ctx context.Context) ([]dao.Resource, error) {
	// Only list ACTIVE job definitions
	status := "ACTIVE"
	defs, err := appaws.Paginate(ctx, func(token *string) ([]types.JobDefinition, *string, error) {
		output, err := d.client.DescribeJobDefinitions(ctx, &batch.DescribeJobDefinitionsInput{
			Status:    &status,
			NextToken: token,
		})
		if err != nil {
			return nil, nil, apperrors.Wrap(err, "describe batch job definitions")
		}
		return output.JobDefinitions, output.NextToken, nil
	})
	if err != nil {
		return nil, err
	}

	resources := make([]dao.Resource, len(defs))
	for i, def := range defs {
		resources[i] = NewJobDefinitionResource(def)
	}
	return resources, nil
}

// Get returns a specific job definition.
func (d *JobDefinitionDAO) Get(ctx context.Context, id string) (dao.Resource, error) {
	output, err := d.client.DescribeJobDefinitions(ctx, &batch.DescribeJobDefinitionsInput{
		JobDefinitions: []string{id},
	})
	if err != nil {
		return nil, apperrors.Wrap(err, "describe batch job definition")
	}
	if len(output.JobDefinitions) == 0 {
		return nil, fmt.Errorf("job definition not found: %s", id)
	}
	return NewJobDefinitionResource(output.JobDefinitions[0]), nil
}

// Delete deregisters a Batch job definition.
func (d *JobDefinitionDAO) Delete(ctx context.Context, id string) error {
	_, err := d.client.DeregisterJobDefinition(ctx, &batch.DeregisterJobDefinitionInput{
		JobDefinition: &id,
	})
	if err != nil {
		return apperrors.Wrap(err, "deregister batch job definition")
	}
	return nil
}

// JobDefinitionResource wraps a Batch job definition.
type JobDefinitionResource struct {
	dao.BaseResource
	Def *types.JobDefinition
}

// NewJobDefinitionResource creates a new JobDefinitionResource.
func NewJobDefinitionResource(def types.JobDefinition) *JobDefinitionResource {
	// Extract name from ARN or use name:revision format
	name := appaws.Str(def.JobDefinitionName)
	if def.Revision != nil && *def.Revision > 0 {
		name = fmt.Sprintf("%s:%d", name, *def.Revision)
	}
	return &JobDefinitionResource{
		BaseResource: dao.BaseResource{
			ID:  name,
			ARN: appaws.Str(def.JobDefinitionArn),
		},
		Def: &def,
	}
}

// Name returns the job definition name.
func (r *JobDefinitionResource) Name() string {
	if r.Def != nil && r.Def.JobDefinitionName != nil {
		return *r.Def.JobDefinitionName
	}
	return ""
}

// Revision returns the job definition revision.
func (r *JobDefinitionResource) Revision() int32 {
	if r.Def != nil && r.Def.Revision != nil {
		return *r.Def.Revision
	}
	return 0
}

// Type returns the job definition type.
func (r *JobDefinitionResource) Type() string {
	if r.Def != nil && r.Def.Type != nil {
		return *r.Def.Type
	}
	return ""
}

// Status returns the job definition status.
func (r *JobDefinitionResource) Status() string {
	if r.Def != nil && r.Def.Status != nil {
		return *r.Def.Status
	}
	return ""
}

// ContainerImage returns the container image.
func (r *JobDefinitionResource) ContainerImage() string {
	if r.Def != nil && r.Def.ContainerProperties != nil && r.Def.ContainerProperties.Image != nil {
		// Shorten the image name if it's from ECR
		image := *r.Def.ContainerProperties.Image
		if idx := strings.LastIndex(image, "/"); idx != -1 {
			return image[idx+1:]
		}
		return image
	}
	return ""
}
