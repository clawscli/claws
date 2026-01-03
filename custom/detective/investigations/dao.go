package investigations

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/detective"
	"github.com/aws/aws-sdk-go-v2/service/detective/types"

	appaws "github.com/clawscli/claws/internal/aws"
	"github.com/clawscli/claws/internal/dao"
	apperrors "github.com/clawscli/claws/internal/errors"
)

// InvestigationDAO provides data access for Detective investigations.
type InvestigationDAO struct {
	dao.BaseDAO
	client *detective.Client
}

// NewInvestigationDAO creates a new InvestigationDAO.
func NewInvestigationDAO(ctx context.Context) (dao.DAO, error) {
	cfg, err := appaws.NewConfig(ctx)
	if err != nil {
		return nil, apperrors.Wrap(err, "new "+ServiceResourcePath+" dao")
	}
	return &InvestigationDAO{
		BaseDAO: dao.NewBaseDAO("detective", "investigations"),
		client:  detective.NewFromConfig(cfg),
	}, nil
}

// List returns investigations for the specified graph.
func (d *InvestigationDAO) List(ctx context.Context) ([]dao.Resource, error) {
	graphArn := dao.GetFilterFromContext(ctx, "GraphArn")
	if graphArn == "" {
		return nil, fmt.Errorf("graph ARN filter required")
	}

	investigations, err := appaws.Paginate(ctx, func(token *string) ([]types.InvestigationDetail, *string, error) {
		output, err := d.client.ListInvestigations(ctx, &detective.ListInvestigationsInput{
			GraphArn:  &graphArn,
			NextToken: token,
		})
		if err != nil {
			return nil, nil, apperrors.Wrap(err, "list detective investigations")
		}
		return output.InvestigationDetails, output.NextToken, nil
	})
	if err != nil {
		return nil, err
	}

	resources := make([]dao.Resource, len(investigations))
	for i, inv := range investigations {
		resources[i] = NewInvestigationResource(inv, graphArn)
	}
	return resources, nil
}

// Get returns a specific investigation.
func (d *InvestigationDAO) Get(ctx context.Context, id string) (dao.Resource, error) {
	graphArn := dao.GetFilterFromContext(ctx, "GraphArn")
	if graphArn == "" {
		return nil, fmt.Errorf("graph ARN filter required")
	}

	output, err := d.client.GetInvestigation(ctx, &detective.GetInvestigationInput{
		GraphArn:        &graphArn,
		InvestigationId: &id,
	})
	if err != nil {
		return nil, apperrors.Wrap(err, "get detective investigation")
	}

	return &InvestigationResource{
		BaseResource: dao.BaseResource{
			ID:  id,
			ARN: graphArn + "/investigation/" + id,
		},
		Investigation: &types.InvestigationDetail{
			InvestigationId: output.InvestigationId,
			EntityArn:       output.EntityArn,
			EntityType:      output.EntityType,
			Severity:        output.Severity,
			Status:          output.Status,
			State:           output.State,
			CreatedTime:     output.CreatedTime,
		},
		graphArn: graphArn,
	}, nil
}

// Delete is not supported for investigations.
func (d *InvestigationDAO) Delete(ctx context.Context, id string) error {
	return fmt.Errorf("delete not supported for detective investigations")
}

// InvestigationResource wraps a Detective investigation.
type InvestigationResource struct {
	dao.BaseResource
	Investigation *types.InvestigationDetail
	graphArn      string
}

// NewInvestigationResource creates a new InvestigationResource.
func NewInvestigationResource(inv types.InvestigationDetail, graphArn string) *InvestigationResource {
	id := appaws.Str(inv.InvestigationId)
	return &InvestigationResource{
		BaseResource: dao.BaseResource{
			ID:  id,
			ARN: graphArn + "/investigation/" + id,
		},
		Investigation: &inv,
		graphArn:      graphArn,
	}
}

// EntityArn returns the entity ARN.
func (r *InvestigationResource) EntityArn() string {
	if r.Investigation != nil {
		return appaws.Str(r.Investigation.EntityArn)
	}
	return ""
}

// EntityType returns the entity type.
func (r *InvestigationResource) EntityType() string {
	if r.Investigation != nil {
		return string(r.Investigation.EntityType)
	}
	return ""
}

// Severity returns the severity.
func (r *InvestigationResource) Severity() string {
	if r.Investigation != nil {
		return string(r.Investigation.Severity)
	}
	return ""
}

// Status returns the status.
func (r *InvestigationResource) Status() string {
	if r.Investigation != nil {
		return string(r.Investigation.Status)
	}
	return ""
}

// State returns the state.
func (r *InvestigationResource) State() string {
	if r.Investigation != nil {
		return string(r.Investigation.State)
	}
	return ""
}

// CreatedTime returns when the investigation was created.
func (r *InvestigationResource) CreatedTime() *time.Time {
	if r.Investigation != nil {
		return r.Investigation.CreatedTime
	}
	return nil
}
