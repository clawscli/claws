package parameters

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/aws/aws-sdk-go-v2/service/ssm/types"

	appaws "github.com/clawscli/claws/internal/aws"
	"github.com/clawscli/claws/internal/dao"
)

// ParameterDAO provides data access for SSM Parameter Store
type ParameterDAO struct {
	dao.BaseDAO
	client *ssm.Client
}

// NewParameterDAO creates a new ParameterDAO
func NewParameterDAO(ctx context.Context) (dao.DAO, error) {
	cfg, err := appaws.NewConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("new ssm/parameters dao: %w", err)
	}
	return &ParameterDAO{
		BaseDAO: dao.NewBaseDAO("ssm", "parameters"),
		client:  ssm.NewFromConfig(cfg),
	}, nil
}

// List returns parameters (first page only for backwards compatibility).
// For paginated access, use ListPage instead.
func (d *ParameterDAO) List(ctx context.Context) ([]dao.Resource, error) {
	resources, _, err := d.ListPage(ctx, 50, "")
	return resources, err
}

// ListPage returns a page of SSM parameters.
// Implements dao.PaginatedDAO interface.
func (d *ParameterDAO) ListPage(ctx context.Context, pageSize int, pageToken string) ([]dao.Resource, string, error) {
	maxResults := int32(pageSize)
	if maxResults > 50 {
		maxResults = 50 // AWS API max
	}

	input := &ssm.DescribeParametersInput{
		MaxResults: &maxResults,
	}
	if pageToken != "" {
		input.NextToken = &pageToken
	}

	output, err := d.client.DescribeParameters(ctx, input)
	if err != nil {
		return nil, "", fmt.Errorf("describe parameters: %w", err)
	}

	resources := make([]dao.Resource, len(output.Parameters))
	for i, param := range output.Parameters {
		resources[i] = NewParameterResource(param)
	}

	nextToken := ""
	if output.NextToken != nil {
		nextToken = *output.NextToken
	}

	return resources, nextToken, nil
}

func (d *ParameterDAO) Get(ctx context.Context, id string) (dao.Resource, error) {
	// First get metadata
	descInput := &ssm.DescribeParametersInput{
		ParameterFilters: []types.ParameterStringFilter{
			{
				Key:    appaws.StringPtr("Name"),
				Option: appaws.StringPtr("Equals"),
				Values: []string{id},
			},
		},
	}

	descOutput, err := d.client.DescribeParameters(ctx, descInput)
	if err != nil {
		return nil, fmt.Errorf("describe parameter %s: %w", id, err)
	}

	if len(descOutput.Parameters) == 0 {
		return nil, fmt.Errorf("parameter not found: %s", id)
	}

	return NewParameterResource(descOutput.Parameters[0]), nil
}

func (d *ParameterDAO) Delete(ctx context.Context, id string) error {
	input := &ssm.DeleteParameterInput{
		Name: &id,
	}

	_, err := d.client.DeleteParameter(ctx, input)
	if err != nil {
		return fmt.Errorf("delete parameter %s: %w", id, err)
	}

	return nil
}

// ParameterResource wraps an SSM Parameter
type ParameterResource struct {
	dao.BaseResource
	Item types.ParameterMetadata
}

// NewParameterResource creates a new ParameterResource
func NewParameterResource(param types.ParameterMetadata) *ParameterResource {
	name := appaws.Str(param.Name)

	return &ParameterResource{
		BaseResource: dao.BaseResource{
			ID:   name,
			Name: name,
			ARN:  appaws.Str(param.ARN),
			Tags: nil, // Tags require separate API call
			Data: param,
		},
		Item: param,
	}
}

// Type returns the parameter type (String, StringList, SecureString)
func (r *ParameterResource) Type() string {
	return string(r.Item.Type)
}

// Tier returns the parameter tier (Standard, Advanced, Intelligent-Tiering)
func (r *ParameterResource) Tier() string {
	return string(r.Item.Tier)
}

// Description returns the parameter description
func (r *ParameterResource) Description() string {
	return appaws.Str(r.Item.Description)
}

// Version returns the parameter version
func (r *ParameterResource) Version() int64 {
	return r.Item.Version
}

// LastModifiedDate returns the last modified date as string
func (r *ParameterResource) LastModifiedDate() string {
	if r.Item.LastModifiedDate != nil {
		return r.Item.LastModifiedDate.Format("2006-01-02 15:04:05")
	}
	return ""
}

// LastModifiedUser returns the user who last modified the parameter
func (r *ParameterResource) LastModifiedUser() string {
	return appaws.Str(r.Item.LastModifiedUser)
}

// DataType returns the data type
func (r *ParameterResource) DataType() string {
	return appaws.Str(r.Item.DataType)
}
