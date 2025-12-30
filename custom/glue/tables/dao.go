package tables

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/glue"
	"github.com/aws/aws-sdk-go-v2/service/glue/types"

	appaws "github.com/clawscli/claws/internal/aws"
	"github.com/clawscli/claws/internal/dao"
)

// TableDAO provides data access for Glue tables.
type TableDAO struct {
	dao.BaseDAO
	client *glue.Client
}

// NewTableDAO creates a new TableDAO.
func NewTableDAO(ctx context.Context) (dao.DAO, error) {
	cfg, err := appaws.NewConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("new glue/tables dao: %w", err)
	}
	return &TableDAO{
		BaseDAO: dao.NewBaseDAO("glue", "tables"),
		client:  glue.NewFromConfig(cfg),
	}, nil
}

// List returns all Glue tables for a database.
func (d *TableDAO) List(ctx context.Context) ([]dao.Resource, error) {
	// Get database name from filter
	databaseName := dao.GetFilterFromContext(ctx, "DatabaseName")
	if databaseName == "" {
		return nil, fmt.Errorf("database name filter required")
	}

	tables, err := appaws.Paginate(ctx, func(token *string) ([]types.Table, *string, error) {
		output, err := d.client.GetTables(ctx, &glue.GetTablesInput{
			DatabaseName: &databaseName,
			NextToken:    token,
		})
		if err != nil {
			return nil, nil, fmt.Errorf("get glue tables: %w", err)
		}
		return output.TableList, output.NextToken, nil
	})
	if err != nil {
		return nil, err
	}

	resources := make([]dao.Resource, len(tables))
	for i, table := range tables {
		resources[i] = NewTableResource(table, databaseName)
	}
	return resources, nil
}

// Get returns a specific Glue table by name.
func (d *TableDAO) Get(ctx context.Context, id string) (dao.Resource, error) {
	databaseName := dao.GetFilterFromContext(ctx, "DatabaseName")
	if databaseName == "" {
		return nil, fmt.Errorf("database name filter required")
	}

	output, err := d.client.GetTable(ctx, &glue.GetTableInput{
		DatabaseName: &databaseName,
		Name:         &id,
	})
	if err != nil {
		return nil, fmt.Errorf("get glue table %s: %w", id, err)
	}
	return NewTableResource(*output.Table, databaseName), nil
}

// Delete deletes a Glue table by name.
func (d *TableDAO) Delete(ctx context.Context, id string) error {
	databaseName := dao.GetFilterFromContext(ctx, "DatabaseName")
	if databaseName == "" {
		return fmt.Errorf("database name filter required")
	}

	_, err := d.client.DeleteTable(ctx, &glue.DeleteTableInput{
		DatabaseName: &databaseName,
		Name:         &id,
	})
	if err != nil {
		return fmt.Errorf("delete glue table %s: %w", id, err)
	}
	return nil
}

// TableResource wraps a Glue table.
type TableResource struct {
	dao.BaseResource
	Item         types.Table
	DatabaseName string
}

// NewTableResource creates a new TableResource.
func NewTableResource(table types.Table, databaseName string) *TableResource {
	return &TableResource{
		BaseResource: dao.BaseResource{
			ID:  appaws.Str(table.Name),
			ARN: "",
		},
		Item:         table,
		DatabaseName: databaseName,
	}
}

// Name returns the table name.
func (r *TableResource) Name() string {
	return appaws.Str(r.Item.Name)
}

// TableType returns the table type.
func (r *TableResource) TableType() string {
	return appaws.Str(r.Item.TableType)
}

// Description returns the table description.
func (r *TableResource) Description() string {
	return appaws.Str(r.Item.Description)
}

// Location returns the table location.
func (r *TableResource) Location() string {
	if r.Item.StorageDescriptor != nil {
		return appaws.Str(r.Item.StorageDescriptor.Location)
	}
	return ""
}

// InputFormat returns the input format.
func (r *TableResource) InputFormat() string {
	if r.Item.StorageDescriptor != nil {
		return appaws.Str(r.Item.StorageDescriptor.InputFormat)
	}
	return ""
}

// OutputFormat returns the output format.
func (r *TableResource) OutputFormat() string {
	if r.Item.StorageDescriptor != nil {
		return appaws.Str(r.Item.StorageDescriptor.OutputFormat)
	}
	return ""
}

// ColumnCount returns the number of columns.
func (r *TableResource) ColumnCount() int {
	if r.Item.StorageDescriptor != nil {
		return len(r.Item.StorageDescriptor.Columns)
	}
	return 0
}

// CreateTime returns when the table was created.
func (r *TableResource) CreateTime() *time.Time {
	return r.Item.CreateTime
}

// UpdateTime returns when the table was last updated.
func (r *TableResource) UpdateTime() *time.Time {
	return r.Item.UpdateTime
}
