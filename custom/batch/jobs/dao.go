package jobs

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/batch"
	"github.com/aws/aws-sdk-go-v2/service/batch/types"

	appaws "github.com/clawscli/claws/internal/aws"
	"github.com/clawscli/claws/internal/dao"
)

// JobDAO provides data access for Batch jobs.
type JobDAO struct {
	dao.BaseDAO
	client *batch.Client
}

// NewJobDAO creates a new JobDAO.
func NewJobDAO(ctx context.Context) (dao.DAO, error) {
	cfg, err := appaws.NewConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("new batch/jobs dao: %w", err)
	}
	return &JobDAO{
		BaseDAO: dao.NewBaseDAO("batch", "jobs"),
		client:  batch.NewFromConfig(cfg),
	}, nil
}

// List returns jobs for the specified job queue.
func (d *JobDAO) List(ctx context.Context) ([]dao.Resource, error) {
	jobQueue := dao.GetFilterFromContext(ctx, "JobQueue")
	if jobQueue == "" {
		return nil, fmt.Errorf("job queue filter required")
	}

	// List jobs in various statuses
	statuses := []types.JobStatus{
		types.JobStatusSubmitted,
		types.JobStatusPending,
		types.JobStatusRunnable,
		types.JobStatusStarting,
		types.JobStatusRunning,
		types.JobStatusSucceeded,
		types.JobStatusFailed,
	}

	var allJobs []types.JobSummary
	for _, status := range statuses {
		jobs, err := appaws.Paginate(ctx, func(token *string) ([]types.JobSummary, *string, error) {
			output, err := d.client.ListJobs(ctx, &batch.ListJobsInput{
				JobQueue:  &jobQueue,
				JobStatus: status,
				NextToken: token,
			})
			if err != nil {
				return nil, nil, fmt.Errorf("list batch jobs: %w", err)
			}
			return output.JobSummaryList, output.NextToken, nil
		})
		if err != nil {
			return nil, err
		}
		allJobs = append(allJobs, jobs...)
	}

	resources := make([]dao.Resource, len(allJobs))
	for i, job := range allJobs {
		resources[i] = NewJobResource(job)
	}
	return resources, nil
}

// Get returns a specific job.
func (d *JobDAO) Get(ctx context.Context, id string) (dao.Resource, error) {
	output, err := d.client.DescribeJobs(ctx, &batch.DescribeJobsInput{
		Jobs: []string{id},
	})
	if err != nil {
		return nil, fmt.Errorf("describe batch job: %w", err)
	}
	if len(output.Jobs) == 0 {
		return nil, fmt.Errorf("job not found: %s", id)
	}

	job := output.Jobs[0]
	platformCaps := make([]string, len(job.PlatformCapabilities))
	for i, pc := range job.PlatformCapabilities {
		platformCaps[i] = string(pc)
	}

	return &JobResource{
		BaseResource: dao.BaseResource{
			ID:  appaws.Str(job.JobId),
			ARN: appaws.Str(job.JobArn),
		},
		Job: &types.JobSummary{
			JobId:     job.JobId,
			JobName:   job.JobName,
			Status:    job.Status,
			CreatedAt: job.CreatedAt,
			StartedAt: job.StartedAt,
			StoppedAt: job.StoppedAt,
		},
		JobDefinition:        appaws.Str(job.JobDefinition),
		StatusReason:         appaws.Str(job.StatusReason),
		JobQueue:             appaws.Str(job.JobQueue),
		Container:            job.Container,
		Attempts:             job.Attempts,
		DependsOn:            job.DependsOn,
		Parameters:           job.Parameters,
		RetryStrategy:        job.RetryStrategy,
		Timeout:              job.Timeout,
		PlatformCapabilities: platformCaps,
		Tags:                 job.Tags,
	}, nil
}

// Delete terminates a Batch job.
func (d *JobDAO) Delete(ctx context.Context, id string) error {
	reason := "Terminated by claws"
	_, err := d.client.TerminateJob(ctx, &batch.TerminateJobInput{
		JobId:  &id,
		Reason: &reason,
	})
	if err != nil {
		return fmt.Errorf("terminate batch job: %w", err)
	}
	return nil
}

// JobResource wraps a Batch job.
type JobResource struct {
	dao.BaseResource
	Job                  *types.JobSummary
	JobDefinition        string
	StatusReason         string
	JobQueue             string
	Container            *types.ContainerDetail
	Attempts             []types.AttemptDetail
	DependsOn            []types.JobDependency
	Parameters           map[string]string
	RetryStrategy        *types.RetryStrategy
	Timeout              *types.JobTimeout
	PlatformCapabilities []string
	Tags                 map[string]string
}

// NewJobResource creates a new JobResource.
func NewJobResource(job types.JobSummary) *JobResource {
	return &JobResource{
		BaseResource: dao.BaseResource{
			ID:  appaws.Str(job.JobId),
			ARN: appaws.Str(job.JobArn),
		},
		Job: &job,
	}
}

// Name returns the job name.
func (r *JobResource) Name() string {
	if r.Job != nil && r.Job.JobName != nil {
		return *r.Job.JobName
	}
	return ""
}

// Status returns the job status.
func (r *JobResource) Status() string {
	if r.Job != nil {
		return string(r.Job.Status)
	}
	return ""
}

// CreatedAt returns when the job was created.
func (r *JobResource) CreatedAt() *time.Time {
	if r.Job != nil && r.Job.CreatedAt != nil {
		t := time.UnixMilli(*r.Job.CreatedAt)
		return &t
	}
	return nil
}

// StartedAt returns when the job started.
func (r *JobResource) StartedAt() *time.Time {
	if r.Job != nil && r.Job.StartedAt != nil {
		t := time.UnixMilli(*r.Job.StartedAt)
		return &t
	}
	return nil
}

// StoppedAt returns when the job stopped.
func (r *JobResource) StoppedAt() *time.Time {
	if r.Job != nil && r.Job.StoppedAt != nil {
		t := time.UnixMilli(*r.Job.StoppedAt)
		return &t
	}
	return nil
}

// GetJobQueue returns the job queue.
func (r *JobResource) GetJobQueue() string {
	return r.JobQueue
}

// GetContainer returns the container detail.
func (r *JobResource) GetContainer() *types.ContainerDetail {
	return r.Container
}

// GetAttempts returns the job attempts.
func (r *JobResource) GetAttempts() []types.AttemptDetail {
	return r.Attempts
}

// GetDependsOn returns the job dependencies.
func (r *JobResource) GetDependsOn() []types.JobDependency {
	return r.DependsOn
}

// GetParameters returns the job parameters.
func (r *JobResource) GetParameters() map[string]string {
	return r.Parameters
}

// GetRetryStrategy returns the retry strategy.
func (r *JobResource) GetRetryStrategy() *types.RetryStrategy {
	return r.RetryStrategy
}

// GetTimeout returns the job timeout.
func (r *JobResource) GetTimeout() *types.JobTimeout {
	return r.Timeout
}

// GetPlatformCapabilities returns the platform capabilities.
func (r *JobResource) GetPlatformCapabilities() []string {
	return r.PlatformCapabilities
}

// GetTags returns the job tags.
func (r *JobResource) GetTags() map[string]string {
	return r.Tags
}
