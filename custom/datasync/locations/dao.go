package locations

import (
	"context"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/datasync"
	"github.com/aws/aws-sdk-go-v2/service/datasync/types"

	appaws "github.com/clawscli/claws/internal/aws"
	"github.com/clawscli/claws/internal/dao"
	apperrors "github.com/clawscli/claws/internal/errors"
)

// LocationDAO provides data access for DataSync locations.
type LocationDAO struct {
	dao.BaseDAO
	client *datasync.Client
}

// NewLocationDAO creates a new LocationDAO.
func NewLocationDAO(ctx context.Context) (dao.DAO, error) {
	cfg, err := appaws.NewConfig(ctx)
	if err != nil {
		return nil, apperrors.Wrap(err, "new "+ServiceResourcePath+" dao")
	}
	return &LocationDAO{
		BaseDAO: dao.NewBaseDAO("datasync", "locations"),
		client:  datasync.NewFromConfig(cfg),
	}, nil
}

// List returns all DataSync locations.
func (d *LocationDAO) List(ctx context.Context) ([]dao.Resource, error) {
	locations, err := appaws.Paginate(ctx, func(token *string) ([]types.LocationListEntry, *string, error) {
		output, err := d.client.ListLocations(ctx, &datasync.ListLocationsInput{
			NextToken: token,
		})
		if err != nil {
			return nil, nil, apperrors.Wrap(err, "list datasync locations")
		}
		return output.Locations, output.NextToken, nil
	})
	if err != nil {
		return nil, err
	}

	resources := make([]dao.Resource, len(locations))
	for i, loc := range locations {
		resources[i] = NewLocationResource(loc)
	}
	return resources, nil
}

// Get returns a specific location.
func (d *LocationDAO) Get(ctx context.Context, id string) (dao.Resource, error) {
	// List and find
	resources, err := d.List(ctx)
	if err != nil {
		return nil, err
	}
	for _, r := range resources {
		if r.GetID() == id || r.GetARN() == id {
			return r, nil
		}
	}
	return nil, fmt.Errorf("location not found: %s", id)
}

// Delete deletes a DataSync location.
func (d *LocationDAO) Delete(ctx context.Context, id string) error {
	resource, err := d.Get(ctx, id)
	if err != nil {
		return err
	}
	locationArn := resource.GetARN()

	_, err = d.client.DeleteLocation(ctx, &datasync.DeleteLocationInput{
		LocationArn: &locationArn,
	})
	if err != nil {
		return apperrors.Wrap(err, "delete datasync location")
	}
	return nil
}

// LocationResource wraps a DataSync location.
type LocationResource struct {
	dao.BaseResource
	Location *types.LocationListEntry
}

// NewLocationResource creates a new LocationResource.
func NewLocationResource(loc types.LocationListEntry) *LocationResource {
	arn := appaws.Str(loc.LocationArn)
	return &LocationResource{
		BaseResource: dao.BaseResource{
			ID:   extractLocationID(arn),
			ARN:  arn,
			Data: loc,
		},
		Location: &loc,
	}
}

// extractLocationID extracts the location ID from an ARN.
func extractLocationID(arn string) string {
	// Format: arn:aws:datasync:region:account:location/loc-xxx
	if idx := strings.LastIndex(arn, "/"); idx != -1 {
		return arn[idx+1:]
	}
	return arn
}

// LocationUri returns the location URI.
func (r *LocationResource) LocationUri() string {
	if r.Location != nil && r.Location.LocationUri != nil {
		return *r.Location.LocationUri
	}
	return ""
}
