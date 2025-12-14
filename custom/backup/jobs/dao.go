package jobs

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/backup"
	"github.com/aws/aws-sdk-go-v2/service/backup/types"
	appaws "github.com/clawscli/claws/internal/aws"
	"github.com/clawscli/claws/internal/dao"
)

// JobDAO provides data access for AWS Backup jobs
type JobDAO struct {
	dao.BaseDAO
	client *backup.Client
}

// NewJobDAO creates a new JobDAO
func NewJobDAO(ctx context.Context) (dao.DAO, error) {
	cfg, err := appaws.NewConfig(ctx)
	if err != nil {
		return nil, err
	}
	return &JobDAO{
		BaseDAO: dao.NewBaseDAO("backup", "jobs"),
		client:  backup.NewFromConfig(cfg),
	}, nil
}

// List returns backup jobs (first page only for backwards compatibility).
// For paginated access, use ListPage instead.
func (d *JobDAO) List(ctx context.Context) ([]dao.Resource, error) {
	resources, _, err := d.ListPage(ctx, 100, "")
	return resources, err
}

// ListPage returns a page of backup jobs.
// Implements dao.PaginatedDAO interface.
// Note: Jobs are filtered client-side by BackupPlanId, so actual count may be less than pageSize.
func (d *JobDAO) ListPage(ctx context.Context, pageSize int, pageToken string) ([]dao.Resource, string, error) {
	// Get backup plan ID from filter context
	backupPlanId := dao.GetFilterFromContext(ctx, "BackupPlanId")
	if backupPlanId == "" {
		return nil, "", fmt.Errorf("backup plan ID filter required")
	}

	maxResults := int32(pageSize)
	if maxResults > 1000 {
		maxResults = 1000 // AWS API max
	}

	input := &backup.ListBackupJobsInput{
		MaxResults: &maxResults,
	}
	if pageToken != "" {
		input.NextToken = &pageToken
	}

	output, err := d.client.ListBackupJobs(ctx, input)
	if err != nil {
		return nil, "", fmt.Errorf("list backup jobs: %w", err)
	}

	// Filter by backup plan ID (client-side filtering since API doesn't support it)
	resources := make([]dao.Resource, 0)
	for _, job := range output.BackupJobs {
		if job.CreatedBy != nil && job.CreatedBy.BackupPlanId != nil {
			if *job.CreatedBy.BackupPlanId == backupPlanId {
				resources = append(resources, NewJobResource(job, backupPlanId))
			}
		}
	}

	nextToken := ""
	if output.NextToken != nil {
		nextToken = *output.NextToken
	}

	return resources, nextToken, nil
}

// Get returns a specific backup job
func (d *JobDAO) Get(ctx context.Context, jobId string) (dao.Resource, error) {
	backupPlanId := dao.GetFilterFromContext(ctx, "BackupPlanId")

	input := &backup.DescribeBackupJobInput{
		BackupJobId: &jobId,
	}

	output, err := d.client.DescribeBackupJob(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("describe backup job %s: %w", jobId, err)
	}

	return NewJobResourceFromDetail(output, backupPlanId), nil
}

// Delete stops a backup job
func (d *JobDAO) Delete(ctx context.Context, jobId string) error {
	_, err := d.client.StopBackupJob(ctx, &backup.StopBackupJobInput{
		BackupJobId: &jobId,
	})
	if err != nil {
		return fmt.Errorf("stop backup job %s: %w", jobId, err)
	}
	return nil
}

// Supports returns supported operations
func (d *JobDAO) Supports(op dao.Operation) bool {
	switch op {
	case dao.OpList, dao.OpGet, dao.OpDelete:
		return true
	default:
		return false
	}
}

// JobResource represents an AWS Backup job
type JobResource struct {
	dao.BaseResource
	Job          *types.BackupJob
	Detail       *backup.DescribeBackupJobOutput
	BackupPlanId string
}

// NewJobResource creates a new JobResource from list
func NewJobResource(job types.BackupJob, backupPlanId string) *JobResource {
	id := appaws.Str(job.BackupJobId)

	return &JobResource{
		BaseResource: dao.BaseResource{
			ID:   id,
			Name: id,
			ARN:  "",
			Tags: make(map[string]string),
			Data: job,
		},
		Job:          &job,
		BackupPlanId: backupPlanId,
	}
}

// NewJobResourceFromDetail creates a new JobResource from detail
func NewJobResourceFromDetail(detail *backup.DescribeBackupJobOutput, backupPlanId string) *JobResource {
	id := appaws.Str(detail.BackupJobId)

	return &JobResource{
		BaseResource: dao.BaseResource{
			ID:   id,
			Name: id,
			ARN:  "",
			Tags: make(map[string]string),
			Data: detail,
		},
		Detail:       detail,
		BackupPlanId: backupPlanId,
	}
}

