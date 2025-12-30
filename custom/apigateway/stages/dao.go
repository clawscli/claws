package stages

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/apigateway"
	"github.com/aws/aws-sdk-go-v2/service/apigateway/types"

	appaws "github.com/clawscli/claws/internal/aws"
	"github.com/clawscli/claws/internal/dao"
)

// StageDAO provides data access for API Gateway REST API stages
type StageDAO struct {
	dao.BaseDAO
	client *apigateway.Client
}

// NewStageDAO creates a new StageDAO
func NewStageDAO(ctx context.Context) (dao.DAO, error) {
	cfg, err := appaws.NewConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("new apigateway/stages dao: %w", err)
	}
	return &StageDAO{
		BaseDAO: dao.NewBaseDAO("apigateway", "stages"),
		client:  apigateway.NewFromConfig(cfg),
	}, nil
}

// List returns all stages for a REST API (requires RestApiId filter)
func (d *StageDAO) List(ctx context.Context) ([]dao.Resource, error) {
	restApiId := dao.GetFilterFromContext(ctx, "RestApiId")
	if restApiId == "" {
		return nil, fmt.Errorf("RestApiId filter required - navigate from a REST API")
	}

	output, err := d.client.GetStages(ctx, &apigateway.GetStagesInput{
		RestApiId: &restApiId,
	})
	if err != nil {
		return nil, fmt.Errorf("list stages: %w", err)
	}

	var resources []dao.Resource
	for _, stage := range output.Item {
		resources = append(resources, NewStageResource(stage, restApiId))
	}

	return resources, nil
}

// Get returns a specific stage
func (d *StageDAO) Get(ctx context.Context, id string) (dao.Resource, error) {
	// ID format: restApiId:stageName
	restApiId, stageName, err := parseStageid(id)
	if err != nil {
		return nil, err
	}

	output, err := d.client.GetStage(ctx, &apigateway.GetStageInput{
		RestApiId: &restApiId,
		StageName: &stageName,
	})
	if err != nil {
		return nil, fmt.Errorf("get stage %s: %w", id, err)
	}

	return NewStageResourceFromGetOutput(output, restApiId), nil
}

// Delete deletes a stage
func (d *StageDAO) Delete(ctx context.Context, id string) error {
	restApiId, stageName, err := parseStageid(id)
	if err != nil {
		return err
	}

	_, err = d.client.DeleteStage(ctx, &apigateway.DeleteStageInput{
		RestApiId: &restApiId,
		StageName: &stageName,
	})
	if err != nil {
		return fmt.Errorf("delete stage %s: %w", id, err)
	}
	return nil
}

func parseStageid(id string) (restApiId, stageName string, err error) {
	for i := len(id) - 1; i >= 0; i-- {
		if id[i] == ':' {
			return id[:i], id[i+1:], nil
		}
	}
	return "", "", fmt.Errorf("invalid stage ID format: %s (expected restApiId:stageName)", id)
}

// StageResource wraps an API Gateway stage
type StageResource struct {
	dao.BaseResource
	Item      types.Stage
	RestApiId string
}

// NewStageResource creates a new StageResource
func NewStageResource(stage types.Stage, restApiId string) *StageResource {
	stageName := appaws.Str(stage.StageName)

	// ID format: restApiId:stageName
	id := fmt.Sprintf("%s:%s", restApiId, stageName)

	// Convert tags
	tags := make(map[string]string)
	for k, v := range stage.Tags {
		tags[k] = v
	}

	return &StageResource{
		BaseResource: dao.BaseResource{
			ID:   id,
			Name: stageName,
			Tags: tags,
			Data: stage,
		},
		Item:      stage,
		RestApiId: restApiId,
	}
}

// NewStageResourceFromGetOutput creates a StageResource from GetStage output
func NewStageResourceFromGetOutput(output *apigateway.GetStageOutput, restApiId string) *StageResource {
	stage := types.Stage{
		StageName:            output.StageName,
		DeploymentId:         output.DeploymentId,
		Description:          output.Description,
		CacheClusterEnabled:  output.CacheClusterEnabled,
		CacheClusterSize:     output.CacheClusterSize,
		CacheClusterStatus:   output.CacheClusterStatus,
		ClientCertificateId:  output.ClientCertificateId,
		CreatedDate:          output.CreatedDate,
		LastUpdatedDate:      output.LastUpdatedDate,
		TracingEnabled:       output.TracingEnabled,
		WebAclArn:            output.WebAclArn,
		Tags:                 output.Tags,
		Variables:            output.Variables,
		AccessLogSettings:    output.AccessLogSettings,
		CanarySettings:       output.CanarySettings,
		DocumentationVersion: output.DocumentationVersion,
		MethodSettings:       output.MethodSettings,
	}
	return NewStageResource(stage, restApiId)
}

// StageName returns the stage name
func (r *StageResource) StageName() string {
	if r.Item.StageName != nil {
		return *r.Item.StageName
	}
	return ""
}

// DeploymentId returns the deployment ID
func (r *StageResource) DeploymentId() string {
	if r.Item.DeploymentId != nil {
		return *r.Item.DeploymentId
	}
	return ""
}

// Description returns the stage description
func (r *StageResource) Description() string {
	if r.Item.Description != nil {
		return *r.Item.Description
	}
	return ""
}

// CacheClusterEnabled returns whether cache is enabled
func (r *StageResource) CacheClusterEnabled() bool {
	return r.Item.CacheClusterEnabled
}

// CacheClusterSize returns the cache cluster size
func (r *StageResource) CacheClusterSize() string {
	return string(r.Item.CacheClusterSize)
}

// CacheClusterStatus returns the cache cluster status
func (r *StageResource) CacheClusterStatus() string {
	return string(r.Item.CacheClusterStatus)
}

// TracingEnabled returns whether X-Ray tracing is enabled
func (r *StageResource) TracingEnabled() bool {
	return r.Item.TracingEnabled
}

// CreatedDate returns the creation date
func (r *StageResource) CreatedDate() time.Time {
	if r.Item.CreatedDate != nil {
		return *r.Item.CreatedDate
	}
	return time.Time{}
}

// LastUpdatedDate returns the last updated date
func (r *StageResource) LastUpdatedDate() time.Time {
	if r.Item.LastUpdatedDate != nil {
		return *r.Item.LastUpdatedDate
	}
	return time.Time{}
}

// WebAclArn returns the WAF web ACL ARN
func (r *StageResource) WebAclArn() string {
	if r.Item.WebAclArn != nil {
		return *r.Item.WebAclArn
	}
	return ""
}

// Variables returns the stage variables
func (r *StageResource) Variables() map[string]string {
	return r.Item.Variables
}

// HasAccessLogs returns whether access logging is configured
func (r *StageResource) HasAccessLogs() bool {
	return r.Item.AccessLogSettings != nil && r.Item.AccessLogSettings.DestinationArn != nil
}

// AccessLogDestination returns the access log destination ARN
func (r *StageResource) AccessLogDestination() string {
	if r.Item.AccessLogSettings != nil && r.Item.AccessLogSettings.DestinationArn != nil {
		return *r.Item.AccessLogSettings.DestinationArn
	}
	return ""
}
