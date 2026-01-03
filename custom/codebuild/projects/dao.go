package projects

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/codebuild"
	"github.com/aws/aws-sdk-go-v2/service/codebuild/types"

	appaws "github.com/clawscli/claws/internal/aws"
	"github.com/clawscli/claws/internal/dao"
	apperrors "github.com/clawscli/claws/internal/errors"
)

// ProjectDAO provides data access for CodeBuild projects
type ProjectDAO struct {
	dao.BaseDAO
	client *codebuild.Client
}

// NewProjectDAO creates a new ProjectDAO
func NewProjectDAO(ctx context.Context) (dao.DAO, error) {
	cfg, err := appaws.NewConfig(ctx)
	if err != nil {
		return nil, apperrors.Wrap(err, "new "+ServiceResourcePath+" dao")
	}
	return &ProjectDAO{
		BaseDAO: dao.NewBaseDAO("codebuild", "projects"),
		client:  codebuild.NewFromConfig(cfg),
	}, nil
}

// List returns all CodeBuild projects
func (d *ProjectDAO) List(ctx context.Context) ([]dao.Resource, error) {
	var resources []dao.Resource
	var nextToken *string

	for {
		// First, get project names
		listInput := &codebuild.ListProjectsInput{
			NextToken: nextToken,
		}

		listOutput, err := d.client.ListProjects(ctx, listInput)
		if err != nil {
			return nil, apperrors.Wrap(err, "list projects")
		}

		if len(listOutput.Projects) == 0 {
			break
		}

		// Then get project details using BatchGetProjects
		batchInput := &codebuild.BatchGetProjectsInput{
			Names: listOutput.Projects,
		}

		batchOutput, err := d.client.BatchGetProjects(ctx, batchInput)
		if err != nil {
			return nil, apperrors.Wrap(err, "batch get projects")
		}

		for _, project := range batchOutput.Projects {
			resources = append(resources, NewProjectResource(project))
		}

		if listOutput.NextToken == nil {
			break
		}
		nextToken = listOutput.NextToken
	}

	return resources, nil
}

// Get returns a specific CodeBuild project by name
func (d *ProjectDAO) Get(ctx context.Context, name string) (dao.Resource, error) {
	input := &codebuild.BatchGetProjectsInput{
		Names: []string{name},
	}

	output, err := d.client.BatchGetProjects(ctx, input)
	if err != nil {
		return nil, apperrors.Wrapf(err, "get project %s", name)
	}

	if len(output.Projects) == 0 {
		return nil, fmt.Errorf("project %s not found", name)
	}

	return NewProjectResource(output.Projects[0]), nil
}

// Delete deletes a CodeBuild project
func (d *ProjectDAO) Delete(ctx context.Context, name string) error {
	_, err := d.client.DeleteProject(ctx, &codebuild.DeleteProjectInput{
		Name: &name,
	})
	if err != nil {
		return apperrors.Wrapf(err, "delete project %s", name)
	}
	return nil
}

// ProjectResource represents a CodeBuild project
type ProjectResource struct {
	dao.BaseResource
	Project types.Project
}

// NewProjectResource creates a new ProjectResource
func NewProjectResource(project types.Project) *ProjectResource {
	name := appaws.Str(project.Name)
	arn := appaws.Str(project.Arn)

	tags := make(map[string]string)
	for _, tag := range project.Tags {
		if tag.Key != nil && tag.Value != nil {
			tags[*tag.Key] = *tag.Value
		}
	}

	return &ProjectResource{
		BaseResource: dao.BaseResource{
			ID:   name,
			Name: name,
			ARN:  arn,
			Tags: tags,
			Data: project,
		},
		Project: project,
	}
}

// ProjectName returns the project name
func (r *ProjectResource) ProjectName() string {
	return appaws.Str(r.Project.Name)
}

// Description returns the project description
func (r *ProjectResource) Description() string {
	return appaws.Str(r.Project.Description)
}

// SourceType returns the source type
func (r *ProjectResource) SourceType() string {
	if r.Project.Source != nil {
		return string(r.Project.Source.Type)
	}
	return ""
}

// SourceLocation returns the source location
func (r *ProjectResource) SourceLocation() string {
	if r.Project.Source != nil {
		return appaws.Str(r.Project.Source.Location)
	}
	return ""
}

// EnvironmentImage returns the build environment image
func (r *ProjectResource) EnvironmentImage() string {
	if r.Project.Environment != nil {
		return appaws.Str(r.Project.Environment.Image)
	}
	return ""
}

// EnvironmentType returns the build environment type
func (r *ProjectResource) EnvironmentType() string {
	if r.Project.Environment != nil {
		return string(r.Project.Environment.Type)
	}
	return ""
}

// ComputeType returns the build compute type
func (r *ProjectResource) ComputeType() string {
	if r.Project.Environment != nil {
		return string(r.Project.Environment.ComputeType)
	}
	return ""
}

// ServiceRole returns the service role ARN
func (r *ProjectResource) ServiceRole() string {
	return appaws.Str(r.Project.ServiceRole)
}

// TimeoutInMinutes returns the build timeout
func (r *ProjectResource) TimeoutInMinutes() int32 {
	if r.Project.TimeoutInMinutes != nil {
		return *r.Project.TimeoutInMinutes
	}
	return 0
}

// CreatedAt returns the creation date
func (r *ProjectResource) CreatedAt() string {
	if r.Project.Created != nil {
		return r.Project.Created.Format("2006-01-02 15:04:05")
	}
	return ""
}

// CreatedAtTime returns the creation date as time.Time
func (r *ProjectResource) CreatedAtTime() *time.Time {
	return r.Project.Created
}

// LastModified returns the last modified date
func (r *ProjectResource) LastModified() string {
	if r.Project.LastModified != nil {
		return r.Project.LastModified.Format("2006-01-02 15:04:05")
	}
	return ""
}

// Badge returns the badge information
func (r *ProjectResource) BadgeEnabled() bool {
	if r.Project.Badge != nil {
		return r.Project.Badge.BadgeEnabled
	}
	return false
}

// ConcurrentBuildLimit returns the concurrent build limit
func (r *ProjectResource) ConcurrentBuildLimit() int32 {
	if r.Project.ConcurrentBuildLimit != nil {
		return *r.Project.ConcurrentBuildLimit
	}
	return 0
}

// BuildBatchConfig returns whether batch builds are enabled
func (r *ProjectResource) BatchBuildEnabled() bool {
	return r.Project.BuildBatchConfig != nil
}
