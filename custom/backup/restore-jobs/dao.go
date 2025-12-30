package restorejobs

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/backup"
	"github.com/aws/aws-sdk-go-v2/service/backup/types"

	appaws "github.com/clawscli/claws/internal/aws"
	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/render"
)

// RestoreJobDAO provides data access for AWS Backup restore jobs
type RestoreJobDAO struct {
	dao.BaseDAO
	client *backup.Client
}

// NewRestoreJobDAO creates a new RestoreJobDAO
func NewRestoreJobDAO(ctx context.Context) (dao.DAO, error) {
	cfg, err := appaws.NewConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("new backup/restore-jobs dao: %w", err)
	}
	return &RestoreJobDAO{
		BaseDAO: dao.NewBaseDAO("backup", "restore-jobs"),
		client:  backup.NewFromConfig(cfg),
	}, nil
}

// List returns restore jobs (first page)
func (d *RestoreJobDAO) List(ctx context.Context) ([]dao.Resource, error) {
	resources, _, err := d.ListPage(ctx, 100, "")
	return resources, err
}

// ListPage returns a page of restore jobs
func (d *RestoreJobDAO) ListPage(ctx context.Context, pageSize int, pageToken string) ([]dao.Resource, string, error) {
	maxResults := int32(pageSize)
	if maxResults > 1000 {
		maxResults = 1000
	}

	input := &backup.ListRestoreJobsInput{
		MaxResults: &maxResults,
	}
	if pageToken != "" {
		input.NextToken = &pageToken
	}

	output, err := d.client.ListRestoreJobs(ctx, input)
	if err != nil {
		return nil, "", fmt.Errorf("list restore jobs: %w", err)
	}

	resources := make([]dao.Resource, len(output.RestoreJobs))
	for i, job := range output.RestoreJobs {
		resources[i] = NewRestoreJobResource(job)
	}

	nextToken := ""
	if output.NextToken != nil {
		nextToken = *output.NextToken
	}

	return resources, nextToken, nil
}

// Get returns a specific restore job
func (d *RestoreJobDAO) Get(ctx context.Context, jobId string) (dao.Resource, error) {
	output, err := d.client.DescribeRestoreJob(ctx, &backup.DescribeRestoreJobInput{
		RestoreJobId: &jobId,
	})
	if err != nil {
		return nil, fmt.Errorf("describe restore job %s: %w", jobId, err)
	}

	return NewRestoreJobResourceFromDetail(output), nil
}

// Delete is not supported for restore jobs
func (d *RestoreJobDAO) Delete(ctx context.Context, id string) error {
	return fmt.Errorf("delete not supported for restore jobs")
}

// Supports returns supported operations
func (d *RestoreJobDAO) Supports(op dao.Operation) bool {
	switch op {
	case dao.OpList, dao.OpGet:
		return true
	default:
		return false
	}
}

// RestoreJobResource represents an AWS Backup restore job
type RestoreJobResource struct {
	dao.BaseResource
	Job    *types.RestoreJobsListMember
	Detail *backup.DescribeRestoreJobOutput
}

// NewRestoreJobResource creates a new RestoreJobResource from list
func NewRestoreJobResource(job types.RestoreJobsListMember) *RestoreJobResource {
	id := appaws.Str(job.RestoreJobId)

	return &RestoreJobResource{
		BaseResource: dao.BaseResource{
			ID:   id,
			Name: id,
			ARN:  "",
			Tags: make(map[string]string),
			Data: job,
		},
		Job: &job,
	}
}

// NewRestoreJobResourceFromDetail creates a new RestoreJobResource from detail
func NewRestoreJobResourceFromDetail(detail *backup.DescribeRestoreJobOutput) *RestoreJobResource {
	id := appaws.Str(detail.RestoreJobId)

	return &RestoreJobResource{
		BaseResource: dao.BaseResource{
			ID:   id,
			Name: id,
			ARN:  "",
			Tags: make(map[string]string),
			Data: detail,
		},
		Detail: detail,
	}
}

// JobId returns the restore job ID
func (r *RestoreJobResource) JobId() string {
	if r.Job != nil {
		return appaws.Str(r.Job.RestoreJobId)
	}
	if r.Detail != nil {
		return appaws.Str(r.Detail.RestoreJobId)
	}
	return ""
}

// Status returns the job status
func (r *RestoreJobResource) Status() string {
	if r.Job != nil {
		return string(r.Job.Status)
	}
	if r.Detail != nil {
		return string(r.Detail.Status)
	}
	return ""
}

// StatusMessage returns the status message
func (r *RestoreJobResource) StatusMessage() string {
	if r.Job != nil {
		return appaws.Str(r.Job.StatusMessage)
	}
	if r.Detail != nil {
		return appaws.Str(r.Detail.StatusMessage)
	}
	return ""
}

// ResourceType returns the resource type
func (r *RestoreJobResource) ResourceType() string {
	if r.Job != nil {
		return appaws.Str(r.Job.ResourceType)
	}
	if r.Detail != nil {
		return appaws.Str(r.Detail.ResourceType)
	}
	return ""
}

