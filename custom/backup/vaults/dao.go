package vaults

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/backup"
	"github.com/aws/aws-sdk-go-v2/service/backup/types"

	appaws "github.com/clawscli/claws/internal/aws"
	"github.com/clawscli/claws/internal/dao"
)

// VaultDAO provides data access for AWS Backup vaults
type VaultDAO struct {
	dao.BaseDAO
	client *backup.Client
}

// NewVaultDAO creates a new VaultDAO
func NewVaultDAO(ctx context.Context) (dao.DAO, error) {
	cfg, err := appaws.NewConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("new backup/vaults dao: %w", err)
	}
	return &VaultDAO{
		BaseDAO: dao.NewBaseDAO("backup", "vaults"),
		client:  backup.NewFromConfig(cfg),
	}, nil
}

// List returns all AWS Backup vaults
func (d *VaultDAO) List(ctx context.Context) ([]dao.Resource, error) {
	vaults, err := appaws.Paginate(ctx, func(token *string) ([]types.BackupVaultListMember, *string, error) {
		output, err := d.client.ListBackupVaults(ctx, &backup.ListBackupVaultsInput{
			NextToken: token,
		})
		if err != nil {
			return nil, nil, fmt.Errorf("list backup vaults: %w", err)
		}
		return output.BackupVaultList, output.NextToken, nil
	})
	if err != nil {
		return nil, err
	}

	resources := make([]dao.Resource, len(vaults))
	for i, vault := range vaults {
		resources[i] = NewVaultResourceFromSummary(vault)
	}

	return resources, nil
}

// Get returns a specific AWS Backup vault by name
func (d *VaultDAO) Get(ctx context.Context, id string) (dao.Resource, error) {
	output, err := d.client.DescribeBackupVault(ctx, &backup.DescribeBackupVaultInput{
		BackupVaultName: &id,
	})
	if err != nil {
		return nil, fmt.Errorf("describe backup vault %s: %w", id, err)
	}

	return NewVaultResourceFromDetail(output), nil
}

// Delete deletes an AWS Backup vault
func (d *VaultDAO) Delete(ctx context.Context, id string) error {
	_, err := d.client.DeleteBackupVault(ctx, &backup.DeleteBackupVaultInput{
		BackupVaultName: &id,
	})
	if err != nil {
		return fmt.Errorf("delete backup vault %s: %w", id, err)
	}
	return nil
}

// VaultResource represents an AWS Backup vault
type VaultResource struct {
	dao.BaseResource
	Summary *types.BackupVaultListMember
	Detail  *backup.DescribeBackupVaultOutput
}

// NewVaultResourceFromSummary creates a new VaultResource from summary
func NewVaultResourceFromSummary(summary types.BackupVaultListMember) *VaultResource {
	name := appaws.Str(summary.BackupVaultName)
	arn := appaws.Str(summary.BackupVaultArn)

	return &VaultResource{
		BaseResource: dao.BaseResource{
			ID:   name,
			Name: name,
			ARN:  arn,
			Tags: make(map[string]string),
			Data: summary,
		},
		Summary: &summary,
	}
}

// NewVaultResourceFromDetail creates a new VaultResource from detail
func NewVaultResourceFromDetail(detail *backup.DescribeBackupVaultOutput) *VaultResource {
	name := appaws.Str(detail.BackupVaultName)
	arn := appaws.Str(detail.BackupVaultArn)

	return &VaultResource{
		BaseResource: dao.BaseResource{
			ID:   name,
			Name: name,
			ARN:  arn,
			Tags: make(map[string]string),
			Data: detail,
		},
		Detail: detail,
	}
}

// VaultName returns the vault name
func (r *VaultResource) VaultName() string {
	if r.Summary != nil {
		return appaws.Str(r.Summary.BackupVaultName)
	}
	if r.Detail != nil {
		return appaws.Str(r.Detail.BackupVaultName)
	}
	return ""
}

// VaultArn returns the vault ARN
func (r *VaultResource) VaultArn() string {
	if r.Summary != nil {
		return appaws.Str(r.Summary.BackupVaultArn)
	}
	if r.Detail != nil {
		return appaws.Str(r.Detail.BackupVaultArn)
	}
	return ""
}

