package jobs

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/glue"
	"github.com/aws/aws-sdk-go-v2/service/glue/types"

	appaws "github.com/clawscli/claws/internal/aws"
	"github.com/clawscli/claws/internal/dao"
	apperrors "github.com/clawscli/claws/internal/errors"
)

// JobDAO provides data access for Glue jobs.
type JobDAO struct {
	dao.BaseDAO
	client *glue.Client
}

// NewJobDAO creates a new JobDAO.
func NewJobDAO(ctx context.Context) (dao.DAO, error) {
	cfg, err := appaws.NewConfig(ctx)
	if err != nil {
		return nil, apperrors.Wrap(err, "new "+ServiceResourcePath+" dao")
	}
	return &JobDAO{
		BaseDAO: dao.NewBaseDAO("glue", "jobs"),
		client:  glue.NewFromConfig(cfg),
	}, nil
}

// List returns all Glue jobs.
func (d *JobDAO) List(ctx context.Context) ([]dao.Resource, error) {
	jobs, err := appaws.Paginate(ctx, func(token *string) ([]types.Job, *string, error) {
		output, err := d.client.GetJobs(ctx, &glue.GetJobsInput{
			NextToken: token,
		})
		if err != nil {
			return nil, nil, apperrors.Wrap(err, "get glue jobs")
		}
		return output.Jobs, output.NextToken, nil
	})
	if err != nil {
		return nil, err
	}

	resources := make([]dao.Resource, len(jobs))
	for i, job := range jobs {
		resources[i] = NewJobResource(job)
	}
	return resources, nil
}

// Get returns a specific Glue job by name.
func (d *JobDAO) Get(ctx context.Context, id string) (dao.Resource, error) {
	output, err := d.client.GetJob(ctx, &glue.GetJobInput{
		JobName: &id,
	})
	if err != nil {
		return nil, apperrors.Wrapf(err, "get glue job %s", id)
	}
	return NewJobResource(*output.Job), nil
}

// Delete deletes a Glue job by name.
func (d *JobDAO) Delete(ctx context.Context, id string) error {
	_, err := d.client.DeleteJob(ctx, &glue.DeleteJobInput{
		JobName: &id,
	})
	if err != nil {
		return apperrors.Wrapf(err, "delete glue job %s", id)
	}
	return nil
}

// JobResource wraps a Glue job.
type JobResource struct {
	dao.BaseResource
	Item types.Job
}

// NewJobResource creates a new JobResource.
func NewJobResource(job types.Job) *JobResource {
	return &JobResource{
		BaseResource: dao.BaseResource{
			ID:   appaws.Str(job.Name),
			ARN:  "", // Glue jobs don't have ARN in the response
			Data: job,
		},
		Item: job,
	}
}

// Name returns the job name.
func (r *JobResource) Name() string {
	return appaws.Str(r.Item.Name)
}

// Description returns the job description.
func (r *JobResource) Description() string {
	return appaws.Str(r.Item.Description)
}

// Role returns the IAM role.
func (r *JobResource) Role() string {
	return appaws.Str(r.Item.Role)
}

// GlueVersion returns the Glue version.
func (r *JobResource) GlueVersion() string {
	return appaws.Str(r.Item.GlueVersion)
}

// WorkerType returns the worker type.
func (r *JobResource) WorkerType() string {
	return string(r.Item.WorkerType)
}

// NumberOfWorkers returns the number of workers.
func (r *JobResource) NumberOfWorkers() int32 {
	return appaws.Int32(r.Item.NumberOfWorkers)
}

// MaxRetries returns the max retries.
func (r *JobResource) MaxRetries() int32 {
	return r.Item.MaxRetries
}

// Timeout returns the timeout in minutes.
func (r *JobResource) Timeout() int32 {
	return appaws.Int32(r.Item.Timeout)
}

// CreatedOn returns when the job was created.
func (r *JobResource) CreatedOn() *time.Time {
	return r.Item.CreatedOn
}

// LastModifiedOn returns when the job was last modified.
func (r *JobResource) LastModifiedOn() *time.Time {
	return r.Item.LastModifiedOn
}

// Command returns the job command.
func (r *JobResource) Command() *types.JobCommand {
	return r.Item.Command
}

// ExecutionClass returns the execution class.
func (r *JobResource) ExecutionClass() string {
	return string(r.Item.ExecutionClass)
}

// MaxCapacity returns the max capacity (DPU).
func (r *JobResource) MaxCapacity() float64 {
	if r.Item.MaxCapacity != nil {
		return *r.Item.MaxCapacity
	}
	return 0
}

// Connections returns the connection names.
func (r *JobResource) Connections() []string {
	if r.Item.Connections != nil {
		return r.Item.Connections.Connections
	}
	return nil
}

// DefaultArguments returns the default arguments.
func (r *JobResource) DefaultArguments() map[string]string {
	return r.Item.DefaultArguments
}

// SecurityConfiguration returns the security configuration.
func (r *JobResource) SecurityConfiguration() string {
	return appaws.Str(r.Item.SecurityConfiguration)
}

// JobMode returns the job mode (SCRIPT, NOTEBOOK, VISUAL).
func (r *JobResource) JobMode() string {
	return string(r.Item.JobMode)
}
