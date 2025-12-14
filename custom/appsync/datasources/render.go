package datasources

import (
	"fmt"

	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/render"
)

// DataSourceRenderer renders AppSync data sources.
type DataSourceRenderer struct {
	render.BaseRenderer
}

// NewDataSourceRenderer creates a new DataSourceRenderer.
func NewDataSourceRenderer() render.Renderer {
	return &DataSourceRenderer{
		BaseRenderer: render.BaseRenderer{
			Service:  "appsync",
			Resource: "data-sources",
			Cols: []render.Column{
				{Name: "NAME", Width: 35, Getter: func(r dao.Resource) string { return r.GetID() }},
				{Name: "TYPE", Width: 20, Getter: getType},
				{Name: "DESCRIPTION", Width: 40, Getter: getDescription},
			},
		},
	}
}

func getType(r dao.Resource) string {
	ds, ok := r.(*DataSourceResource)
	if !ok {
		return ""
	}
	return ds.Type()
}

func getDescription(r dao.Resource) string {
	ds, ok := r.(*DataSourceResource)
	if !ok {
		return ""
	}
	return ds.Description()
}

// RenderDetail renders the detail view for a data source.
func (r *DataSourceRenderer) RenderDetail(resource dao.Resource) string {
	ds, ok := resource.(*DataSourceResource)
	if !ok {
		return ""
	}

	d := render.NewDetailBuilder()
	s := ds.DataSource

	d.Title("AppSync Data Source", ds.Name())

	// Basic Info
	d.Section("Basic Information")
	d.Field("Name", ds.Name())
	d.Field("ARN", ds.GetARN())
	d.Field("Type", ds.Type())
	if ds.Description() != "" {
		d.Field("Description", ds.Description())
	}
	d.Field("Metrics", string(s.MetricsConfig))

	// Type-specific configuration
	switch ds.Type() {
	case "AMAZON_DYNAMODB":
		if s.DynamodbConfig != nil {
			d.Section("DynamoDB Configuration")
			if s.DynamodbConfig.TableName != nil {
				d.Field("Table Name", *s.DynamodbConfig.TableName)
			}
			if s.DynamodbConfig.AwsRegion != nil {
				d.Field("Region", *s.DynamodbConfig.AwsRegion)
			}
			d.Field("Versioned", fmt.Sprintf("%v", s.DynamodbConfig.Versioned))
			d.Field("Use Caller Credentials", fmt.Sprintf("%v", s.DynamodbConfig.UseCallerCredentials))
		}
	case "AWS_LAMBDA":
		if s.LambdaConfig != nil {
			d.Section("Lambda Configuration")
			if s.LambdaConfig.LambdaFunctionArn != nil {
				d.Field("Function ARN", *s.LambdaConfig.LambdaFunctionArn)
			}
		}
	case "HTTP":
		if s.HttpConfig != nil {
			d.Section("HTTP Configuration")
			if s.HttpConfig.Endpoint != nil {
				d.Field("Endpoint", *s.HttpConfig.Endpoint)
			}
			if s.HttpConfig.AuthorizationConfig != nil {
				d.Field("Auth Type", string(s.HttpConfig.AuthorizationConfig.AuthorizationType))
			}
		}
	case "AMAZON_OPENSEARCH_SERVICE":
		if s.OpenSearchServiceConfig != nil {
			d.Section("OpenSearch Configuration")
			if s.OpenSearchServiceConfig.Endpoint != nil {
				d.Field("Endpoint", *s.OpenSearchServiceConfig.Endpoint)
			}
			if s.OpenSearchServiceConfig.AwsRegion != nil {
				d.Field("Region", *s.OpenSearchServiceConfig.AwsRegion)
			}
		}
	case "RELATIONAL_DATABASE":
		if s.RelationalDatabaseConfig != nil {
			d.Section("RDS Configuration")
			d.Field("Source Type", string(s.RelationalDatabaseConfig.RelationalDatabaseSourceType))
			if s.RelationalDatabaseConfig.RdsHttpEndpointConfig != nil {
				rds := s.RelationalDatabaseConfig.RdsHttpEndpointConfig
				if rds.DbClusterIdentifier != nil {
					d.Field("Cluster", *rds.DbClusterIdentifier)
				}
				if rds.DatabaseName != nil {
					d.Field("Database", *rds.DatabaseName)
				}
				if rds.AwsRegion != nil {
					d.Field("Region", *rds.AwsRegion)
				}
				if rds.AwsSecretStoreArn != nil {
					d.Field("Secret ARN", *rds.AwsSecretStoreArn)
				}
			}
		}
	case "AMAZON_EVENTBRIDGE":
		if s.EventBridgeConfig != nil {
			d.Section("EventBridge Configuration")
			if s.EventBridgeConfig.EventBusArn != nil {
				d.Field("Event Bus ARN", *s.EventBridgeConfig.EventBusArn)
			}
		}
	}

	// IAM
	if ds.ServiceRoleArn() != "" {
		d.Section("IAM")
		d.Field("Service Role", ds.ServiceRoleArn())
	}

	return d.String()
}

// RenderSummary renders summary fields for a data source.
func (r *DataSourceRenderer) RenderSummary(resource dao.Resource) []render.SummaryField {
	ds, ok := resource.(*DataSourceResource)
	if !ok {
		return r.BaseRenderer.RenderSummary(resource)
	}

	return []render.SummaryField{
		{Label: "Name", Value: ds.Name()},
		{Label: "Type", Value: ds.Type()},
	}
}
