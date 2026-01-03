package recoverypoints

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/backup"
	"github.com/aws/aws-sdk-go-v2/service/backup/types"

	appaws "github.com/clawscli/claws/internal/aws"
	"github.com/clawscli/claws/internal/dao"
	apperrors "github.com/clawscli/claws/internal/errors"
	"github.com/clawscli/claws/internal/render"
)

// RecoveryPointDAO provides data access for AWS Backup recovery points
type RecoveryPointDAO struct {
	dao.BaseDAO
	client *backup.Client
}

// NewRecoveryPointDAO creates a new RecoveryPointDAO
func NewRecoveryPointDAO(ctx context.Context) (dao.DAO, error) {
	cfg, err := appaws.NewConfig(ctx)
	if err != nil {
		return nil, apperrors.Wrap(err, "new "+ServiceResourcePath+" dao")
	}
	return &RecoveryPointDAO{
		BaseDAO: dao.NewBaseDAO("backup", "recovery-points"),
		client:  backup.NewFromConfig(cfg),
	}, nil
}

// List returns AWS Backup recovery points for a vault
func (d *RecoveryPointDAO) List(ctx context.Context) ([]dao.Resource, error) {
	vaultName := dao.GetFilterFromContext(ctx, "VaultName")
	if vaultName == "" {
		return nil, fmt.Errorf("navigate from vaults using 'r' key")
	}

	points, err := appaws.Paginate(ctx, func(token *string) ([]types.RecoveryPointByBackupVault, *string, error) {
		output, err := d.client.ListRecoveryPointsByBackupVault(ctx, &backup.ListRecoveryPointsByBackupVaultInput{
			BackupVaultName: &vaultName,
			NextToken:       token,
		})
		if err != nil {
			return nil, nil, apperrors.Wrapf(err, "list recovery points for vault %s", vaultName)
		}
		return output.RecoveryPoints, output.NextToken, nil
	})
	if err != nil {
		return nil, err
	}

	resources := make([]dao.Resource, len(points))
	for i, point := range points {
		resources[i] = NewRecoveryPointResourceFromSummary(point, vaultName)
	}

	return resources, nil
}

// Get returns a specific AWS Backup recovery point
func (d *RecoveryPointDAO) Get(ctx context.Context, id string) (dao.Resource, error) {
	vaultName := dao.GetFilterFromContext(ctx, "VaultName")
	if vaultName == "" {
		return nil, fmt.Errorf("navigate from vaults using 'r' key")
	}

	output, err := d.client.DescribeRecoveryPoint(ctx, &backup.DescribeRecoveryPointInput{
		BackupVaultName:  &vaultName,
		RecoveryPointArn: &id,
	})
	if err != nil {
		return nil, apperrors.Wrapf(err, "describe recovery point %s", id)
	}

	return NewRecoveryPointResourceFromDetail(output, vaultName), nil
}

// Delete deletes an AWS Backup recovery point
func (d *RecoveryPointDAO) Delete(ctx context.Context, id string) error {
	vaultName := dao.GetFilterFromContext(ctx, "VaultName")
	if vaultName == "" {
		return fmt.Errorf("navigate from vaults using 'r' key")
	}

	_, err := d.client.DeleteRecoveryPoint(ctx, &backup.DeleteRecoveryPointInput{
		BackupVaultName:  &vaultName,
		RecoveryPointArn: &id,
	})
	if err != nil {
		return apperrors.Wrapf(err, "delete recovery point %s", id)
	}
	return nil
}

// RecoveryPointResource represents an AWS Backup recovery point
type RecoveryPointResource struct {
	dao.BaseResource
	Summary   *types.RecoveryPointByBackupVault
	Detail    *backup.DescribeRecoveryPointOutput
	VaultName string
}

// NewRecoveryPointResourceFromSummary creates a new RecoveryPointResource from summary
func NewRecoveryPointResourceFromSummary(summary types.RecoveryPointByBackupVault, vaultName string) *RecoveryPointResource {
	arn := appaws.Str(summary.RecoveryPointArn)
	// Extract ID from ARN (last segment)
	id := arn
	if len(arn) > 0 {
		// Recovery point ARN format: arn:aws:backup:region:account:recovery-point:id
		id = extractRecoveryPointId(arn)
	}

	return &RecoveryPointResource{
		BaseResource: dao.BaseResource{
			ID:   arn,
			Name: id,
			ARN:  arn,
			Tags: make(map[string]string),
			Data: summary,
		},
		Summary:   &summary,
		VaultName: vaultName,
	}
}

