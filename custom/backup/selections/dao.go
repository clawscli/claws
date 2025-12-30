package selections

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/backup"
	"github.com/aws/aws-sdk-go-v2/service/backup/types"

	appaws "github.com/clawscli/claws/internal/aws"
	"github.com/clawscli/claws/internal/dao"
)

// SelectionDAO provides data access for AWS Backup selections
type SelectionDAO struct {
	dao.BaseDAO
	client *backup.Client
}

// NewSelectionDAO creates a new SelectionDAO
func NewSelectionDAO(ctx context.Context) (dao.DAO, error) {
	cfg, err := appaws.NewConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("new backup/selections dao: %w", err)
	}
	return &SelectionDAO{
		BaseDAO: dao.NewBaseDAO("backup", "selections"),
		client:  backup.NewFromConfig(cfg),
	}, nil
}

// List returns backup selections for a plan
func (d *SelectionDAO) List(ctx context.Context) ([]dao.Resource, error) {
	backupPlanId := dao.GetFilterFromContext(ctx, "BackupPlanId")
	if backupPlanId == "" {
		return nil, fmt.Errorf("navigate from plans using 's' key")
	}

	selections, err := appaws.Paginate(ctx, func(token *string) ([]types.BackupSelectionsListMember, *string, error) {
		output, err := d.client.ListBackupSelections(ctx, &backup.ListBackupSelectionsInput{
			BackupPlanId: &backupPlanId,
			NextToken:    token,
		})
		if err != nil {
			return nil, nil, fmt.Errorf("list backup selections: %w", err)
		}
		return output.BackupSelectionsList, output.NextToken, nil
	})
	if err != nil {
		return nil, err
	}

	resources := make([]dao.Resource, len(selections))
	for i, sel := range selections {
		resources[i] = NewSelectionResourceFromSummary(sel, backupPlanId)
	}

	return resources, nil
}

// Get returns a specific backup selection
func (d *SelectionDAO) Get(ctx context.Context, selectionId string) (dao.Resource, error) {
	backupPlanId := dao.GetFilterFromContext(ctx, "BackupPlanId")
	if backupPlanId == "" {
		return nil, fmt.Errorf("navigate from plans using 's' key")
	}

	output, err := d.client.GetBackupSelection(ctx, &backup.GetBackupSelectionInput{
		BackupPlanId: &backupPlanId,
		SelectionId:  &selectionId,
	})
	if err != nil {
		return nil, fmt.Errorf("get backup selection %s: %w", selectionId, err)
	}

	return NewSelectionResourceFromDetail(output, backupPlanId), nil
}

// Delete deletes a backup selection
func (d *SelectionDAO) Delete(ctx context.Context, selectionId string) error {
	backupPlanId := dao.GetFilterFromContext(ctx, "BackupPlanId")
	if backupPlanId == "" {
		return fmt.Errorf("navigate from plans using 's' key")
	}

	_, err := d.client.DeleteBackupSelection(ctx, &backup.DeleteBackupSelectionInput{
		BackupPlanId: &backupPlanId,
		SelectionId:  &selectionId,
	})
	if err != nil {
		return fmt.Errorf("delete backup selection %s: %w", selectionId, err)
	}
	return nil
}

// SelectionResource represents an AWS Backup selection
type SelectionResource struct {
	dao.BaseResource
	Summary      *types.BackupSelectionsListMember
	Detail       *backup.GetBackupSelectionOutput
	BackupPlanId string
}

// NewSelectionResourceFromSummary creates a new SelectionResource from summary
func NewSelectionResourceFromSummary(summary types.BackupSelectionsListMember, backupPlanId string) *SelectionResource {
	id := appaws.Str(summary.SelectionId)
	name := appaws.Str(summary.SelectionName)

	return &SelectionResource{
		BaseResource: dao.BaseResource{
			ID:   id,
			Name: name,
			ARN:  "",
			Tags: make(map[string]string),
			Data: summary,
		},
		Summary:      &summary,
		BackupPlanId: backupPlanId,
	}
}

// NewSelectionResourceFromDetail creates a new SelectionResource from detail
func NewSelectionResourceFromDetail(detail *backup.GetBackupSelectionOutput, backupPlanId string) *SelectionResource {
	id := appaws.Str(detail.SelectionId)
	name := ""
	if detail.BackupSelection != nil {
		name = appaws.Str(detail.BackupSelection.SelectionName)
	}

	return &SelectionResource{
		BaseResource: dao.BaseResource{
			ID:   id,
			Name: name,
			ARN:  "",
			Tags: make(map[string]string),
			Data: detail,
		},
		Detail:       detail,
		BackupPlanId: backupPlanId,
	}
}

// SelectionId returns the selection ID
func (r *SelectionResource) SelectionId() string {
	if r.Summary != nil {
		return appaws.Str(r.Summary.SelectionId)
	}
	if r.Detail != nil {
		return appaws.Str(r.Detail.SelectionId)
	}
	return ""
}

// SelectionName returns the selection name
func (r *SelectionResource) SelectionName() string {
	if r.Summary != nil {
		return appaws.Str(r.Summary.SelectionName)
	}
	if r.Detail != nil && r.Detail.BackupSelection != nil {
		return appaws.Str(r.Detail.BackupSelection.SelectionName)
	}
	return ""
}

// IamRoleArn returns the IAM role ARN
func (r *SelectionResource) IamRoleArn() string {
	if r.Summary != nil {
		return appaws.Str(r.Summary.IamRoleArn)
	}
	if r.Detail != nil && r.Detail.BackupSelection != nil {
		return appaws.Str(r.Detail.BackupSelection.IamRoleArn)
	}
	return ""
}

// CreationDate returns the creation date
func (r *SelectionResource) CreationDate() string {
	if r.Summary != nil && r.Summary.CreationDate != nil {
		return r.Summary.CreationDate.Format("2006-01-02 15:04:05")
	}
	if r.Detail != nil && r.Detail.CreationDate != nil {
		return r.Detail.CreationDate.Format("2006-01-02 15:04:05")
	}
	return ""
}

// Resources returns the list of resource ARNs
func (r *SelectionResource) Resources() []string {
	if r.Detail != nil && r.Detail.BackupSelection != nil {
		return r.Detail.BackupSelection.Resources
	}
	return nil
}

// NotResources returns the list of excluded resource ARNs
func (r *SelectionResource) NotResources() []string {
	if r.Detail != nil && r.Detail.BackupSelection != nil {
		return r.Detail.BackupSelection.NotResources
	}
	return nil
}

// ListOfTags returns the list of tag conditions
func (r *SelectionResource) ListOfTags() []types.Condition {
	if r.Detail != nil && r.Detail.BackupSelection != nil {
		return r.Detail.BackupSelection.ListOfTags
	}
	return nil
}

// Conditions returns the conditions
func (r *SelectionResource) Conditions() *types.Conditions {
	if r.Detail != nil && r.Detail.BackupSelection != nil {
		return r.Detail.BackupSelection.Conditions
	}
	return nil
}