// RecoveryPointCount returns the number of recovery points
func (r *VaultResource) RecoveryPointCount() int64 {
	if r.Summary != nil {
		return r.Summary.NumberOfRecoveryPoints
	}
	if r.Detail != nil {
		return r.Detail.NumberOfRecoveryPoints
	}
	return 0
}

// EncryptionKeyArn returns the encryption key ARN
func (r *VaultResource) EncryptionKeyArn() string {
	if r.Summary != nil {
		return appaws.Str(r.Summary.EncryptionKeyArn)
	}
	if r.Detail != nil {
		return appaws.Str(r.Detail.EncryptionKeyArn)
	}
	return ""
}

// CreatorRequestId returns the creator request ID
func (r *VaultResource) CreatorRequestId() string {
	if r.Summary != nil {
		return appaws.Str(r.Summary.CreatorRequestId)
	}
	if r.Detail != nil {
		return appaws.Str(r.Detail.CreatorRequestId)
	}
	return ""
}

// Locked returns whether the vault is locked
func (r *VaultResource) Locked() bool {
	if r.Summary != nil && r.Summary.Locked != nil {
		return *r.Summary.Locked
	}
	if r.Detail != nil && r.Detail.Locked != nil {
		return *r.Detail.Locked
	}
	return false
}

// LockDate returns the lock date
func (r *VaultResource) LockDate() string {
	if r.Summary != nil && r.Summary.LockDate != nil {
		return r.Summary.LockDate.Format("2006-01-02 15:04:05")
	}
	if r.Detail != nil && r.Detail.LockDate != nil {
		return r.Detail.LockDate.Format("2006-01-02 15:04:05")
	}
	return ""
}

// MinRetentionDays returns minimum retention days
func (r *VaultResource) MinRetentionDays() int64 {
	if r.Summary != nil && r.Summary.MinRetentionDays != nil {
		return *r.Summary.MinRetentionDays
	}
	if r.Detail != nil && r.Detail.MinRetentionDays != nil {
		return *r.Detail.MinRetentionDays
	}
	return 0
}

// MaxRetentionDays returns maximum retention days
func (r *VaultResource) MaxRetentionDays() int64 {
	if r.Summary != nil && r.Summary.MaxRetentionDays != nil {
		return *r.Summary.MaxRetentionDays
	}
	if r.Detail != nil && r.Detail.MaxRetentionDays != nil {
		return *r.Detail.MaxRetentionDays
	}
	return 0
}

// VaultType returns the vault type
func (r *VaultResource) VaultType() string {
	if r.Detail != nil {
		return string(r.Detail.VaultType)
	}
	return ""
}

// VaultState returns the vault state
func (r *VaultResource) VaultState() string {
	if r.Detail != nil {
		return string(r.Detail.VaultState)
	}
	return ""
}

// CreatedAt returns the creation date
func (r *VaultResource) CreatedAt() string {
	if r.Summary != nil && r.Summary.CreationDate != nil {
		return r.Summary.CreationDate.Format("2006-01-02 15:04:05")
	}
	if r.Detail != nil && r.Detail.CreationDate != nil {
		return r.Detail.CreationDate.Format("2006-01-02 15:04:05")
	}
	return ""
}

// CreatedAtTime returns the creation date as time.Time
func (r *VaultResource) CreatedAtTime() *time.Time {
	if r.Summary != nil && r.Summary.CreationDate != nil {
		return r.Summary.CreationDate
	}
	if r.Detail != nil && r.Detail.CreationDate != nil {
		return r.Detail.CreationDate
	}
	return nil
}

// SourceBackupVaultArn returns the source backup vault ARN (for logically air-gapped vaults)
func (r *VaultResource) SourceBackupVaultArn() string {
	if r.Detail != nil {
		return appaws.Str(r.Detail.SourceBackupVaultArn)
	}
	return ""
}

// EncryptionKeyType returns the encryption key type
func (r *VaultResource) EncryptionKeyType() string {
	if r.Detail != nil {
		return string(r.Detail.EncryptionKeyType)
	}
	return ""
}
