package copyjobs

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

// CopyJobDAO provides data access for AWS Backup copy jobs
type CopyJobDAO struct {
	dao.BaseDAO
	client *backup.Client
}

// NewCopyJobDAO creates a new CopyJobDAO
func NewCopyJobDAO(ctx context.Context) (dao.DAO, error) {
	cfg, err := appaws.NewConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("new backup/copy-jobs dao: %w", err)
	}
	return &CopyJobDAO{
		BaseDAO: dao.NewBaseDAO("backup", "copy-jobs"),
		client:  backup.NewFromConfig(cfg),
	}, nil
}

// List returns copy jobs (first page)
func (d *CopyJobDAO) List(ctx context.Context) ([]dao.Resource, error) {
	resources, _, err := d.ListPage(ctx, 100, "")
	return resources, err
}

// ListPage returns a page of copy jobs
func (d *CopyJobDAO) ListPage(ctx context.Context, pageSize int, pageToken string) ([]dao.Resource, string, error) {
	maxResults := int32(pageSize)
	if maxResults > 1000 {
		maxResults = 1000
	}

	input := &backup.ListCopyJobsInput{
		MaxResults: &maxResults,
	}
	if pageToken != "" {
		input.NextToken = &pageToken
	}

	output, err := d.client.ListCopyJobs(ctx, input)
	if err != nil {
		return nil, "", fmt.Errorf("list copy jobs: %w", err)
	}

	resources := make([]dao.Resource, len(output.CopyJobs))
	for i, job := range output.CopyJobs {
		resources[i] = NewCopyJobResource(job)
	}

	nextToken := ""
	if output.NextToken != nil {
		nextToken = *output.NextToken
	}

	return resources, nextToken, nil
}

// Get returns a specific copy job
func (d *CopyJobDAO) Get(ctx context.Context, jobId string) (dao.Resource, error) {
	output, err := d.client.DescribeCopyJob(ctx, &backup.DescribeCopyJobInput{
		CopyJobId: &jobId,
	})
	if err != nil {
		return nil, fmt.Errorf("describe copy job %s: %w", jobId, err)
	}

	return NewCopyJobResourceFromDetail(output), nil
}

// Delete is not supported for copy jobs
func (d *CopyJobDAO) Delete(ctx context.Context, id string) error {
	return fmt.Errorf("delete not supported for copy jobs")
}

// Supports returns supported operations
func (d *CopyJobDAO) Supports(op dao.Operation) bool {
	switch op {
	case dao.OpList, dao.OpGet:
		return true
	default:
		return false
	}
}

// CopyJobResource represents an AWS Backup copy job
type CopyJobResource struct {
	dao.BaseResource
	Job    *types.CopyJob
	Detail *backup.DescribeCopyJobOutput
}