// NewRecoveryPointResourceFromDetail creates a new RecoveryPointResource from detail
func NewRecoveryPointResourceFromDetail(detail *backup.DescribeRecoveryPointOutput, vaultName string) *RecoveryPointResource {
	arn := appaws.Str(detail.RecoveryPointArn)
	id := extractRecoveryPointId(arn)

	return &RecoveryPointResource{
		BaseResource: dao.BaseResource{
			ID:   arn,
			Name: id,
			ARN:  arn,
			Tags: make(map[string]string),
			Data: detail,
		},
		Detail:    detail,
		VaultName: vaultName,
	}
}

func extractRecoveryPointId(arn string) string {
	// Recovery point ARN format: arn:aws:backup:region:account:recovery-point:id
	// or: arn:aws:ec2:region:account:snapshot/snap-xxx
	if len(arn) == 0 {
		return ""
	}
	// Just return last 40 chars if ARN is too long for display
	if len(arn) > 50 {
		return "..." + arn[len(arn)-40:]
	}
	return arn
}

// RecoveryPointArn returns the recovery point ARN
func (r *RecoveryPointResource) RecoveryPointArn() string {
	if r.Summary != nil {
		return appaws.Str(r.Summary.RecoveryPointArn)
	}
	if r.Detail != nil {
		return appaws.Str(r.Detail.RecoveryPointArn)
	}
	return ""
}

// Status returns the recovery point status
func (r *RecoveryPointResource) Status() string {
	if r.Summary != nil {
		return string(r.Summary.Status)
	}
	if r.Detail != nil {
		return string(r.Detail.Status)
	}
	return ""
}

// StatusMessage returns the status message
func (r *RecoveryPointResource) StatusMessage() string {
	if r.Summary != nil {
		return appaws.Str(r.Summary.StatusMessage)
	}
	if r.Detail != nil {
		return appaws.Str(r.Detail.StatusMessage)
	}
	return ""
}

// ResourceType returns the resource type
func (r *RecoveryPointResource) ResourceType() string {
	if r.Summary != nil {
		return appaws.Str(r.Summary.ResourceType)
	}
	if r.Detail != nil {
		return appaws.Str(r.Detail.ResourceType)
	}
	return ""
}

// ResourceArn returns the resource ARN
func (r *RecoveryPointResource) ResourceArn() string {
	if r.Summary != nil {
		return appaws.Str(r.Summary.ResourceArn)
	}
	if r.Detail != nil {
		return appaws.Str(r.Detail.ResourceArn)
	}
	return ""
}

// ResourceName returns the resource name (extracted from ARN)
func (r *RecoveryPointResource) ResourceName() string {
	if r.Detail != nil {
		return appaws.Str(r.Detail.ResourceName)
	}
	return ""
}

// BackupSizeBytes returns the backup size in bytes
func (r *RecoveryPointResource) BackupSizeBytes() int64 {
	if r.Summary != nil && r.Summary.BackupSizeInBytes != nil {
		return *r.Summary.BackupSizeInBytes
	}
	if r.Detail != nil && r.Detail.BackupSizeInBytes != nil {
		return *r.Detail.BackupSizeInBytes
	}
	return 0
}

// BackupSizeFormatted returns the formatted backup size
func (r *RecoveryPointResource) BackupSizeFormatted() string {
	bytes := r.BackupSizeBytes()
	if bytes == 0 {
		return "-"
	}
	return render.FormatSize(bytes)
}

// IsEncrypted returns whether the recovery point is encrypted
func (r *RecoveryPointResource) IsEncrypted() bool {
	if r.Summary != nil {
		return r.Summary.IsEncrypted
	}
	if r.Detail != nil {
		return r.Detail.IsEncrypted
	}
	return false
}

// EncryptionKeyArn returns the encryption key ARN
func (r *RecoveryPointResource) EncryptionKeyArn() string {
	if r.Summary != nil {
		return appaws.Str(r.Summary.EncryptionKeyArn)
	}
	if r.Detail != nil {
		return appaws.Str(r.Detail.EncryptionKeyArn)
	}
	return ""
}

