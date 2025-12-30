package backupjobs

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

// BackupJobDAO provides data access for AWS Backup jobs
type BackupJobDAO struct {
	dao.BaseDAO
	client *backup.Client
}

// NewBackupJobDAO creates a new BackupJobDAO
func NewBackupJobDAO(ctx context.Context) (dao.DAO, error) {
	cfg, err := appaws.NewConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("new backup/backup-jobs dao: %w", err)
	}
	return &BackupJobDAO{
		BaseDAO: dao.NewBaseDAO("backup", "backup-jobs"),
		client:  backup.NewFromConfig(cfg),
	}, nil
}

// List returns backup jobs (first page only for backwards compatibility).
// For paginated access, use ListPage instead.
func (d *BackupJobDAO) List(ctx context.Context) ([]dao.Resource, error) {
	resources, _, err := d.ListPage(ctx, 100, "")
	return resources, err
}

// ListPage returns a page of backup jobs.
// Implements dao.PaginatedDAO interface.
// If BackupPlanId filter is set, jobs are filtered client-side.
func (d *BackupJobDAO) ListPage(ctx context.Context, pageSize int, pageToken string) ([]dao.Resource, string, error) {
	// Get optional backup plan ID from filter context
	backupPlanId := dao.GetFilterFromContext(ctx, "BackupPlanId")

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

	resources := make([]dao.Resource, 0)
	for _, job := range output.BackupJobs {
		// If BackupPlanId filter is set, filter client-side
		if backupPlanId != "" {
			if job.CreatedBy != nil && job.CreatedBy.BackupPlanId != nil {
				if *job.CreatedBy.BackupPlanId == backupPlanId {
					resources = append(resources, NewBackupJobResource(job))
				}
			}
		} else {
			resources = append(resources, NewBackupJobResource(job))
		}
	}

	nextToken := ""
	if output.NextToken != nil {
		nextToken = *output.NextToken
	}

	return resources, nextToken, nil
}

// Get returns a specific backup job
func (d *BackupJobDAO) Get(ctx context.Context, jobId string) (dao.Resource, error) {
	input := &backup.DescribeBackupJobInput{
		BackupJobId: &jobId,
	}

	output, err := d.client.DescribeBackupJob(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("describe backup job %s: %w", jobId, err)
	}

	return NewBackupJobResourceFromDetail(output), nil
}

// Delete stops a backup job
func (d *BackupJobDAO) Delete(ctx context.Context, jobId string) error {
	_, err := d.client.StopBackupJob(ctx, &backup.StopBackupJobInput{
		BackupJobId: &jobId,
	})
	if err != nil {
		return fmt.Errorf("stop backup job %s: %w", jobId, err)
	}
	return nil
}

// Supports returns supported operations
func (d *BackupJobDAO) Supports(op dao.Operation) bool {
	switch op {
	case dao.OpList, dao.OpGet, dao.OpDelete:
		return true
	default:
		return false
	}
}

// BackupJobResource represents an AWS Backup job
type BackupJobResource struct {
	dao.BaseResource
	Job    *types.BackupJob
	Detail *backup.DescribeBackupJobOutput
}

