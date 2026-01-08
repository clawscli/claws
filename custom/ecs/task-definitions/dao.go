package taskdefinitions

import (
	"context"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/ecs"
	"github.com/aws/aws-sdk-go-v2/service/ecs/types"

	appaws "github.com/clawscli/claws/internal/aws"
	"github.com/clawscli/claws/internal/dao"
	apperrors "github.com/clawscli/claws/internal/errors"
	"github.com/clawscli/claws/internal/log"
)

type TaskDefinitionDAO struct {
	dao.BaseDAO
	client *ecs.Client
}

func NewTaskDefinitionDAO(ctx context.Context) (dao.DAO, error) {
	cfg, err := appaws.NewConfig(ctx)
	if err != nil {
		return nil, apperrors.Wrap(err, "new ecs/task-definitions dao")
	}
	return &TaskDefinitionDAO{
		BaseDAO: dao.NewBaseDAO("ecs", "task-definitions"),
		client:  ecs.NewFromConfig(cfg),
	}, nil
}

func (d *TaskDefinitionDAO) List(ctx context.Context) ([]dao.Resource, error) {
	taskDefArns, err := appaws.Paginate(ctx, func(token *string) ([]string, *string, error) {
		output, err := d.client.ListTaskDefinitions(ctx, &ecs.ListTaskDefinitionsInput{
			Status:    types.TaskDefinitionStatusActive,
			Sort:      types.SortOrderDesc,
			NextToken: token,
		})
		if err != nil {
			return nil, nil, apperrors.Wrap(err, "list task definitions")
		}
		return output.TaskDefinitionArns, output.NextToken, nil
	})
	if err != nil {
		return nil, err
	}

	seenFamilies := make(map[string]bool)
	var latestArns []string
	for _, arn := range taskDefArns {
		family := extractFamilyFromArn(arn)
		if !seenFamilies[family] {
			seenFamilies[family] = true
			latestArns = append(latestArns, arn)
		}
	}

	resources := make([]dao.Resource, 0, len(latestArns))
	for _, arn := range latestArns {
		output, err := d.client.DescribeTaskDefinition(ctx, &ecs.DescribeTaskDefinitionInput{
			TaskDefinition: &arn,
		})
		if err != nil {
			log.Warn("failed to describe task definition", "arn", arn, "error", err)
			continue
		}
		if output.TaskDefinition != nil {
			resources = append(resources, NewTaskDefinitionResource(*output.TaskDefinition))
		}
	}

	return resources, nil
}

func (d *TaskDefinitionDAO) Get(ctx context.Context, id string) (dao.Resource, error) {
	output, err := d.client.DescribeTaskDefinition(ctx, &ecs.DescribeTaskDefinitionInput{
		TaskDefinition: &id,
	})
	if err != nil {
		return nil, apperrors.Wrapf(err, "describe task definition %s", id)
	}

	if output.TaskDefinition == nil {
		return nil, fmt.Errorf("task definition not found: %s", id)
	}

	return NewTaskDefinitionResource(*output.TaskDefinition), nil
}

func (d *TaskDefinitionDAO) Delete(ctx context.Context, id string) error {
	_, err := d.client.DeregisterTaskDefinition(ctx, &ecs.DeregisterTaskDefinitionInput{
		TaskDefinition: &id,
	})
	if err != nil {
		return apperrors.Wrapf(err, "deregister task definition %s", id)
	}
	return nil
}

func extractFamilyFromArn(arn string) string {
	parts := strings.Split(arn, "/")
	if len(parts) < 2 {
		return arn
	}
	familyRevision := parts[len(parts)-1]
	colonIdx := strings.LastIndex(familyRevision, ":")
	if colonIdx == -1 {
		return familyRevision
	}
	return familyRevision[:colonIdx]
}

type TaskDefinitionResource struct {
	dao.BaseResource
	Item types.TaskDefinition
}

func NewTaskDefinitionResource(td types.TaskDefinition) *TaskDefinitionResource {
	family := appaws.Str(td.Family)
	revision := td.Revision
	id := fmt.Sprintf("%s:%d", family, revision)

	return &TaskDefinitionResource{
		BaseResource: dao.BaseResource{
			ID:   id,
			Name: family,
			ARN:  appaws.Str(td.TaskDefinitionArn),
			Data: td,
		},
		Item: td,
	}
}

func (r *TaskDefinitionResource) Family() string {
	return appaws.Str(r.Item.Family)
}

func (r *TaskDefinitionResource) Revision() int32 {
	return r.Item.Revision
}

func (r *TaskDefinitionResource) Status() string {
	return string(r.Item.Status)
}

func (r *TaskDefinitionResource) CPU() string {
	return appaws.Str(r.Item.Cpu)
}

func (r *TaskDefinitionResource) Memory() string {
	return appaws.Str(r.Item.Memory)
}

func (r *TaskDefinitionResource) NetworkMode() string {
	return string(r.Item.NetworkMode)
}

func (r *TaskDefinitionResource) RequiresCompatibilities() []types.Compatibility {
	return r.Item.RequiresCompatibilities
}

func (r *TaskDefinitionResource) ContainerDefinitions() []types.ContainerDefinition {
	return r.Item.ContainerDefinitions
}

func (r *TaskDefinitionResource) TaskRoleArn() string {
	return appaws.Str(r.Item.TaskRoleArn)
}

func (r *TaskDefinitionResource) ExecutionRoleArn() string {
	return appaws.Str(r.Item.ExecutionRoleArn)
}

func (r *TaskDefinitionResource) Volumes() []types.Volume {
	return r.Item.Volumes
}

func (r *TaskDefinitionResource) RuntimePlatform() *types.RuntimePlatform {
	return r.Item.RuntimePlatform
}

func (r *TaskDefinitionResource) GetLogConfiguration(containerName string) *types.LogConfiguration {
	containers := r.Item.ContainerDefinitions
	if len(containers) == 0 {
		return nil
	}

	if containerName != "" {
		for _, c := range containers {
			if appaws.Str(c.Name) == containerName {
				return c.LogConfiguration
			}
		}
		return nil
	}

	return containers[0].LogConfiguration
}

func (r *TaskDefinitionResource) GetCloudWatchLogGroup(containerName string) string {
	logConfig := r.GetLogConfiguration(containerName)
	if logConfig == nil {
		return ""
	}

	if logConfig.LogDriver != types.LogDriverAwslogs {
		return ""
	}

	if logConfig.Options == nil {
		return ""
	}

	return logConfig.Options["awslogs-group"]
}

func (r *TaskDefinitionResource) GetAllCloudWatchLogGroups() []string {
	var groups []string
	seen := make(map[string]bool)

	for _, c := range r.Item.ContainerDefinitions {
		if c.LogConfiguration == nil {
			continue
		}
		if c.LogConfiguration.LogDriver != types.LogDriverAwslogs {
			continue
		}
		if c.LogConfiguration.Options == nil {
			continue
		}
		if group := c.LogConfiguration.Options["awslogs-group"]; group != "" && !seen[group] {
			seen[group] = true
			groups = append(groups, group)
		}
	}

	return groups
}
