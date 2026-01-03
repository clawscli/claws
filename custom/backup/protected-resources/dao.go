package protectedresources

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/backup"
	"github.com/aws/aws-sdk-go-v2/service/backup/types"

	appaws "github.com/clawscli/claws/internal/aws"
	"github.com/clawscli/claws/internal/dao"
	apperrors "github.com/clawscli/claws/internal/errors"
)

// ProtectedResourceDAO provides data access for AWS Backup protected resources
type ProtectedResourceDAO struct {
	dao.BaseDAO
	client *backup.Client
}

// NewProtectedResourceDAO creates a new ProtectedResourceDAO
func NewProtectedResourceDAO(ctx context.Context) (dao.DAO, error) {
	cfg, err := appaws.NewConfig(ctx)
	if err != nil {
		return nil, apperrors.Wrap(err, "new "+ServiceResourcePath+" dao")
	}
	return &ProtectedResourceDAO{
		BaseDAO: dao.NewBaseDAO("backup", "protected-resources"),
		client:  backup.NewFromConfig(cfg),
	}, nil
}

// List returns protected resources
func (d *ProtectedResourceDAO) List(ctx context.Context) ([]dao.Resource, error) {
	resources, err := appaws.Paginate(ctx, func(token *string) ([]types.ProtectedResource, *string, error) {
		output, err := d.client.ListProtectedResources(ctx, &backup.ListProtectedResourcesInput{
			NextToken: token,
		})
		if err != nil {
			return nil, nil, apperrors.Wrap(err, "list protected resources")
		}
		return output.Results, output.NextToken, nil
	})
	if err != nil {
		return nil, err
	}

	result := make([]dao.Resource, len(resources))
	for i, res := range resources {
		result[i] = NewProtectedResourceFromSummary(res)
	}

	return result, nil
}

// Get returns a specific protected resource
func (d *ProtectedResourceDAO) Get(ctx context.Context, arn string) (dao.Resource, error) {
	output, err := d.client.DescribeProtectedResource(ctx, &backup.DescribeProtectedResourceInput{
		ResourceArn: &arn,
	})
	if err != nil {
		return nil, apperrors.Wrapf(err, "describe protected resource %s", arn)
	}

	return NewProtectedResourceFromDetail(output), nil
}

// Delete is not supported for protected resources
func (d *ProtectedResourceDAO) Delete(ctx context.Context, id string) error {
	return fmt.Errorf("delete not supported for protected resources")
}

// Supports returns supported operations
func (d *ProtectedResourceDAO) Supports(op dao.Operation) bool {
	switch op {
	case dao.OpList, dao.OpGet:
		return true
	default:
		return false
	}
}

// ProtectedResourceResource represents an AWS Backup protected resource
type ProtectedResourceResource struct {
	dao.BaseResource
	Summary *types.ProtectedResource
	Detail  *backup.DescribeProtectedResourceOutput
}

// NewProtectedResourceFromSummary creates a new ProtectedResourceResource from summary
func NewProtectedResourceFromSummary(summary types.ProtectedResource) *ProtectedResourceResource {
	arn := appaws.Str(summary.ResourceArn)
	name := extractResourceName(arn)

	return &ProtectedResourceResource{
		BaseResource: dao.BaseResource{
			ID:   arn,
			Name: name,
			ARN:  arn,
			Tags: make(map[string]string),
			Data: summary,
		},
		Summary: &summary,
	}
}

// NewProtectedResourceFromDetail creates a new ProtectedResourceResource from detail
func NewProtectedResourceFromDetail(detail *backup.DescribeProtectedResourceOutput) *ProtectedResourceResource {
	arn := appaws.Str(detail.ResourceArn)
	name := extractResourceName(arn)

	return &ProtectedResourceResource{
		BaseResource: dao.BaseResource{
			ID:   arn,
			Name: name,
			ARN:  arn,
			Tags: make(map[string]string),
			Data: detail,
		},
		Detail: detail,
	}
}

// extractResourceName extracts a friendly name from an ARN
func extractResourceName(arn string) string {
	if len(arn) > 60 {
		return "..." + arn[len(arn)-57:]
	}
	return arn
}

// ResourceArn returns the resource ARN
func (r *ProtectedResourceResource) ResourceArn() string {
	if r.Summary != nil {
		return appaws.Str(r.Summary.ResourceArn)
	}
	if r.Detail != nil {
		return appaws.Str(r.Detail.ResourceArn)
	}
	return ""
}

// ResourceType returns the resource type
func (r *ProtectedResourceResource) ResourceType() string {
	if r.Summary != nil {
		return appaws.Str(r.Summary.ResourceType)
	}
	if r.Detail != nil {
		return appaws.Str(r.Detail.ResourceType)
	}
	return ""
}

// ResourceName returns the resource name
func (r *ProtectedResourceResource) ResourceName() string {
	if r.Summary != nil {
		return appaws.Str(r.Summary.ResourceName)
	}
	if r.Detail != nil {
		return appaws.Str(r.Detail.ResourceName)
	}
	return ""
}

// LastBackupTime returns the last backup time
func (r *ProtectedResourceResource) LastBackupTime() string {
	if r.Summary != nil && r.Summary.LastBackupTime != nil {
		return r.Summary.LastBackupTime.Format("2006-01-02 15:04:05")
	}
	if r.Detail != nil && r.Detail.LastBackupTime != nil {
		return r.Detail.LastBackupTime.Format("2006-01-02 15:04:05")
	}
	return ""
}

// LastBackupTimeT returns the last backup time as time.Time
func (r *ProtectedResourceResource) LastBackupTimeT() *time.Time {
	if r.Summary != nil {
		return r.Summary.LastBackupTime
	}
	if r.Detail != nil {
		return r.Detail.LastBackupTime
	}
	return nil
}

// LastBackupVaultArn returns the last backup vault ARN
func (r *ProtectedResourceResource) LastBackupVaultArn() string {
	if r.Detail != nil {
		return appaws.Str(r.Detail.LastBackupVaultArn)
	}
	return ""
}

// LastRecoveryPointArn returns the last recovery point ARN
func (r *ProtectedResourceResource) LastRecoveryPointArn() string {
	if r.Detail != nil {
		return appaws.Str(r.Detail.LastRecoveryPointArn)
	}
	return ""
}

// LatestRestoreExecutionTimeMinutes returns the latest restore execution time
func (r *ProtectedResourceResource) LatestRestoreExecutionTimeMinutes() int64 {
	if r.Detail != nil && r.Detail.LatestRestoreExecutionTimeMinutes != nil {
		return *r.Detail.LatestRestoreExecutionTimeMinutes
	}
	return 0
}

// LatestRestoreJobCreationDate returns the latest restore job creation date
func (r *ProtectedResourceResource) LatestRestoreJobCreationDate() string {
	if r.Detail != nil && r.Detail.LatestRestoreJobCreationDate != nil {
		return r.Detail.LatestRestoreJobCreationDate.Format("2006-01-02 15:04:05")
	}
	return ""
}

// LatestRestoreRecoveryPointCreationDate returns the latest restore recovery point creation date
func (r *ProtectedResourceResource) LatestRestoreRecoveryPointCreationDate() string {
	if r.Detail != nil && r.Detail.LatestRestoreRecoveryPointCreationDate != nil {
		return r.Detail.LatestRestoreRecoveryPointCreationDate.Format("2006-01-02 15:04:05")
	}
	return ""
}
