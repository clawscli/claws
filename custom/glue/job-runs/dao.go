package jobruns

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/glue"
	"github.com/aws/aws-sdk-go-v2/service/glue/types"

	appaws "github.com/clawscli/claws/internal/aws"
	"github.com/clawscli/claws/internal/dao"
	apperrors "github.com/clawscli/claws/internal/errors"
)

// JobRunDAO provides data access for Glue job runs.
type JobRunDAO struct {
	dao.BaseDAO
	client *glue.Client
}

// NewJobRunDAO creates a new JobRunDAO.
func NewJobRunDAO(ctx context.Context) (dao.DAO, error) {
	cfg, err := appaws.NewConfig(ctx)
	if err != nil {
		return nil, apperrors.Wrap(err, "new glue/jobruns dao")
	}
	return &JobRunDAO{
		BaseDAO: dao.NewBaseDAO("glue", "job-runs"),
		client:  glue.NewFromConfig(cfg),
	}, nil
}

// List returns job runs (first page only for backwards compatibility).
// For paginated access, use ListPage instead.
func (d *JobRunDAO) List(ctx context.Context) ([]dao.Resource, error) {
	resources, _, err := d.ListPage(ctx, 100, "")
	return resources, err
}

// ListPage returns a page of Glue job runs.
// Implements dao.PaginatedDAO interface.
func (d *JobRunDAO) ListPage(ctx context.Context, pageSize int, pageToken string) ([]dao.Resource, string, error) {
	jobName := dao.GetFilterFromContext(ctx, "JobName")
	if jobName == "" {
		return nil, "", fmt.Errorf("job name filter required")
	}

	maxResults := int32(pageSize)
	if maxResults > 200 {
		maxResults = 200 // AWS API max
	}

	input := &glue.GetJobRunsInput{
		JobName:    &jobName,
		MaxResults: &maxResults,
	}
	if pageToken != "" {
		input.NextToken = &pageToken
	}

	output, err := d.client.GetJobRuns(ctx, input)
	if err != nil {
		return nil, "", apperrors.Wrap(err, "get glue job runs")
	}

	resources := make([]dao.Resource, len(output.JobRuns))
	for i, run := range output.JobRuns {
		resources[i] = NewJobRunResource(run)
	}

	nextToken := ""
	if output.NextToken != nil {
		nextToken = *output.NextToken
	}

	return resources, nextToken, nil
}

// Get returns a specific job run by ID.
func (d *JobRunDAO) Get(ctx context.Context, id string) (dao.Resource, error) {
	jobName := dao.GetFilterFromContext(ctx, "JobName")
	if jobName == "" {
		return nil, fmt.Errorf("job name filter required")
	}

	output, err := d.client.GetJobRun(ctx, &glue.GetJobRunInput{
		JobName: &jobName,
		RunId:   &id,
	})
	if err != nil {
		return nil, apperrors.Wrapf(err, "get glue job run %s", id)
	}
	return NewJobRunResource(*output.JobRun), nil
}

// Delete is not supported for job runs.
func (d *JobRunDAO) Delete(ctx context.Context, id string) error {
	return fmt.Errorf("delete not supported for job runs")
}

// JobRunResource wraps a Glue job run.
type JobRunResource struct {
	dao.BaseResource
	Item types.JobRun
}

// NewJobRunResource creates a new JobRunResource.
func NewJobRunResource(run types.JobRun) *JobRunResource {
	return &JobRunResource{
		BaseResource: dao.BaseResource{
			ID:  appaws.Str(run.Id),
			ARN: "",
		},
		Item: run,
	}
}

// JobName returns the job name.
func (r *JobRunResource) JobName() string {
	return appaws.Str(r.Item.JobName)
}

// JobRunState returns the run state.
func (r *JobRunResource) JobRunState() string {
	return string(r.Item.JobRunState)
}

// StartedOn returns when the run started.
func (r *JobRunResource) StartedOn() *time.Time {
	return r.Item.StartedOn
}

// CompletedOn returns when the run completed.
func (r *JobRunResource) CompletedOn() *time.Time {
	return r.Item.CompletedOn
}

// ExecutionTime returns the execution time in seconds.
func (r *JobRunResource) ExecutionTime() int32 {
	return r.Item.ExecutionTime
}

// ErrorMessage returns the error message if any.
func (r *JobRunResource) ErrorMessage() string {
	return appaws.Str(r.Item.ErrorMessage)
}

// Attempt returns the attempt number.
func (r *JobRunResource) Attempt() int32 {
	return r.Item.Attempt
}

// MaxCapacity returns the max DPU capacity.
func (r *JobRunResource) MaxCapacity() float64 {
	return appaws.Float64(r.Item.MaxCapacity)
}

// WorkerType returns the worker type.
func (r *JobRunResource) WorkerType() string {
	return string(r.Item.WorkerType)
}

// NumberOfWorkers returns the number of workers.
func (r *JobRunResource) NumberOfWorkers() int32 {
	return appaws.Int32(r.Item.NumberOfWorkers)
}

// GlueVersion returns the Glue version.
func (r *JobRunResource) GlueVersion() string {
	return appaws.Str(r.Item.GlueVersion)
}
