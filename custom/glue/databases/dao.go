package databases

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/glue"
	"github.com/aws/aws-sdk-go-v2/service/glue/types"

	appaws "github.com/clawscli/claws/internal/aws"
	"github.com/clawscli/claws/internal/dao"
)

// DatabaseDAO provides data access for Glue databases.
type DatabaseDAO struct {
	dao.BaseDAO
	client *glue.Client
}

// NewDatabaseDAO creates a new DatabaseDAO.
func NewDatabaseDAO(ctx context.Context) (dao.DAO, error) {
	cfg, err := appaws.NewConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("new glue/databases dao: %w", err)
	}
	return &DatabaseDAO{
		BaseDAO: dao.NewBaseDAO("glue", "databases"),
		client:  glue.NewFromConfig(cfg),
	}, nil
}

// List returns all Glue databases.
func (d *DatabaseDAO) List(ctx context.Context) ([]dao.Resource, error) {
	databases, err := appaws.Paginate(ctx, func(token *string) ([]types.Database, *string, error) {
		output, err := d.client.GetDatabases(ctx, &glue.GetDatabasesInput{
			NextToken: token,
		})
		if err != nil {
			return nil, nil, fmt.Errorf("get glue databases: %w", err)
		}
		return output.DatabaseList, output.NextToken, nil
	})
	if err != nil {
		return nil, err
	}

	resources := make([]dao.Resource, len(databases))
	for i, db := range databases {
		resources[i] = NewDatabaseResource(db)
	}
	return resources, nil
}

// Get returns a specific Glue database by name.
func (d *DatabaseDAO) Get(ctx context.Context, id string) (dao.Resource, error) {
	output, err := d.client.GetDatabase(ctx, &glue.GetDatabaseInput{
		Name: &id,
	})
	if err != nil {
		return nil, fmt.Errorf("get glue database %s: %w", id, err)
	}
	return NewDatabaseResource(*output.Database), nil
}

// Delete deletes a Glue database by name.
func (d *DatabaseDAO) Delete(ctx context.Context, id string) error {
	_, err := d.client.DeleteDatabase(ctx, &glue.DeleteDatabaseInput{
		Name: &id,
	})
	if err != nil {
		return fmt.Errorf("delete glue database %s: %w", id, err)
	}
	return nil
}

// DatabaseResource wraps a Glue database.
type DatabaseResource struct {
	dao.BaseResource
	Item types.Database
}

// NewDatabaseResource creates a new DatabaseResource.
func NewDatabaseResource(db types.Database) *DatabaseResource {
	return &DatabaseResource{
		BaseResource: dao.BaseResource{
			ID:  appaws.Str(db.Name),
			ARN: appaws.Str(db.CatalogId),
		},
		Item: db,
	}
}

// Name returns the database name.
func (r *DatabaseResource) Name() string {
	return appaws.Str(r.Item.Name)
}

// Description returns the database description.
func (r *DatabaseResource) Description() string {
	return appaws.Str(r.Item.Description)
}

// CatalogId returns the catalog ID.
func (r *DatabaseResource) CatalogId() string {
	return appaws.Str(r.Item.CatalogId)
}

// LocationUri returns the location URI.
func (r *DatabaseResource) LocationUri() string {
	return appaws.Str(r.Item.LocationUri)
}

// CreateTime returns when the database was created.
func (r *DatabaseResource) CreateTime() *time.Time {
	return r.Item.CreateTime
}
