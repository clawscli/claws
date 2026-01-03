package tables

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"

	appaws "github.com/clawscli/claws/internal/aws"
	"github.com/clawscli/claws/internal/dao"
	apperrors "github.com/clawscli/claws/internal/errors"
	"github.com/clawscli/claws/internal/log"
)

// TableDAO provides data access for DynamoDB tables
type TableDAO struct {
	dao.BaseDAO
	client *dynamodb.Client
}

// NewTableDAO creates a new TableDAO
func NewTableDAO(ctx context.Context) (dao.DAO, error) {
	cfg, err := appaws.NewConfig(ctx)
	if err != nil {
		return nil, apperrors.Wrap(err, "new "+ServiceResourcePath+" dao")
	}
	return &TableDAO{
		BaseDAO: dao.NewBaseDAO("dynamodb", "tables"),
		client:  dynamodb.NewFromConfig(cfg),
	}, nil
}

func (d *TableDAO) List(ctx context.Context) ([]dao.Resource, error) {
	tableNames, err := appaws.Paginate(ctx, func(token *string) ([]string, *string, error) {
		output, err := d.client.ListTables(ctx, &dynamodb.ListTablesInput{
			ExclusiveStartTableName: token,
		})
		if err != nil {
			return nil, nil, apperrors.Wrap(err, "list tables")
		}
		return output.TableNames, output.LastEvaluatedTableName, nil
	})
	if err != nil {
		return nil, err
	}

	resources := make([]dao.Resource, 0, len(tableNames))
	for _, tableName := range tableNames {
		descOutput, err := d.client.DescribeTable(ctx, &dynamodb.DescribeTableInput{
			TableName: &tableName,
		})
		if err != nil {
			log.Warn("failed to describe table", "table", tableName, "error", err)
			continue
		}
		if descOutput.Table != nil {
			resources = append(resources, NewTableResource(*descOutput.Table))
		}
	}

	return resources, nil
}

func (d *TableDAO) Get(ctx context.Context, id string) (dao.Resource, error) {
	input := &dynamodb.DescribeTableInput{
		TableName: &id,
	}

	output, err := d.client.DescribeTable(ctx, input)
	if err != nil {
		return nil, apperrors.Wrapf(err, "describe table %s", id)
	}

	if output.Table == nil {
		return nil, fmt.Errorf("table not found: %s", id)
	}

	return NewTableResource(*output.Table), nil
}

func (d *TableDAO) Delete(ctx context.Context, id string) error {
	input := &dynamodb.DeleteTableInput{
		TableName: &id,
	}

	_, err := d.client.DeleteTable(ctx, input)
	if err != nil {
		if apperrors.IsNotFound(err) {
			return nil // Already deleted
		}
		if apperrors.IsResourceInUse(err) {
			return apperrors.Wrapf(err, "table %s is in use", id)
		}
		return apperrors.Wrapf(err, "delete table %s", id)
	}

	return nil
}

// TableResource wraps a DynamoDB table
type TableResource struct {
	dao.BaseResource
	Item types.TableDescription
}

// NewTableResource creates a new TableResource
func NewTableResource(table types.TableDescription) *TableResource {
	return &TableResource{
		BaseResource: dao.BaseResource{
			ID:   appaws.Str(table.TableName),
			Name: appaws.Str(table.TableName),
			ARN:  appaws.Str(table.TableArn),
			Data: table,
		},
		Item: table,
	}
}

// Status returns the table status
func (r *TableResource) Status() string {
	return string(r.Item.TableStatus)
}

// ItemCount returns the item count
func (r *TableResource) ItemCount() int64 {
	if r.Item.ItemCount != nil {
		return *r.Item.ItemCount
	}
	return 0
}

// SizeBytes returns the table size in bytes
func (r *TableResource) SizeBytes() int64 {
	if r.Item.TableSizeBytes != nil {
		return *r.Item.TableSizeBytes
	}
	return 0
}

// BillingMode returns the billing mode
func (r *TableResource) BillingMode() string {
	if r.Item.BillingModeSummary != nil {
		return string(r.Item.BillingModeSummary.BillingMode)
	}
	return "PROVISIONED"
}

// ReadCapacity returns the read capacity units
func (r *TableResource) ReadCapacity() int64 {
	if r.Item.ProvisionedThroughput != nil && r.Item.ProvisionedThroughput.ReadCapacityUnits != nil {
		return *r.Item.ProvisionedThroughput.ReadCapacityUnits
	}
	return 0
}

// WriteCapacity returns the write capacity units
func (r *TableResource) WriteCapacity() int64 {
	if r.Item.ProvisionedThroughput != nil && r.Item.ProvisionedThroughput.WriteCapacityUnits != nil {
		return *r.Item.ProvisionedThroughput.WriteCapacityUnits
	}
	return 0
}

// GSICount returns the number of Global Secondary Indexes
func (r *TableResource) GSICount() int {
	return len(r.Item.GlobalSecondaryIndexes)
}

// LSICount returns the number of Local Secondary Indexes
func (r *TableResource) LSICount() int {
	return len(r.Item.LocalSecondaryIndexes)
}

// KeySchema returns the key schema
func (r *TableResource) KeySchema() []types.KeySchemaElement {
	return r.Item.KeySchema
}

// GlobalSecondaryIndexes returns the GSIs
func (r *TableResource) GlobalSecondaryIndexes() []types.GlobalSecondaryIndexDescription {
	return r.Item.GlobalSecondaryIndexes
}

// CreationDateTime returns the creation time
func (r *TableResource) CreationDateTime() string {
	if r.Item.CreationDateTime != nil {
		return r.Item.CreationDateTime.Format("2006-01-02 15:04:05")
	}
	return ""
}

// TableClass returns the table class
func (r *TableResource) TableClass() string {
	if r.Item.TableClassSummary != nil {
		return string(r.Item.TableClassSummary.TableClass)
	}
	return "STANDARD"
}

// DeletionProtectionEnabled returns whether deletion protection is enabled
func (r *TableResource) DeletionProtectionEnabled() bool {
	return r.Item.DeletionProtectionEnabled != nil && *r.Item.DeletionProtectionEnabled
}

// SSEDescription returns the server-side encryption description
func (r *TableResource) SSEDescription() *types.SSEDescription {
	return r.Item.SSEDescription
}

// Replicas returns the global table replicas
func (r *TableResource) Replicas() []types.ReplicaDescription {
	return r.Item.Replicas
}

// RestoreSummary returns the restore summary if restored from backup
func (r *TableResource) RestoreSummary() *types.RestoreSummary {
	return r.Item.RestoreSummary
}

// TableId returns the unique table ID
func (r *TableResource) TableId() string {
	return appaws.Str(r.Item.TableId)
}

// StreamArn returns the DynamoDB Streams ARN
func (r *TableResource) StreamArn() string {
	return appaws.Str(r.Item.LatestStreamArn)
}