// RecoveryPointArn returns the recovery point ARN
func (r *RestoreJobResource) RecoveryPointArn() string {
	if r.Job != nil {
		return appaws.Str(r.Job.RecoveryPointArn)
	}
	if r.Detail != nil {
		return appaws.Str(r.Detail.RecoveryPointArn)
	}
	return ""
}

// CreatedResourceArn returns the created resource ARN
func (r *RestoreJobResource) CreatedResourceArn() string {
	if r.Job != nil {
		return appaws.Str(r.Job.CreatedResourceArn)
	}
	if r.Detail != nil {
		return appaws.Str(r.Detail.CreatedResourceArn)
	}
	return ""
}

// BackupSizeBytes returns the backup size in bytes
func (r *RestoreJobResource) BackupSizeBytes() int64 {
	if r.Job != nil && r.Job.BackupSizeInBytes != nil {
		return *r.Job.BackupSizeInBytes
	}
	if r.Detail != nil && r.Detail.BackupSizeInBytes != nil {
		return *r.Detail.BackupSizeInBytes
	}
	return 0
}

// BackupSizeFormatted returns the formatted backup size
func (r *RestoreJobResource) BackupSizeFormatted() string {
	bytes := r.BackupSizeBytes()
	if bytes == 0 {
		return "-"
	}
	return render.FormatSize(bytes)
}

// PercentDone returns the completion percentage
func (r *RestoreJobResource) PercentDone() string {
	if r.Job != nil && r.Job.PercentDone != nil {
		return appaws.Str(r.Job.PercentDone)
	}
	if r.Detail != nil && r.Detail.PercentDone != nil {
		return appaws.Str(r.Detail.PercentDone)
	}
	return ""
}

// IamRoleArn returns the IAM role ARN
func (r *RestoreJobResource) IamRoleArn() string {
	if r.Job != nil {
		return appaws.Str(r.Job.IamRoleArn)
	}
	if r.Detail != nil {
		return appaws.Str(r.Detail.IamRoleArn)
	}
	return ""
}

// ExpectedCompletionTimeMinutes returns expected completion time
func (r *RestoreJobResource) ExpectedCompletionTimeMinutes() int64 {
	if r.Job != nil && r.Job.ExpectedCompletionTimeMinutes != nil {
		return *r.Job.ExpectedCompletionTimeMinutes
	}
	if r.Detail != nil && r.Detail.ExpectedCompletionTimeMinutes != nil {
		return *r.Detail.ExpectedCompletionTimeMinutes
	}
	return 0
}

// ValidationStatus returns the validation status
func (r *RestoreJobResource) ValidationStatus() string {
	if r.Detail != nil {
		return string(r.Detail.ValidationStatus)
	}
	return ""
}

// ValidationStatusMessage returns the validation status message
func (r *RestoreJobResource) ValidationStatusMessage() string {
	if r.Detail != nil {
		return appaws.Str(r.Detail.ValidationStatusMessage)
	}
	return ""
}

// DeletionStatus returns the deletion status
func (r *RestoreJobResource) DeletionStatus() string {
	if r.Detail != nil {
		return string(r.Detail.DeletionStatus)
	}
	return ""
}

// DeletionStatusMessage returns the deletion status message
func (r *RestoreJobResource) DeletionStatusMessage() string {
	if r.Detail != nil {
		return appaws.Str(r.Detail.DeletionStatusMessage)
	}
	return ""
}

// RecoveryPointCreationDate returns the recovery point creation date
func (r *RestoreJobResource) RecoveryPointCreationDate() string {
	if r.Detail != nil && r.Detail.RecoveryPointCreationDate != nil {
		return r.Detail.RecoveryPointCreationDate.Format("2006-01-02 15:04:05")
	}
	return ""
}

// CreationDate returns the creation date
func (r *RestoreJobResource) CreationDate() string {
	if r.Job != nil && r.Job.CreationDate != nil {
		return r.Job.CreationDate.Format("2006-01-02 15:04:05")
	}
	if r.Detail != nil && r.Detail.CreationDate != nil {
		return r.Detail.CreationDate.Format("2006-01-02 15:04:05")
	}
	return ""
}

// CreationDateT returns the creation date as time.Time
func (r *RestoreJobResource) CreationDateT() *time.Time {
	if r.Job != nil {
		return r.Job.CreationDate
	}
	if r.Detail != nil {
		return r.Detail.CreationDate
	}
	return nil
}

// CompletionDate returns the completion date
func (r *RestoreJobResource) CompletionDate() string {
	if r.Job != nil && r.Job.CompletionDate != nil {
		return r.Job.CompletionDate.Format("2006-01-02 15:04:05")
	}
	if r.Detail != nil && r.Detail.CompletionDate != nil {
		return r.Detail.CompletionDate.Format("2006-01-02 15:04:05")
	}
	return ""
}
