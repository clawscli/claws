package stagesv2

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/apigatewayv2"
	"github.com/aws/aws-sdk-go-v2/service/apigatewayv2/types"

	appaws "github.com/clawscli/claws/internal/aws"
	"github.com/clawscli/claws/internal/dao"
	apperrors "github.com/clawscli/claws/internal/errors"
)

// StageV2DAO provides data access for API Gateway HTTP/WebSocket API stages (v2)
type StageV2DAO struct {
	dao.BaseDAO
	client *apigatewayv2.Client
}

// NewStageV2DAO creates a new StageV2DAO
func NewStageV2DAO(ctx context.Context) (dao.DAO, error) {
	cfg, err := appaws.NewConfig(ctx)
	if err != nil {
		return nil, apperrors.Wrap(err, "new apigateway/stages-v2 dao")
	}
	return &StageV2DAO{
		BaseDAO: dao.NewBaseDAO("apigateway", "stages-v2"),
		client:  apigatewayv2.NewFromConfig(cfg),
	}, nil
}

// List returns all stages for an HTTP/WebSocket API (requires ApiId filter)
func (d *StageV2DAO) List(ctx context.Context) ([]dao.Resource, error) {
	apiId := dao.GetFilterFromContext(ctx, "ApiId")
	if apiId == "" {
		return nil, fmt.Errorf("ApiId filter required - navigate from an HTTP API")
	}

	stages, err := appaws.Paginate(ctx, func(token *string) ([]types.Stage, *string, error) {
		output, err := d.client.GetStages(ctx, &apigatewayv2.GetStagesInput{
			ApiId:     &apiId,
			NextToken: token,
		})
		if err != nil {
			return nil, nil, apperrors.Wrap(err, "list stages")
		}
		return output.Items, output.NextToken, nil
	})
	if err != nil {
		return nil, err
	}

	resources := make([]dao.Resource, len(stages))
	for i, stage := range stages {
		resources[i] = NewStageV2Resource(stage, apiId)
	}

	return resources, nil
}

// Get returns a specific stage
func (d *StageV2DAO) Get(ctx context.Context, id string) (dao.Resource, error) {
	apiId, stageName, err := parseStageV2id(id)
	if err != nil {
		return nil, err
	}

	output, err := d.client.GetStage(ctx, &apigatewayv2.GetStageInput{
		ApiId:     &apiId,
		StageName: &stageName,
	})
	if err != nil {
		return nil, apperrors.Wrapf(err, "get stage %s", id)
	}

	return NewStageV2ResourceFromGetOutput(output, apiId), nil
}

// Delete deletes a stage
func (d *StageV2DAO) Delete(ctx context.Context, id string) error {
	apiId, stageName, err := parseStageV2id(id)
	if err != nil {
		return err
	}

	_, err = d.client.DeleteStage(ctx, &apigatewayv2.DeleteStageInput{
		ApiId:     &apiId,
		StageName: &stageName,
	})
	if err != nil {
		return apperrors.Wrapf(err, "delete stage %s", id)
	}
	return nil
}

func parseStageV2id(id string) (apiId, stageName string, err error) {
	for i := len(id) - 1; i >= 0; i-- {
		if id[i] == ':' {
			return id[:i], id[i+1:], nil
		}
	}
	return "", "", fmt.Errorf("invalid stage ID format: %s (expected apiId:stageName)", id)
}

// StageV2Resource wraps an API Gateway HTTP/WebSocket stage (v2)
type StageV2Resource struct {
	dao.BaseResource
	Item  types.Stage
	ApiId string
}

// NewStageV2Resource creates a new StageV2Resource
func NewStageV2Resource(stage types.Stage, apiId string) *StageV2Resource {
	stageName := appaws.Str(stage.StageName)

	// ID format: apiId:stageName
	id := fmt.Sprintf("%s:%s", apiId, stageName)

	// Convert tags
	tags := make(map[string]string)
	for k, v := range stage.Tags {
		tags[k] = v
	}

	return &StageV2Resource{
		BaseResource: dao.BaseResource{
			ID:   id,
			Name: stageName,
			Tags: tags,
			Data: stage,
		},
		Item:  stage,
		ApiId: apiId,
	}
}

