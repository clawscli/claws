package plans

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/backup"
	"github.com/aws/aws-sdk-go-v2/service/backup/types"

	appaws "github.com/clawscli/claws/internal/aws"
	"github.com/clawscli/claws/internal/dao"
	apperrors "github.com/clawscli/claws/internal/errors"
)

// BackupPlanDAO provides data access for AWS Backup plans
type BackupPlanDAO struct {
	dao.BaseDAO
	client *backup.Client
}

// NewBackupPlanDAO creates a new BackupPlanDAO
func NewBackupPlanDAO(ctx context.Context) (dao.DAO, error) {
	cfg, err := appaws.NewConfig(ctx)
	if err != nil {
		return nil, apperrors.Wrap(err, "new "+ServiceResourcePath+" dao")
	}
	return &BackupPlanDAO{
		BaseDAO: dao.NewBaseDAO("backup", "plans"),
		client:  backup.NewFromConfig(cfg),
	}, nil
}

// List returns all AWS Backup plans
func (d *BackupPlanDAO) List(ctx context.Context) ([]dao.Resource, error) {
	plans, err := appaws.Paginate(ctx, func(token *string) ([]types.BackupPlansListMember, *string, error) {
		output, err := d.client.ListBackupPlans(ctx, &backup.ListBackupPlansInput{
			NextToken: token,
		})
		if err != nil {
			return nil, nil, apperrors.Wrap(err, "list backup plans")
		}
		return output.BackupPlansList, output.NextToken, nil
	})
	if err != nil {
		return nil, err
	}

	resources := make([]dao.Resource, len(plans))
	for i, plan := range plans {
		resources[i] = NewBackupPlanResourceFromSummary(plan)
	}

	return resources, nil
}

// Get returns a specific AWS Backup plan by ID
func (d *BackupPlanDAO) Get(ctx context.Context, id string) (dao.Resource, error) {
	output, err := d.client.GetBackupPlan(ctx, &backup.GetBackupPlanInput{
		BackupPlanId: &id,
	})
	if err != nil {
		return nil, apperrors.Wrapf(err, "get backup plan %s", id)
	}

	return NewBackupPlanResourceFromDetail(output), nil
}

// Delete deletes an AWS Backup plan
func (d *BackupPlanDAO) Delete(ctx context.Context, id string) error {
	_, err := d.client.DeleteBackupPlan(ctx, &backup.DeleteBackupPlanInput{
		BackupPlanId: &id,
	})
	if err != nil {
		return apperrors.Wrapf(err, "delete backup plan %s", id)
	}
	return nil
}

// BackupPlanResource represents an AWS Backup plan
type BackupPlanResource struct {
	dao.BaseResource
	Summary *types.BackupPlansListMember
	Detail  *backup.GetBackupPlanOutput
}

// NewBackupPlanResourceFromSummary creates a new BackupPlanResource from summary
func NewBackupPlanResourceFromSummary(summary types.BackupPlansListMember) *BackupPlanResource {
	id := appaws.Str(summary.BackupPlanId)
	name := appaws.Str(summary.BackupPlanName)
	arn := appaws.Str(summary.BackupPlanArn)

	return &BackupPlanResource{
		BaseResource: dao.BaseResource{
			ID:   id,
			Name: name,
			ARN:  arn,
			Tags: make(map[string]string),
			Data: summary,
		},
		Summary: &summary,
	}
}

// NewBackupPlanResourceFromDetail creates a new BackupPlanResource from detail
func NewBackupPlanResourceFromDetail(detail *backup.GetBackupPlanOutput) *BackupPlanResource {
	id := appaws.Str(detail.BackupPlanId)
	arn := appaws.Str(detail.BackupPlanArn)

	name := ""
	if detail.BackupPlan != nil {
		name = appaws.Str(detail.BackupPlan.BackupPlanName)
	}

	return &BackupPlanResource{
		BaseResource: dao.BaseResource{
			ID:   id,
			Name: name,
			ARN:  arn,
			Tags: make(map[string]string),
			Data: detail,
		},
		Detail: detail,
	}
}

// PlanId returns the backup plan ID
func (r *BackupPlanResource) PlanId() string {
	if r.Summary != nil {
		return appaws.Str(r.Summary.BackupPlanId)
	}
	if r.Detail != nil {
		return appaws.Str(r.Detail.BackupPlanId)
	}
	return ""
}

// PlanName returns the backup plan name
func (r *BackupPlanResource) PlanName() string {
	if r.Summary != nil {
		return appaws.Str(r.Summary.BackupPlanName)
	}
	if r.Detail != nil && r.Detail.BackupPlan != nil {
		return appaws.Str(r.Detail.BackupPlan.BackupPlanName)
	}
	return ""
}

// VersionId returns the backup plan version ID
func (r *BackupPlanResource) VersionId() string {
	if r.Summary != nil {
		return appaws.Str(r.Summary.VersionId)
	}
	if r.Detail != nil {
		return appaws.Str(r.Detail.VersionId)
	}
	return ""
}

// RuleCount returns the number of backup rules
func (r *BackupPlanResource) RuleCount() int {
	if r.Detail != nil && r.Detail.BackupPlan != nil {
		return len(r.Detail.BackupPlan.Rules)
	}
	return 0
}

// Rules returns the backup rules
func (r *BackupPlanResource) Rules() []types.BackupRule {
	if r.Detail != nil && r.Detail.BackupPlan != nil {
		return r.Detail.BackupPlan.Rules
	}
	return nil
}

// AdvancedBackupSettings returns the advanced backup settings
func (r *BackupPlanResource) AdvancedBackupSettings() []types.AdvancedBackupSetting {
	if r.Summary != nil {
		return r.Summary.AdvancedBackupSettings
	}
	if r.Detail != nil && r.Detail.BackupPlan != nil {
		return r.Detail.BackupPlan.AdvancedBackupSettings
	}
	return nil
}

// CreatedAt returns the creation date
func (r *BackupPlanResource) CreatedAt() string {
	if r.Summary != nil && r.Summary.CreationDate != nil {
		return r.Summary.CreationDate.Format("2006-01-02 15:04:05")
	}
	if r.Detail != nil && r.Detail.CreationDate != nil {
		return r.Detail.CreationDate.Format("2006-01-02 15:04:05")
	}
	return ""
}

// CreatedAtTime returns the creation date as time.Time
func (r *BackupPlanResource) CreatedAtTime() *time.Time {
	if r.Summary != nil && r.Summary.CreationDate != nil {
		return r.Summary.CreationDate
	}
	if r.Detail != nil && r.Detail.CreationDate != nil {
		return r.Detail.CreationDate
	}
	return nil
}

// LastExecutionDate returns the last execution date
func (r *BackupPlanResource) LastExecutionDate() string {
	if r.Summary != nil && r.Summary.LastExecutionDate != nil {
		return r.Summary.LastExecutionDate.Format("2006-01-02 15:04:05")
	}
	return ""
}

// DeletionDate returns the deletion date (if plan is being deleted)
func (r *BackupPlanResource) DeletionDate() string {
	if r.Summary != nil && r.Summary.DeletionDate != nil {
		return r.Summary.DeletionDate.Format("2006-01-02 15:04:05")
	}
	return ""
}
