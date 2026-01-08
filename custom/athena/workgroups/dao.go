package workgroups

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/athena"
	"github.com/aws/aws-sdk-go-v2/service/athena/types"

	appaws "github.com/clawscli/claws/internal/aws"
	"github.com/clawscli/claws/internal/dao"
	apperrors "github.com/clawscli/claws/internal/errors"
)

// WorkgroupDAO provides data access for Athena workgroups.
type WorkgroupDAO struct {
	dao.BaseDAO
	client *athena.Client
}

// NewWorkgroupDAO creates a new WorkgroupDAO.
func NewWorkgroupDAO(ctx context.Context) (dao.DAO, error) {
	cfg, err := appaws.NewConfig(ctx)
	if err != nil {
		return nil, apperrors.Wrap(err, "new "+ServiceResourcePath+" dao")
	}
	return &WorkgroupDAO{
		BaseDAO: dao.NewBaseDAO("athena", "workgroups"),
		client:  athena.NewFromConfig(cfg),
	}, nil
}

// List returns all Athena workgroups.
func (d *WorkgroupDAO) List(ctx context.Context) ([]dao.Resource, error) {
	workgroups, err := appaws.Paginate(ctx, func(token *string) ([]types.WorkGroupSummary, *string, error) {
		output, err := d.client.ListWorkGroups(ctx, &athena.ListWorkGroupsInput{
			NextToken: token,
		})
		if err != nil {
			return nil, nil, apperrors.Wrap(err, "list athena workgroups")
		}
		return output.WorkGroups, output.NextToken, nil
	})
	if err != nil {
		return nil, err
	}

	resources := make([]dao.Resource, len(workgroups))
	for i, wg := range workgroups {
		resources[i] = NewWorkgroupResource(wg)
	}
	return resources, nil
}

// Get returns a specific Athena workgroup by name.
func (d *WorkgroupDAO) Get(ctx context.Context, id string) (dao.Resource, error) {
	output, err := d.client.GetWorkGroup(ctx, &athena.GetWorkGroupInput{
		WorkGroup: &id,
	})
	if err != nil {
		return nil, apperrors.Wrapf(err, "get athena workgroup %s", id)
	}
	return NewWorkgroupResourceFromDetail(*output.WorkGroup), nil
}

// Delete deletes an Athena workgroup by name.
func (d *WorkgroupDAO) Delete(ctx context.Context, id string) error {
	_, err := d.client.DeleteWorkGroup(ctx, &athena.DeleteWorkGroupInput{
		WorkGroup: &id,
	})
	if err != nil {
		return apperrors.Wrapf(err, "delete athena workgroup %s", id)
	}
	return nil
}

// WorkgroupResource wraps an Athena workgroup.
type WorkgroupResource struct {
	dao.BaseResource
	Summary *types.WorkGroupSummary
	Detail  *types.WorkGroup
}

// NewWorkgroupResource creates a new WorkgroupResource from summary.
func NewWorkgroupResource(wg types.WorkGroupSummary) *WorkgroupResource {
	return &WorkgroupResource{
		BaseResource: dao.BaseResource{
			ID:   appaws.Str(wg.Name),
			ARN:  "",
			Data: wg,
		},
		Summary: &wg,
	}
}

// NewWorkgroupResourceFromDetail creates a new WorkgroupResource from detail.
func NewWorkgroupResourceFromDetail(wg types.WorkGroup) *WorkgroupResource {
	return &WorkgroupResource{
		BaseResource: dao.BaseResource{
			ID:   appaws.Str(wg.Name),
			ARN:  "",
			Data: wg,
		},
		Detail: &wg,
	}
}

// Name returns the workgroup name.
func (r *WorkgroupResource) Name() string {
	return r.ID
}

// State returns the workgroup state.
func (r *WorkgroupResource) State() string {
	if r.Summary != nil {
		return string(r.Summary.State)
	}
	if r.Detail != nil {
		return string(r.Detail.State)
	}
	return ""
}

// Description returns the workgroup description.
func (r *WorkgroupResource) Description() string {
	if r.Summary != nil {
		return appaws.Str(r.Summary.Description)
	}
	if r.Detail != nil {
		return appaws.Str(r.Detail.Description)
	}
	return ""
}

// EngineVersion returns the engine version.
func (r *WorkgroupResource) EngineVersion() string {
	if r.Summary != nil && r.Summary.EngineVersion != nil {
		return appaws.Str(r.Summary.EngineVersion.EffectiveEngineVersion)
	}
	if r.Detail != nil && r.Detail.Configuration != nil && r.Detail.Configuration.EngineVersion != nil {
		return appaws.Str(r.Detail.Configuration.EngineVersion.EffectiveEngineVersion)
	}
	return ""
}

// CreationTime returns when the workgroup was created.
func (r *WorkgroupResource) CreationTime() *time.Time {
	if r.Summary != nil {
		return r.Summary.CreationTime
	}
	if r.Detail != nil {
		return r.Detail.CreationTime
	}
	return nil
}

// OutputLocation returns the output location.
func (r *WorkgroupResource) OutputLocation() string {
	if r.Detail != nil && r.Detail.Configuration != nil && r.Detail.Configuration.ResultConfiguration != nil {
		return appaws.Str(r.Detail.Configuration.ResultConfiguration.OutputLocation)
	}
	return ""
}
