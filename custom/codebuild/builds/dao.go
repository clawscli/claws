package builds

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

// BuildDAO provides data access for CodeBuild builds
type BuildDAO struct {
	dao.BaseDAO
	client *codebuild.Client
}

// NewBuildDAO creates a new BuildDAO
func NewBuildDAO(ctx context.Context) (dao.DAO, error) {
	cfg, err := appaws.NewConfig(ctx)
	if err != nil {
		return nil, apperrors.Wrap(err, "new "+ServiceResourcePath+" dao")
	}
	return &BuildDAO{
		BaseDAO: dao.NewBaseDAO("codebuild", "builds"),
		client:  codebuild.NewFromConfig(cfg),
	}, nil
}

// List returns builds (first page only for backwards compatibility).
// For paginated access, use ListPage instead.
func (d *BuildDAO) List(ctx context.Context) ([]dao.Resource, error) {
	resources, _, err := d.ListPage(ctx, 100, "")
	return resources, err
}

// ListPage returns a page of CodeBuild builds.
// Implements dao.PaginatedDAO interface.
func (d *BuildDAO) ListPage(ctx context.Context, pageSize int, pageToken string) ([]dao.Resource, string, error) {
	// Get project name from filter context
	projectName := dao.GetFilterFromContext(ctx, "ProjectName")
	if projectName == "" {
		return nil, "", fmt.Errorf("project name filter required")
	}

	// List build IDs for the project
	listInput := &codebuild.ListBuildsForProjectInput{
		ProjectName: &projectName,
	}
	if pageToken != "" {
		listInput.NextToken = &pageToken
	}

	listOutput, err := d.client.ListBuildsForProject(ctx, listInput)
	if err != nil {
		return nil, "", apperrors.Wrap(err, "list builds")
	}

	if len(listOutput.Ids) == 0 {
		return []dao.Resource{}, "", nil
	}

	// Limit IDs to pageSize
	ids := listOutput.Ids
	if len(ids) > pageSize {
		ids = ids[:pageSize]
	}

	// Get build details
	batchInput := &codebuild.BatchGetBuildsInput{
		Ids: ids,
	}

	batchOutput, err := d.client.BatchGetBuilds(ctx, batchInput)
	if err != nil {
		return nil, "", apperrors.Wrap(err, "batch get builds")
	}

	resources := make([]dao.Resource, 0, len(batchOutput.Builds))
	for _, build := range batchOutput.Builds {
		resources = append(resources, NewBuildResource(build, projectName))
	}

	nextToken := ""
	if listOutput.NextToken != nil {
		nextToken = *listOutput.NextToken
	}

	return resources, nextToken, nil
}

// Get returns a specific build
func (d *BuildDAO) Get(ctx context.Context, buildId string) (dao.Resource, error) {
	projectName := dao.GetFilterFromContext(ctx, "ProjectName")
	if projectName == "" {
		return nil, fmt.Errorf("project name filter required")
	}

	input := &codebuild.BatchGetBuildsInput{
		Ids: []string{buildId},
	}

	output, err := d.client.BatchGetBuilds(ctx, input)
	if err != nil {
		return nil, apperrors.Wrapf(err, "get build %s", buildId)
	}

	if len(output.Builds) == 0 {
		return nil, fmt.Errorf("build %s not found", buildId)
	}

	return NewBuildResource(output.Builds[0], projectName), nil
}

// Delete stops a build
func (d *BuildDAO) Delete(ctx context.Context, buildId string) error {
	_, err := d.client.StopBuild(ctx, &codebuild.StopBuildInput{
		Id: &buildId,
	})
	if err != nil {
		return apperrors.Wrapf(err, "stop build %s", buildId)
	}
	return nil
}

// Supports returns supported operations
func (d *BuildDAO) Supports(op dao.Operation) bool {
	switch op {
	case dao.OpList, dao.OpGet, dao.OpDelete:
		return true
	default:
		return false
	}
}

// BuildResource represents a CodeBuild build
type BuildResource struct {
	dao.BaseResource
	Build       types.Build
	ProjectName string
}

// NewBuildResource creates a new BuildResource
func NewBuildResource(build types.Build, projectName string) *BuildResource {
	id := appaws.Str(build.Id)
	arn := appaws.Str(build.Arn)

	return &BuildResource{
		BaseResource: dao.BaseResource{
			ID:   id,
			Name: id,
			ARN:  arn,
			Tags: make(map[string]string),
			Data: build,
		},
		Build:       build,
		ProjectName: projectName,
	}
}

// BuildId returns the build ID
func (r *BuildResource) BuildId() string {
	return appaws.Str(r.Build.Id)
}

// BuildNumber returns the build number
func (r *BuildResource) BuildNumber() int64 {
	if r.Build.BuildNumber != nil {
		return *r.Build.BuildNumber
	}
	return 0
}

// Status returns the build status
func (r *BuildResource) Status() string {
	return string(r.Build.BuildStatus)
}

// CurrentPhase returns the current build phase
func (r *BuildResource) CurrentPhase() string {
	return appaws.Str(r.Build.CurrentPhase)
}

// Initiator returns who initiated the build
func (r *BuildResource) Initiator() string {
	return appaws.Str(r.Build.Initiator)
}

// SourceVersion returns the source version
func (r *BuildResource) SourceVersion() string {
	return appaws.Str(r.Build.SourceVersion)
}

// ResolvedSourceVersion returns the resolved source version
func (r *BuildResource) ResolvedSourceVersion() string {
	return appaws.Str(r.Build.ResolvedSourceVersion)
}

// StartTime returns the start time
func (r *BuildResource) StartTime() string {
	if r.Build.StartTime != nil {
		return r.Build.StartTime.Format("2006-01-02 15:04:05")
	}
	return ""
}

// StartTimeT returns the start time as time.Time
func (r *BuildResource) StartTimeT() *time.Time {
	return r.Build.StartTime
}

// EndTime returns the end time
func (r *BuildResource) EndTime() string {
	if r.Build.EndTime != nil {
		return r.Build.EndTime.Format("2006-01-02 15:04:05")
	}
	return ""
}

// Duration returns the build duration
func (r *BuildResource) Duration() string {
	if r.Build.StartTime != nil && r.Build.EndTime != nil {
		d := r.Build.EndTime.Sub(*r.Build.StartTime)
		return d.Round(time.Second).String()
	}
	return ""
}

// EnvironmentImage returns the build environment image
func (r *BuildResource) EnvironmentImage() string {
	if r.Build.Environment != nil {
		return appaws.Str(r.Build.Environment.Image)
	}
	return ""
}

// ComputeType returns the compute type
func (r *BuildResource) ComputeType() string {
	if r.Build.Environment != nil {
		return string(r.Build.Environment.ComputeType)
	}
	return ""
}

// Phases returns the build phases
func (r *BuildResource) Phases() []types.BuildPhase {
	return r.Build.Phases
}

// Logs returns the CloudWatch logs info
func (r *BuildResource) LogsGroupName() string {
	if r.Build.Logs != nil {
		return appaws.Str(r.Build.Logs.GroupName)
	}
	return ""
}

// LogsStreamName returns the log stream name
func (r *BuildResource) LogsStreamName() string {
	if r.Build.Logs != nil {
		return appaws.Str(r.Build.Logs.StreamName)
	}
	return ""
}