// NewBackupJobResource creates a new BackupJobResource from list
func NewBackupJobResource(job types.BackupJob) *BackupJobResource {
	id := appaws.Str(job.BackupJobId)

	return &BackupJobResource{
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

// NewBackupJobResourceFromDetail creates a new BackupJobResource from detail
func NewBackupJobResourceFromDetail(detail *backup.DescribeBackupJobOutput) *BackupJobResource {
	id := appaws.Str(detail.BackupJobId)

	return &BackupJobResource{
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

// JobId returns the backup job ID
func (r *BackupJobResource) JobId() string {
	if r.Job != nil {
		return appaws.Str(r.Job.BackupJobId)
	}
	if r.Detail != nil {
		return appaws.Str(r.Detail.BackupJobId)
	}
	return ""
}

// State returns the job state
func (r *BackupJobResource) State() string {
	if r.Job != nil {
		return string(r.Job.State)
	}
	if r.Detail != nil {
		return string(r.Detail.State)
	}
	return ""
}

// ResourceType returns the resource type being backed up
func (r *BackupJobResource) ResourceType() string {
	if r.Job != nil {
		return appaws.Str(r.Job.ResourceType)
	}
	if r.Detail != nil {
		return appaws.Str(r.Detail.ResourceType)
	}
	return ""
}

// ResourceArn returns the ARN of the resource being backed up
func (r *BackupJobResource) ResourceArn() string {
	if r.Job != nil {
		return appaws.Str(r.Job.ResourceArn)
	}
	if r.Detail != nil {
		return appaws.Str(r.Detail.ResourceArn)
	}
	return ""
}

// BackupVaultName returns the backup vault name
func (r *BackupJobResource) BackupVaultName() string {
	if r.Job != nil {
		return appaws.Str(r.Job.BackupVaultName)
	}
	if r.Detail != nil {
		return appaws.Str(r.Detail.BackupVaultName)
	}
	return ""
}

// BackupVaultArn returns the backup vault ARN
func (r *BackupJobResource) BackupVaultArn() string {
	if r.Job != nil {
		return appaws.Str(r.Job.BackupVaultArn)
	}
	if r.Detail != nil {
		return appaws.Str(r.Detail.BackupVaultArn)
	}
	return ""
}

// RecoveryPointArn returns the recovery point ARN
func (r *BackupJobResource) RecoveryPointArn() string {
	if r.Job != nil {
		return appaws.Str(r.Job.RecoveryPointArn)
	}
	if r.Detail != nil {
		return appaws.Str(r.Detail.RecoveryPointArn)
	}
	return ""
}

// BackupPlanId returns the backup plan ID
func (r *BackupJobResource) BackupPlanId() string {
	if r.Job != nil && r.Job.CreatedBy != nil {
		return appaws.Str(r.Job.CreatedBy.BackupPlanId)
	}
	if r.Detail != nil && r.Detail.CreatedBy != nil {
		return appaws.Str(r.Detail.CreatedBy.BackupPlanId)
	}
	return ""
}

// IamRoleArn returns the IAM role ARN
func (r *BackupJobResource) IamRoleArn() string {
	if r.Job != nil {
		return appaws.Str(r.Job.IamRoleArn)
	}
	if r.Detail != nil {
		return appaws.Str(r.Detail.IamRoleArn)
	}
	return ""
}

// BackupSizeInBytes returns the backup size
func (r *BackupJobResource) BackupSizeInBytes() int64 {
	if r.Job != nil && r.Job.BackupSizeInBytes != nil {
		return *r.Job.BackupSizeInBytes
	}
	if r.Detail != nil && r.Detail.BackupSizeInBytes != nil {
		return *r.Detail.BackupSizeInBytes
	}
	return 0
}

// BackupSizeFormatted returns the backup size formatted
func (r *BackupJobResource) BackupSizeFormatted() string {
	bytes := r.BackupSizeInBytes()
	if bytes == 0 {
		return "-"
	}
	return render.FormatSize(bytes)
}

// PercentDone returns the completion percentage
func (r *BackupJobResource) PercentDone() string {
	if r.Job != nil && r.Job.PercentDone != nil {
		return appaws.Str(r.Job.PercentDone)
	}
	if r.Detail != nil && r.Detail.PercentDone != nil {
		return appaws.Str(r.Detail.PercentDone)
	}
	return ""
}

// StatusMessage returns the status message
func (r *BackupJobResource) StatusMessage() string {
	if r.Job != nil {
		return appaws.Str(r.Job.StatusMessage)
	}
	if r.Detail != nil {
		return appaws.Str(r.Detail.StatusMessage)
	}
	return ""
}

// StartBy returns the start by time
func (r *BackupJobResource) StartBy() string {
	if r.Job != nil && r.Job.StartBy != nil {
		return r.Job.StartBy.Format("2006-01-02 15:04:05")
	}
	if r.Detail != nil && r.Detail.StartBy != nil {
		return r.Detail.StartBy.Format("2006-01-02 15:04:05")
	}
	return ""
}

// CreationDate returns the creation date
func (r *BackupJobResource) CreationDate() string {
	if r.Job != nil && r.Job.CreationDate != nil {
		return r.Job.CreationDate.Format("2006-01-02 15:04:05")
	}
	if r.Detail != nil && r.Detail.CreationDate != nil {
		return r.Detail.CreationDate.Format("2006-01-02 15:04:05")
	}
	return ""
}

// CreationDateT returns the creation date as time.Time
func (r *BackupJobResource) CreationDateT() *time.Time {
	if r.Job != nil {
		return r.Job.CreationDate
	}
	if r.Detail != nil {
		return r.Detail.CreationDate
	}
	return nil
}

// CompletionDate returns the completion date
func (r *BackupJobResource) CompletionDate() string {
	if r.Job != nil && r.Job.CompletionDate != nil {
		return r.Job.CompletionDate.Format("2006-01-02 15:04:05")
	}
	if r.Detail != nil && r.Detail.CompletionDate != nil {
		return r.Detail.CompletionDate.Format("2006-01-02 15:04:05")
	}
	return ""
}

// ExpectedCompletionDate returns the expected completion date
func (r *BackupJobResource) ExpectedCompletionDate() string {
	if r.Detail != nil && r.Detail.ExpectedCompletionDate != nil {
		return r.Detail.ExpectedCompletionDate.Format("2006-01-02 15:04:05")
	}
	return ""
}

// BackupType returns the backup type
func (r *BackupJobResource) BackupType() string {
	if r.Job != nil {
		return appaws.Str(r.Job.BackupType)
	}
	if r.Detail != nil {
		return appaws.Str(r.Detail.BackupType)
	}
	return ""
}

// IsParent returns whether this is a parent job
func (r *BackupJobResource) IsParent() bool {
	if r.Job != nil {
		return r.Job.IsParent
	}
	if r.Detail != nil {
		return r.Detail.IsParent
	}
	return false
}

// ParentJobId returns the parent job ID
func (r *BackupJobResource) ParentJobId() string {
	if r.Job != nil {
		return appaws.Str(r.Job.ParentJobId)
	}
	if r.Detail != nil {
		return appaws.Str(r.Detail.ParentJobId)
	}
	return ""
}

// MessageCategory returns the message category
func (r *BackupJobResource) MessageCategory() string {
	if r.Job != nil {
		return appaws.Str(r.Job.MessageCategory)
	}
	if r.Detail != nil {
		return appaws.Str(r.Detail.MessageCategory)
	}
	return ""
}

// IsEncrypted returns whether the backup is encrypted
func (r *BackupJobResource) IsEncrypted() bool {
	if r.Detail != nil {
		return r.Detail.IsEncrypted
	}
	return false
}

// EncryptionKeyArn returns the encryption key ARN
func (r *BackupJobResource) EncryptionKeyArn() string {
	if r.Detail != nil {
		return appaws.Str(r.Detail.EncryptionKeyArn)
	}
	return ""
}

// ResourceName returns the resource name
func (r *BackupJobResource) ResourceName() string {
	if r.Detail != nil {
		return appaws.Str(r.Detail.ResourceName)
	}
	return ""
}

// AccountId returns the account ID
func (r *BackupJobResource) AccountId() string {
	if r.Job != nil {
		return appaws.Str(r.Job.AccountId)
	}
	if r.Detail != nil {
		return appaws.Str(r.Detail.AccountId)
	}
	return ""
}

// BytesTransferred returns bytes transferred
func (r *BackupJobResource) BytesTransferred() int64 {
	if r.Detail != nil && r.Detail.BytesTransferred != nil {
		return *r.Detail.BytesTransferred
	}
	return 0
}

// InitiationDate returns the initiation date
func (r *BackupJobResource) InitiationDate() string {
	if r.Detail != nil && r.Detail.InitiationDate != nil {
		return r.Detail.InitiationDate.Format("2006-01-02 15:04:05")
	}
	return ""
}
