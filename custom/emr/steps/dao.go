package steps

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/emr"
	"github.com/aws/aws-sdk-go-v2/service/emr/types"

	appaws "github.com/clawscli/claws/internal/aws"
	"github.com/clawscli/claws/internal/dao"
	apperrors "github.com/clawscli/claws/internal/errors"
)

// StepDAO provides data access for EMR steps.
type StepDAO struct {
	dao.BaseDAO
	client *emr.Client
}

// NewStepDAO creates a new StepDAO.
func NewStepDAO(ctx context.Context) (dao.DAO, error) {
	cfg, err := appaws.NewConfig(ctx)
	if err != nil {
		return nil, apperrors.Wrap(err, "new "+ServiceResourcePath+" dao")
	}
	return &StepDAO{
		BaseDAO: dao.NewBaseDAO("emr", "steps"),
		client:  emr.NewFromConfig(cfg),
	}, nil
}

// List returns steps for the specified cluster.
func (d *StepDAO) List(ctx context.Context) ([]dao.Resource, error) {
	clusterId := dao.GetFilterFromContext(ctx, "ClusterId")
	if clusterId == "" {
		return nil, fmt.Errorf("cluster ID filter required")
	}

	steps, err := appaws.Paginate(ctx, func(token *string) ([]types.StepSummary, *string, error) {
		output, err := d.client.ListSteps(ctx, &emr.ListStepsInput{
			ClusterId: &clusterId,
			Marker:    token,
		})
		if err != nil {
			return nil, nil, apperrors.Wrap(err, "list emr steps")
		}
		return output.Steps, output.Marker, nil
	})
	if err != nil {
		return nil, err
	}

	resources := make([]dao.Resource, len(steps))
	for i, step := range steps {
		resources[i] = NewStepResource(step, clusterId)
	}
	return resources, nil
}

// Get returns a specific step.
func (d *StepDAO) Get(ctx context.Context, id string) (dao.Resource, error) {
	clusterId := dao.GetFilterFromContext(ctx, "ClusterId")
	if clusterId == "" {
		return nil, fmt.Errorf("cluster ID filter required")
	}

	output, err := d.client.DescribeStep(ctx, &emr.DescribeStepInput{
		ClusterId: &clusterId,
		StepId:    &id,
	})
	if err != nil {
		return nil, apperrors.Wrap(err, "describe emr step")
	}

	step := output.Step
	return &StepResource{
		BaseResource: dao.BaseResource{
			ID:   appaws.Str(step.Id),
			ARN:  "",
			Data: step,
		},
		Step: &types.StepSummary{
			Id:     step.Id,
			Name:   step.Name,
			Status: step.Status,
		},
		clusterId:       clusterId,
		ActionOnFailure: string(step.ActionOnFailure),
	}, nil
}

// Delete cancels an EMR step.
func (d *StepDAO) Delete(ctx context.Context, id string) error {
	clusterId := dao.GetFilterFromContext(ctx, "ClusterId")
	if clusterId == "" {
		return fmt.Errorf("cluster ID filter required")
	}

	_, err := d.client.CancelSteps(ctx, &emr.CancelStepsInput{
		ClusterId: &clusterId,
		StepIds:   []string{id},
	})
	if err != nil {
		return apperrors.Wrap(err, "cancel emr step")
	}
	return nil
}

// StepResource wraps an EMR step.
type StepResource struct {
	dao.BaseResource
	Step            *types.StepSummary
	clusterId       string
	ActionOnFailure string
}

// NewStepResource creates a new StepResource.
func NewStepResource(step types.StepSummary, clusterId string) *StepResource {
	return &StepResource{
		BaseResource: dao.BaseResource{
			ID:   appaws.Str(step.Id),
			ARN:  "",
			Data: step,
		},
		Step:      &step,
		clusterId: clusterId,
	}
}

// Name returns the step name.
func (r *StepResource) Name() string {
	if r.Step != nil && r.Step.Name != nil {
		return *r.Step.Name
	}
	return ""
}

// State returns the step state.
func (r *StepResource) State() string {
	if r.Step != nil && r.Step.Status != nil {
		return string(r.Step.Status.State)
	}
	return ""
}

// StateReason returns the reason for the step state.
func (r *StepResource) StateReason() string {
	if r.Step != nil && r.Step.Status != nil && r.Step.Status.StateChangeReason != nil && r.Step.Status.StateChangeReason.Message != nil {
		return *r.Step.Status.StateChangeReason.Message
	}
	return ""
}
