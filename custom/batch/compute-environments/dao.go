package computeenvironments

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/batch"
	"github.com/aws/aws-sdk-go-v2/service/batch/types"

	appaws "github.com/clawscli/claws/internal/aws"
	"github.com/clawscli/claws/internal/dao"
	apperrors "github.com/clawscli/claws/internal/errors"
)

// ComputeEnvironmentDAO provides data access for Batch compute environments.
type ComputeEnvironmentDAO struct {
	dao.BaseDAO
	client *batch.Client
}

// NewComputeEnvironmentDAO creates a new ComputeEnvironmentDAO.
func NewComputeEnvironmentDAO(ctx context.Context) (dao.DAO, error) {
	cfg, err := appaws.NewConfig(ctx)
	if err != nil {
		return nil, apperrors.Wrap(err, "new batch/compute-environments dao")
	}
	return &ComputeEnvironmentDAO{
		BaseDAO: dao.NewBaseDAO("batch", "compute-environments"),
		client:  batch.NewFromConfig(cfg),
	}, nil
}

// List returns all Batch compute environments.
func (d *ComputeEnvironmentDAO) List(ctx context.Context) ([]dao.Resource, error) {
	envs, err := appaws.Paginate(ctx, func(token *string) ([]types.ComputeEnvironmentDetail, *string, error) {
		output, err := d.client.DescribeComputeEnvironments(ctx, &batch.DescribeComputeEnvironmentsInput{
			NextToken: token,
		})
		if err != nil {
			return nil, nil, apperrors.Wrap(err, "describe batch compute environments")
		}
		return output.ComputeEnvironments, output.NextToken, nil
	})
	if err != nil {
		return nil, err
	}

	resources := make([]dao.Resource, len(envs))
	for i, env := range envs {
		resources[i] = NewComputeEnvironmentResource(env)
	}
	return resources, nil
}

// Get returns a specific compute environment.
func (d *ComputeEnvironmentDAO) Get(ctx context.Context, name string) (dao.Resource, error) {
	output, err := d.client.DescribeComputeEnvironments(ctx, &batch.DescribeComputeEnvironmentsInput{
		ComputeEnvironments: []string{name},
	})
	if err != nil {
		return nil, apperrors.Wrap(err, "describe batch compute environment")
	}
	if len(output.ComputeEnvironments) == 0 {
		return nil, fmt.Errorf("compute environment not found: %s", name)
	}
	return NewComputeEnvironmentResource(output.ComputeEnvironments[0]), nil
}

// Delete deletes a Batch compute environment.
func (d *ComputeEnvironmentDAO) Delete(ctx context.Context, name string) error {
	// First disable the environment
	_, err := d.client.UpdateComputeEnvironment(ctx, &batch.UpdateComputeEnvironmentInput{
		ComputeEnvironment: &name,
		State:              types.CEStateDisabled,
	})
	if err != nil {
		return apperrors.Wrap(err, "disable batch compute environment")
	}

	// Then delete it
	_, err = d.client.DeleteComputeEnvironment(ctx, &batch.DeleteComputeEnvironmentInput{
		ComputeEnvironment: &name,
	})
	if err != nil {
		return apperrors.Wrap(err, "delete batch compute environment")
	}
	return nil
}

// ComputeEnvironmentResource wraps a Batch compute environment.
type ComputeEnvironmentResource struct {
	dao.BaseResource
	Env *types.ComputeEnvironmentDetail
}

// NewComputeEnvironmentResource creates a new ComputeEnvironmentResource.
func NewComputeEnvironmentResource(env types.ComputeEnvironmentDetail) *ComputeEnvironmentResource {
	return &ComputeEnvironmentResource{
		BaseResource: dao.BaseResource{
			ID:  appaws.Str(env.ComputeEnvironmentName),
			ARN: appaws.Str(env.ComputeEnvironmentArn),
		},
		Env: &env,
	}
}

// State returns the environment state.
func (r *ComputeEnvironmentResource) State() string {
	if r.Env != nil {
		return string(r.Env.State)
	}
	return ""
}

// Status returns the environment status.
func (r *ComputeEnvironmentResource) Status() string {
	if r.Env != nil {
		return string(r.Env.Status)
	}
	return ""
}

// Type returns the environment type.
func (r *ComputeEnvironmentResource) Type() string {
	if r.Env != nil {
		return string(r.Env.Type)
	}
	return ""
}

// ServiceRole returns the service role.
func (r *ComputeEnvironmentResource) ServiceRole() string {
	if r.Env != nil && r.Env.ServiceRole != nil {
		return *r.Env.ServiceRole
	}
	return ""
}
