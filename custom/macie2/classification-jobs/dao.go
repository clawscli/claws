package classificationjobs

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/macie2"
	"github.com/aws/aws-sdk-go-v2/service/macie2/types"

	appaws "github.com/clawscli/claws/internal/aws"
	"github.com/clawscli/claws/internal/dao"
	apperrors "github.com/clawscli/claws/internal/errors"
)

// ClassificationJobDAO provides data access for Macie classification jobs.
type ClassificationJobDAO struct {
	dao.BaseDAO
	client *macie2.Client
}

// NewClassificationJobDAO creates a new ClassificationJobDAO.
func NewClassificationJobDAO(ctx context.Context) (dao.DAO, error) {
	cfg, err := appaws.NewConfig(ctx)
	if err != nil {
		return nil, apperrors.Wrap(err, "new macie/classificationjobs dao")
	}
	return &ClassificationJobDAO{
		BaseDAO: dao.NewBaseDAO("macie2", "classification-jobs"),
		client:  macie2.NewFromConfig(cfg),
	}, nil
}

// List returns all classification jobs.
func (d *ClassificationJobDAO) List(ctx context.Context) ([]dao.Resource, error) {
	jobs, err := appaws.Paginate(ctx, func(token *string) ([]types.JobSummary, *string, error) {
		output, err := d.client.ListClassificationJobs(ctx, &macie2.ListClassificationJobsInput{
			NextToken: token,
		})
		if err != nil {
			return nil, nil, apperrors.Wrap(err, "list macie classification jobs")
		}
		return output.Items, output.NextToken, nil
	})
	if err != nil {
		return nil, err
	}

	resources := make([]dao.Resource, len(jobs))
	for i, job := range jobs {
		resources[i] = NewClassificationJobResource(job)
	}
	return resources, nil
}

// Get returns a specific classification job.
func (d *ClassificationJobDAO) Get(ctx context.Context, id string) (dao.Resource, error) {
	output, err := d.client.DescribeClassificationJob(ctx, &macie2.DescribeClassificationJobInput{
		JobId: &id,
	})
	if err != nil {
		return nil, apperrors.Wrap(err, "describe macie classification job")
	}
	return &ClassificationJobResource{
		BaseResource: dao.BaseResource{
			ID:  appaws.Str(output.JobId),
			ARN: appaws.Str(output.JobArn),
		},
		Job: &types.JobSummary{
			JobId:     output.JobId,
			Name:      output.Name,
			JobStatus: output.JobStatus,
			JobType:   output.JobType,
			CreatedAt: output.CreatedAt,
		},
	}, nil
}

// Delete cancels a classification job.
func (d *ClassificationJobDAO) Delete(ctx context.Context, id string) error {
	status := types.JobStatusCancelled
	_, err := d.client.UpdateClassificationJob(ctx, &macie2.UpdateClassificationJobInput{
		JobId:     &id,
		JobStatus: status,
	})
	if err != nil {
		return apperrors.Wrap(err, "cancel macie classification job")
	}
	return nil
}

// ClassificationJobResource wraps a Macie classification job.
type ClassificationJobResource struct {
	dao.BaseResource
	Job *types.JobSummary
}

// NewClassificationJobResource creates a new ClassificationJobResource.
func NewClassificationJobResource(job types.JobSummary) *ClassificationJobResource {
	return &ClassificationJobResource{
		BaseResource: dao.BaseResource{
			ID:  appaws.Str(job.JobId),
			ARN: "",
		},
		Job: &job,
	}
}

// Name returns the job name.
func (r *ClassificationJobResource) Name() string {
	if r.Job != nil && r.Job.Name != nil {
		return *r.Job.Name
	}
	return ""
}

// Status returns the job status.
func (r *ClassificationJobResource) Status() string {
	if r.Job != nil {
		return string(r.Job.JobStatus)
	}
	return ""
}

// JobType returns the job type.
func (r *ClassificationJobResource) JobType() string {
	if r.Job != nil {
		return string(r.Job.JobType)
	}
	return ""
}

// CreatedAt returns when the job was created.
func (r *ClassificationJobResource) CreatedAt() *time.Time {
	if r.Job != nil {
		return r.Job.CreatedAt
	}
	return nil
}