// StorageClass returns the storage class
func (r *RecoveryPointResource) StorageClass() string {
	if r.Detail != nil {
		return string(r.Detail.StorageClass)
	}
	return ""
}

// IamRoleArn returns the IAM role ARN
func (r *RecoveryPointResource) IamRoleArn() string {
	if r.Summary != nil {
		return appaws.Str(r.Summary.IamRoleArn)
	}
	if r.Detail != nil {
		return appaws.Str(r.Detail.IamRoleArn)
	}
	return ""
}

// BackupPlanId returns the backup plan ID (from calculated lifecycle)
func (r *RecoveryPointResource) BackupPlanId() string {
	// CalculatedLifecycle doesn't contain BackupPlanId; this info is not directly available
	return ""
}

// CreatedBy returns info about what created this recovery point
func (r *RecoveryPointResource) CreatedBy() *types.RecoveryPointCreator {
	if r.Summary != nil {
		return r.Summary.CreatedBy
	}
	if r.Detail != nil {
		return r.Detail.CreatedBy
	}
	return nil
}

// Lifecycle returns the lifecycle configuration
func (r *RecoveryPointResource) Lifecycle() *types.Lifecycle {
	if r.Summary != nil {
		return r.Summary.Lifecycle
	}
	if r.Detail != nil {
		return r.Detail.Lifecycle
	}
	return nil
}

// CalculatedLifecycle returns the calculated lifecycle
func (r *RecoveryPointResource) CalculatedLifecycle() *types.CalculatedLifecycle {
	if r.Summary != nil {
		return r.Summary.CalculatedLifecycle
	}
	if r.Detail != nil {
		return r.Detail.CalculatedLifecycle
	}
	return nil
}

// IsParent returns whether this is a parent recovery point
func (r *RecoveryPointResource) IsParent() bool {
	if r.Summary != nil {
		return r.Summary.IsParent
	}
	if r.Detail != nil {
		return r.Detail.IsParent
	}
	return false
}

// ParentRecoveryPointArn returns the parent recovery point ARN
func (r *RecoveryPointResource) ParentRecoveryPointArn() string {
	if r.Summary != nil {
		return appaws.Str(r.Summary.ParentRecoveryPointArn)
	}
	if r.Detail != nil {
		return appaws.Str(r.Detail.ParentRecoveryPointArn)
	}
	return ""
}

// CompositeMemberIdentifier returns the composite member identifier
func (r *RecoveryPointResource) CompositeMemberIdentifier() string {
	if r.Summary != nil {
		return appaws.Str(r.Summary.CompositeMemberIdentifier)
	}
	if r.Detail != nil {
		return appaws.Str(r.Detail.CompositeMemberIdentifier)
	}
	return ""
}

// CreationDate returns the creation date
func (r *RecoveryPointResource) CreationDate() string {
	if r.Summary != nil && r.Summary.CreationDate != nil {
		return r.Summary.CreationDate.Format("2006-01-02 15:04:05")
	}
	if r.Detail != nil && r.Detail.CreationDate != nil {
		return r.Detail.CreationDate.Format("2006-01-02 15:04:05")
	}
	return ""
}

// CreationDateT returns the creation date as time.Time
func (r *RecoveryPointResource) CreationDateT() *time.Time {
	if r.Summary != nil && r.Summary.CreationDate != nil {
		return r.Summary.CreationDate
	}
	if r.Detail != nil && r.Detail.CreationDate != nil {
		return r.Detail.CreationDate
	}
	return nil
}

// CompletionDate returns the completion date
func (r *RecoveryPointResource) CompletionDate() string {
	if r.Summary != nil && r.Summary.CompletionDate != nil {
		return r.Summary.CompletionDate.Format("2006-01-02 15:04:05")
	}
	if r.Detail != nil && r.Detail.CompletionDate != nil {
		return r.Detail.CompletionDate.Format("2006-01-02 15:04:05")
	}
	return ""
}

// LastRestoreTime returns the last restore time
func (r *RecoveryPointResource) LastRestoreTime() string {
	if r.Detail != nil && r.Detail.LastRestoreTime != nil {
		return r.Detail.LastRestoreTime.Format("2006-01-02 15:04:05")
	}
	return ""
}
