package fleets

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/gamelift"
	"github.com/aws/aws-sdk-go-v2/service/gamelift/types"

	appaws "github.com/clawscli/claws/internal/aws"
	"github.com/clawscli/claws/internal/dao"
	apperrors "github.com/clawscli/claws/internal/errors"
)

// FleetDAO provides data access for GameLift fleets.
type FleetDAO struct {
	dao.BaseDAO
	client *gamelift.Client
}

// NewFleetDAO creates a new FleetDAO.
func NewFleetDAO(ctx context.Context) (dao.DAO, error) {
	cfg, err := appaws.NewConfig(ctx)
	if err != nil {
		return nil, apperrors.Wrap(err, "new "+ServiceResourcePath+" dao")
	}
	return &FleetDAO{
		BaseDAO: dao.NewBaseDAO("gamelift", "fleets"),
		client:  gamelift.NewFromConfig(cfg),
	}, nil
}

// List returns all GameLift fleets.
func (d *FleetDAO) List(ctx context.Context) ([]dao.Resource, error) {
	attrs, err := appaws.Paginate(ctx, func(token *string) ([]types.FleetAttributes, *string, error) {
		output, err := d.client.DescribeFleetAttributes(ctx, &gamelift.DescribeFleetAttributesInput{
			NextToken: token,
		})
		if err != nil {
			return nil, nil, apperrors.Wrap(err, "describe gamelift fleet attributes")
		}
		return output.FleetAttributes, output.NextToken, nil
	})
	if err != nil {
		return nil, err
	}

	resources := make([]dao.Resource, len(attrs))
	for i, attr := range attrs {
		resources[i] = NewFleetResource(attr)
	}
	return resources, nil
}

// Get returns a specific GameLift fleet by ID.
func (d *FleetDAO) Get(ctx context.Context, id string) (dao.Resource, error) {
	output, err := d.client.DescribeFleetAttributes(ctx, &gamelift.DescribeFleetAttributesInput{
		FleetIds: []string{id},
	})
	if err != nil {
		return nil, apperrors.Wrapf(err, "describe gamelift fleet %s", id)
	}
	if len(output.FleetAttributes) == 0 {
		return nil, fmt.Errorf("gamelift fleet %s not found", id)
	}
	return NewFleetResource(output.FleetAttributes[0]), nil
}

// Delete deletes a GameLift fleet by ID.
func (d *FleetDAO) Delete(ctx context.Context, id string) error {
	_, err := d.client.DeleteFleet(ctx, &gamelift.DeleteFleetInput{
		FleetId: &id,
	})
	if err != nil {
		return apperrors.Wrapf(err, "delete gamelift fleet %s", id)
	}
	return nil
}

// FleetResource wraps a GameLift fleet.
type FleetResource struct {
	dao.BaseResource
	Fleet types.FleetAttributes
}

// NewFleetResource creates a new FleetResource.
func NewFleetResource(fleet types.FleetAttributes) *FleetResource {
	return &FleetResource{
		BaseResource: dao.BaseResource{
			ID:   appaws.Str(fleet.FleetId),
			Name: appaws.Str(fleet.Name),
			ARN:  appaws.Str(fleet.FleetArn),
			Data: fleet,
		},
		Fleet: fleet,
	}
}

// Status returns the fleet status.
func (r *FleetResource) Status() string {
	return string(r.Fleet.Status)
}

// FleetType returns the fleet type (ON_DEMAND or SPOT).
func (r *FleetResource) FleetType() string {
	return string(r.Fleet.FleetType)
}

// InstanceType returns the EC2 instance type.
func (r *FleetResource) InstanceType() string {
	return string(r.Fleet.InstanceType)
}

// ComputeType returns the compute type.
func (r *FleetResource) ComputeType() string {
	return string(r.Fleet.ComputeType)
}

// OperatingSystem returns the OS.
func (r *FleetResource) OperatingSystem() string {
	return string(r.Fleet.OperatingSystem)
}

// BuildId returns the build ID.
func (r *FleetResource) BuildId() string {
	return appaws.Str(r.Fleet.BuildId)
}

// BuildArn returns the build ARN.
func (r *FleetResource) BuildArn() string {
	return appaws.Str(r.Fleet.BuildArn)
}

// ScriptId returns the script ID.
func (r *FleetResource) ScriptId() string {
	return appaws.Str(r.Fleet.ScriptId)
}

// ScriptArn returns the script ARN.
func (r *FleetResource) ScriptArn() string {
	return appaws.Str(r.Fleet.ScriptArn)
}

// Description returns the fleet description.
func (r *FleetResource) Description() string {
	return appaws.Str(r.Fleet.Description)
}

// CreationTime returns when the fleet was created.
func (r *FleetResource) CreationTime() *time.Time {
	return r.Fleet.CreationTime
}

// TerminationTime returns when the fleet was terminated.
func (r *FleetResource) TerminationTime() *time.Time {
	return r.Fleet.TerminationTime
}

// InstanceRoleArn returns the instance role ARN.
func (r *FleetResource) InstanceRoleArn() string {
	return appaws.Str(r.Fleet.InstanceRoleArn)
}

// ProtectionPolicy returns the new game session protection policy.
func (r *FleetResource) ProtectionPolicy() string {
	return string(r.Fleet.NewGameSessionProtectionPolicy)
}

// MetricGroups returns the metric groups.
func (r *FleetResource) MetricGroups() []string {
	return r.Fleet.MetricGroups
}

// StoppedActions returns the stopped fleet actions.
func (r *FleetResource) StoppedActions() []string {
	actions := make([]string, len(r.Fleet.StoppedActions))
	for i, a := range r.Fleet.StoppedActions {
		actions[i] = string(a)
	}
	return actions
}

// CertificateType returns the certificate type.
func (r *FleetResource) CertificateType() string {
	if r.Fleet.CertificateConfiguration != nil {
		return string(r.Fleet.CertificateConfiguration.CertificateType)
	}
	return ""
}