// JobId returns the backup job ID
func (r *JobResource) JobId() string {
	if r.Job != nil {
		return appaws.Str(r.Job.BackupJobId)
	}
	if r.Detail != nil {
		return appaws.Str(r.Detail.BackupJobId)
	}
	return ""
}

// State returns the job state
func (r *JobResource) State() string {
	if r.Job != nil {
		return string(r.Job.State)
	}
	if r.Detail != nil {
		return string(r.Detail.State)
	}
	return ""
}

// ResourceType returns the resource type being backed up
func (r *JobResource) ResourceType() string {
	if r.Job != nil {
		return appaws.Str(r.Job.ResourceType)
	}
	if r.Detail != nil {
		return appaws.Str(r.Detail.ResourceType)
	}
	return ""
}

// ResourceArn returns the ARN of the resource being backed up
func (r *JobResource) ResourceArn() string {
	if r.Job != nil {
		return appaws.Str(r.Job.ResourceArn)
	}
	if r.Detail != nil {
		return appaws.Str(r.Detail.ResourceArn)
	}
	return ""
}

// BackupVaultName returns the backup vault name
func (r *JobResource) BackupVaultName() string {
	if r.Job != nil {
		return appaws.Str(r.Job.BackupVaultName)
	}
	if r.Detail != nil {
		return appaws.Str(r.Detail.BackupVaultName)
	}
	return ""
}

// BackupSizeInBytes returns the backup size
func (r *JobResource) BackupSizeInBytes() int64 {
	if r.Job != nil && r.Job.BackupSizeInBytes != nil {
		return *r.Job.BackupSizeInBytes
	}
	if r.Detail != nil && r.Detail.BackupSizeInBytes != nil {
		return *r.Detail.BackupSizeInBytes
	}
	return 0
}

// BackupSizeFormatted returns the backup size formatted
func (r *JobResource) BackupSizeFormatted() string {
	bytes := r.BackupSizeInBytes()
	if bytes == 0 {
		return "-"
	}
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
		TB = GB * 1024
	)
	switch {
	case bytes >= TB:
		return fmt.Sprintf("%.2f TB", float64(bytes)/TB)
	case bytes >= GB:
		return fmt.Sprintf("%.2f GB", float64(bytes)/GB)
	case bytes >= MB:
		return fmt.Sprintf("%.2f MB", float64(bytes)/MB)
	case bytes >= KB:
		return fmt.Sprintf("%.2f KB", float64(bytes)/KB)
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}

// PercentDone returns the completion percentage
func (r *JobResource) PercentDone() string {
	if r.Job != nil && r.Job.PercentDone != nil {
		return appaws.Str(r.Job.PercentDone)
	}
	if r.Detail != nil && r.Detail.PercentDone != nil {
		return appaws.Str(r.Detail.PercentDone)
	}
	return ""
}

// StatusMessage returns the status message
func (r *JobResource) StatusMessage() string {
	if r.Job != nil {
		return appaws.Str(r.Job.StatusMessage)
	}
	if r.Detail != nil {
		return appaws.Str(r.Detail.StatusMessage)
	}
	return ""
}

// CreationDate returns the creation date
func (r *JobResource) CreationDate() string {
	if r.Job != nil && r.Job.CreationDate != nil {
		return r.Job.CreationDate.Format("2006-01-02 15:04:05")
	}
	if r.Detail != nil && r.Detail.CreationDate != nil {
		return r.Detail.CreationDate.Format("2006-01-02 15:04:05")
	}
	return ""
}

// CreationDateT returns the creation date as time.Time
func (r *JobResource) CreationDateT() *time.Time {
	if r.Job != nil {
		return r.Job.CreationDate
	}
	if r.Detail != nil {
		return r.Detail.CreationDate
	}
	return nil
}

// CompletionDate returns the completion date
func (r *JobResource) CompletionDate() string {
	if r.Job != nil && r.Job.CompletionDate != nil {
		return r.Job.CompletionDate.Format("2006-01-02 15:04:05")
	}
	if r.Detail != nil && r.Detail.CompletionDate != nil {
		return r.Detail.CompletionDate.Format("2006-01-02 15:04:05")
	}
	return ""
}

// ExpectedCompletionDate returns the expected completion date
func (r *JobResource) ExpectedCompletionDate() string {
	if r.Detail != nil && r.Detail.ExpectedCompletionDate != nil {
		return r.Detail.ExpectedCompletionDate.Format("2006-01-02 15:04:05")
	}
	return ""
}