// NewStageV2ResourceFromGetOutput creates a StageV2Resource from GetStage output
func NewStageV2ResourceFromGetOutput(output *apigatewayv2.GetStageOutput, apiId string) *StageV2Resource {
	stage := types.Stage{
		StageName:                   output.StageName,
		DeploymentId:                output.DeploymentId,
		Description:                 output.Description,
		ApiGatewayManaged:           output.ApiGatewayManaged,
		AutoDeploy:                  output.AutoDeploy,
		CreatedDate:                 output.CreatedDate,
		LastUpdatedDate:             output.LastUpdatedDate,
		DefaultRouteSettings:        output.DefaultRouteSettings,
		AccessLogSettings:           output.AccessLogSettings,
		ClientCertificateId:         output.ClientCertificateId,
		LastDeploymentStatusMessage: output.LastDeploymentStatusMessage,
		RouteSettings:               output.RouteSettings,
		StageVariables:              output.StageVariables,
		Tags:                        output.Tags,
	}
	return NewStageV2Resource(stage, apiId)
}

// StageName returns the stage name
func (r *StageV2Resource) StageName() string {
	if r.Item.StageName != nil {
		return *r.Item.StageName
	}
	return ""
}

// DeploymentId returns the deployment ID
func (r *StageV2Resource) DeploymentId() string {
	if r.Item.DeploymentId != nil {
		return *r.Item.DeploymentId
	}
	return ""
}

// Description returns the stage description
func (r *StageV2Resource) Description() string {
	if r.Item.Description != nil {
		return *r.Item.Description
	}
	return ""
}

// AutoDeploy returns whether auto deploy is enabled
func (r *StageV2Resource) AutoDeploy() bool {
	if r.Item.AutoDeploy != nil {
		return *r.Item.AutoDeploy
	}
	return false
}

// ApiGatewayManaged returns whether the stage is managed by API Gateway
func (r *StageV2Resource) ApiGatewayManaged() bool {
	if r.Item.ApiGatewayManaged != nil {
		return *r.Item.ApiGatewayManaged
	}
	return false
}

// CreatedDate returns the creation date
func (r *StageV2Resource) CreatedDate() time.Time {
	if r.Item.CreatedDate != nil {
		return *r.Item.CreatedDate
	}
	return time.Time{}
}

// LastUpdatedDate returns the last updated date
func (r *StageV2Resource) LastUpdatedDate() time.Time {
	if r.Item.LastUpdatedDate != nil {
		return *r.Item.LastUpdatedDate
	}
	return time.Time{}
}

// HasAccessLogs returns whether access logging is configured
func (r *StageV2Resource) HasAccessLogs() bool {
	return r.Item.AccessLogSettings != nil && r.Item.AccessLogSettings.DestinationArn != nil
}

// AccessLogDestination returns the access log destination ARN
func (r *StageV2Resource) AccessLogDestination() string {
	if r.Item.AccessLogSettings != nil && r.Item.AccessLogSettings.DestinationArn != nil {
		return *r.Item.AccessLogSettings.DestinationArn
	}
	return ""
}

// StageVariables returns the stage variables
func (r *StageV2Resource) StageVariables() map[string]string {
	return r.Item.StageVariables
}

// ThrottlingBurstLimit returns the default throttling burst limit
func (r *StageV2Resource) ThrottlingBurstLimit() int32 {
	if r.Item.DefaultRouteSettings != nil && r.Item.DefaultRouteSettings.ThrottlingBurstLimit != nil {
		return *r.Item.DefaultRouteSettings.ThrottlingBurstLimit
	}
	return 0
}

// ThrottlingRateLimit returns the default throttling rate limit
func (r *StageV2Resource) ThrottlingRateLimit() float64 {
	if r.Item.DefaultRouteSettings != nil && r.Item.DefaultRouteSettings.ThrottlingRateLimit != nil {
		return *r.Item.DefaultRouteSettings.ThrottlingRateLimit
	}
	return 0
}