// NewCopyJobResource creates a new CopyJobResource from list
func NewCopyJobResource(job types.CopyJob) *CopyJobResource {
	id := appaws.Str(job.CopyJobId)

	return &CopyJobResource{
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

// NewCopyJobResourceFromDetail creates a new CopyJobResource from detail
func NewCopyJobResourceFromDetail(detail *backup.DescribeCopyJobOutput) *CopyJobResource {
	id := ""
	if detail.CopyJob != nil {
		id = appaws.Str(detail.CopyJob.CopyJobId)
	}

	return &CopyJobResource{
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

func (r *CopyJobResource) getJob() *types.CopyJob {
	if r.Job != nil {
		return r.Job
	}
	if r.Detail != nil {
		return r.Detail.CopyJob
	}
	return nil
}

// JobId returns the copy job ID
func (r *CopyJobResource) JobId() string {
	if job := r.getJob(); job != nil {
		return appaws.Str(job.CopyJobId)
	}
	return ""
}

// State returns the job state
func (r *CopyJobResource) State() string {
	if job := r.getJob(); job != nil {
		return string(job.State)
	}
	return ""
}

// StatusMessage returns the status message
func (r *CopyJobResource) StatusMessage() string {
	if job := r.getJob(); job != nil {
		return appaws.Str(job.StatusMessage)
	}
	return ""
}

// ResourceType returns the resource type
func (r *CopyJobResource) ResourceType() string {
	if job := r.getJob(); job != nil {
		return appaws.Str(job.ResourceType)
	}
	return ""
}

// ResourceArn returns the resource ARN
func (r *CopyJobResource) ResourceArn() string {
	if job := r.getJob(); job != nil {
		return appaws.Str(job.ResourceArn)
	}
	return ""
}

// SourceBackupVaultArn returns the source backup vault ARN
func (r *CopyJobResource) SourceBackupVaultArn() string {
	if job := r.getJob(); job != nil {
		return appaws.Str(job.SourceBackupVaultArn)
	}
	return ""
}

// SourceRecoveryPointArn returns the source recovery point ARN
func (r *CopyJobResource) SourceRecoveryPointArn() string {
	if job := r.getJob(); job != nil {
		return appaws.Str(job.SourceRecoveryPointArn)
	}
	return ""
}

// DestinationBackupVaultArn returns the destination backup vault ARN
func (r *CopyJobResource) DestinationBackupVaultArn() string {
	if job := r.getJob(); job != nil {
		return appaws.Str(job.DestinationBackupVaultArn)
	}
	return ""
}

// DestinationRecoveryPointArn returns the destination recovery point ARN
func (r *CopyJobResource) DestinationRecoveryPointArn() string {
	if job := r.getJob(); job != nil {
		return appaws.Str(job.DestinationRecoveryPointArn)
	}
	return ""
}

// BackupSizeBytes returns the backup size in bytes
func (r *CopyJobResource) BackupSizeBytes() int64 {
	if job := r.getJob(); job != nil && job.BackupSizeInBytes != nil {
		return *job.BackupSizeInBytes
	}
	return 0
}

// BackupSizeFormatted returns the formatted backup size
func (r *CopyJobResource) BackupSizeFormatted() string {
	bytes := r.BackupSizeBytes()
	if bytes == 0 {
		return "-"
	}
	return render.FormatSize(bytes)
}

// IamRoleArn returns the IAM role ARN
func (r *CopyJobResource) IamRoleArn() string {
	if job := r.getJob(); job != nil {
		return appaws.Str(job.IamRoleArn)
	}
	return ""
}

// AccountId returns the account ID
func (r *CopyJobResource) AccountId() string {
	if job := r.getJob(); job != nil {
		return appaws.Str(job.AccountId)
	}
	return ""
}

// IsParent returns whether this is a parent job
func (r *CopyJobResource) IsParent() bool {
	if job := r.getJob(); job != nil {
		return job.IsParent
	}
	return false
}

// ParentJobId returns the parent job ID
func (r *CopyJobResource) ParentJobId() string {
	if job := r.getJob(); job != nil {
		return appaws.Str(job.ParentJobId)
	}
	return ""
}

// NumberOfChildJobs returns the number of child jobs
func (r *CopyJobResource) NumberOfChildJobs() int64 {
	if job := r.getJob(); job != nil && job.NumberOfChildJobs != nil {
		return *job.NumberOfChildJobs
	}
	return 0
}

// MessageCategory returns the message category
func (r *CopyJobResource) MessageCategory() string {
	if job := r.getJob(); job != nil {
		return appaws.Str(job.MessageCategory)
	}
	return ""
}

// CompositeMemberIdentifier returns the composite member identifier
func (r *CopyJobResource) CompositeMemberIdentifier() string {
	if job := r.getJob(); job != nil {
		return appaws.Str(job.CompositeMemberIdentifier)
	}
	return ""
}

// CreationDate returns the creation date
func (r *CopyJobResource) CreationDate() string {
	if job := r.getJob(); job != nil && job.CreationDate != nil {
		return job.CreationDate.Format("2006-01-02 15:04:05")
	}
	return ""
}

// CreationDateT returns the creation date as time.Time
func (r *CopyJobResource) CreationDateT() *time.Time {
	if job := r.getJob(); job != nil {
		return job.CreationDate
	}
	return nil
}

// CompletionDate returns the completion date
func (r *CopyJobResource) CompletionDate() string {
	if job := r.getJob(); job != nil && job.CompletionDate != nil {
		return job.CompletionDate.Format("2006-01-02 15:04:05")
	}
	return ""
}
