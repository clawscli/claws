package graphs

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/detective"
	"github.com/aws/aws-sdk-go-v2/service/detective/types"

	appaws "github.com/clawscli/claws/internal/aws"
	"github.com/clawscli/claws/internal/dao"
	apperrors "github.com/clawscli/claws/internal/errors"
)

// GraphDAO provides data access for Detective graphs.
type GraphDAO struct {
	dao.BaseDAO
	client *detective.Client
}

// NewGraphDAO creates a new GraphDAO.
func NewGraphDAO(ctx context.Context) (dao.DAO, error) {
	cfg, err := appaws.NewConfig(ctx)
	if err != nil {
		return nil, apperrors.Wrap(err, "new "+ServiceResourcePath+" dao")
	}
	return &GraphDAO{
		BaseDAO: dao.NewBaseDAO("detective", "graphs"),
		client:  detective.NewFromConfig(cfg),
	}, nil
}

// List returns all Detective graphs.
func (d *GraphDAO) List(ctx context.Context) ([]dao.Resource, error) {
	graphs, err := appaws.Paginate(ctx, func(token *string) ([]types.Graph, *string, error) {
		output, err := d.client.ListGraphs(ctx, &detective.ListGraphsInput{
			NextToken: token,
		})
		if err != nil {
			return nil, nil, apperrors.Wrap(err, "list detective graphs")
		}
		return output.GraphList, output.NextToken, nil
	})
	if err != nil {
		return nil, err
	}

	resources := make([]dao.Resource, len(graphs))
	for i, graph := range graphs {
		resources[i] = NewGraphResource(graph)
	}
	return resources, nil
}

// Get returns a specific graph (not supported - graphs only have ARN).
func (d *GraphDAO) Get(ctx context.Context, arn string) (dao.Resource, error) {
	// Detective doesn't have a GetGraph API, so we list and find
	resources, err := d.List(ctx)
	if err != nil {
		return nil, err
	}
	for _, r := range resources {
		if r.GetARN() == arn {
			return r, nil
		}
	}
	return nil, fmt.Errorf("graph not found: %s", arn)
}

// Delete deletes a Detective graph.
func (d *GraphDAO) Delete(ctx context.Context, arn string) error {
	_, err := d.client.DeleteGraph(ctx, &detective.DeleteGraphInput{
		GraphArn: &arn,
	})
	if err != nil {
		return apperrors.Wrap(err, "delete detective graph")
	}
	return nil
}

// GraphResource wraps a Detective graph.
type GraphResource struct {
	dao.BaseResource
	Graph *types.Graph
}

// NewGraphResource creates a new GraphResource.
func NewGraphResource(graph types.Graph) *GraphResource {
	// Extract graph ID from ARN (last part after /)
	arn := appaws.Str(graph.Arn)
	id := arn
	if idx := len(arn) - 1; idx > 0 {
		for i := len(arn) - 1; i >= 0; i-- {
			if arn[i] == '/' {
				id = arn[i+1:]
				break
			}
		}
	}

	return &GraphResource{
		BaseResource: dao.BaseResource{
			ID:  id,
			ARN: arn,
		},
		Graph: &graph,
	}
}

// GraphArn returns the graph ARN.
func (r *GraphResource) GraphArn() string {
	return r.ARN
}

// CreatedTime returns when the graph was created.
func (r *GraphResource) CreatedTime() *time.Time {
	if r.Graph != nil {
		return r.Graph.CreatedTime
	}
	return nil
}
