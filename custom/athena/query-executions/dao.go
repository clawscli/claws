package queryexecutions

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/athena"
	"github.com/aws/aws-sdk-go-v2/service/athena/types"

	appaws "github.com/clawscli/claws/internal/aws"
	"github.com/clawscli/claws/internal/dao"
	apperrors "github.com/clawscli/claws/internal/errors"
)

// QueryExecutionDAO provides data access for Athena query executions.
type QueryExecutionDAO struct {
	dao.BaseDAO
	client *athena.Client
}

// NewQueryExecutionDAO creates a new QueryExecutionDAO.
func NewQueryExecutionDAO(ctx context.Context) (dao.DAO, error) {
	cfg, err := appaws.NewConfig(ctx)
	if err != nil {
		return nil, apperrors.Wrap(err, "new athena/query-executions dao")
	}
	return &QueryExecutionDAO{
		BaseDAO: dao.NewBaseDAO("athena", "query-executions"),
		client:  athena.NewFromConfig(cfg),
	}, nil
}

// List returns query executions (first page only for backwards compatibility).
// For paginated access, use ListPage instead.
func (d *QueryExecutionDAO) List(ctx context.Context) ([]dao.Resource, error) {
	resources, _, err := d.ListPage(ctx, 50, "")
	return resources, err
}

// ListPage returns a page of Athena query executions.
// Implements dao.PaginatedDAO interface.
func (d *QueryExecutionDAO) ListPage(ctx context.Context, pageSize int, pageToken string) ([]dao.Resource, string, error) {
	workGroup := dao.GetFilterFromContext(ctx, "WorkGroup")
	if workGroup == "" {
		return nil, "", fmt.Errorf("workgroup filter required")
	}

	maxResults := int32(pageSize)
	if maxResults > 50 {
		maxResults = 50 // AWS API max
	}

	// Get query execution IDs
	listInput := &athena.ListQueryExecutionsInput{
		WorkGroup:  &workGroup,
		MaxResults: &maxResults,
	}
	if pageToken != "" {
		listInput.NextToken = &pageToken
	}

	listOutput, err := d.client.ListQueryExecutions(ctx, listInput)
	if err != nil {
		return nil, "", apperrors.Wrap(err, "list athena query executions")
	}

	if len(listOutput.QueryExecutionIds) == 0 {
		return []dao.Resource{}, "", nil
	}

	// Batch get query execution details
	batchOutput, err := d.client.BatchGetQueryExecution(ctx, &athena.BatchGetQueryExecutionInput{
		QueryExecutionIds: listOutput.QueryExecutionIds,
	})
	if err != nil {
		return nil, "", apperrors.Wrap(err, "batch get query executions")
	}

	resources := make([]dao.Resource, 0, len(batchOutput.QueryExecutions))
	for _, qe := range batchOutput.QueryExecutions {
		resources = append(resources, NewQueryExecutionResource(qe))
	}

	nextToken := ""
	if listOutput.NextToken != nil {
		nextToken = *listOutput.NextToken
	}

	return resources, nextToken, nil
}

// Get returns a specific query execution by ID.
func (d *QueryExecutionDAO) Get(ctx context.Context, id string) (dao.Resource, error) {
	output, err := d.client.GetQueryExecution(ctx, &athena.GetQueryExecutionInput{
		QueryExecutionId: &id,
	})
	if err != nil {
		return nil, apperrors.Wrapf(err, "get athena query execution %s", id)
	}
	return NewQueryExecutionResource(*output.QueryExecution), nil
}

// Delete stops a query execution.
func (d *QueryExecutionDAO) Delete(ctx context.Context, id string) error {
	_, err := d.client.StopQueryExecution(ctx, &athena.StopQueryExecutionInput{
		QueryExecutionId: &id,
	})
	if err != nil {
		return apperrors.Wrapf(err, "stop athena query execution %s", id)
	}
	return nil
}

// QueryExecutionResource wraps an Athena query execution.
type QueryExecutionResource struct {
	dao.BaseResource
	Item types.QueryExecution
}

// NewQueryExecutionResource creates a new QueryExecutionResource.
func NewQueryExecutionResource(qe types.QueryExecution) *QueryExecutionResource {
	return &QueryExecutionResource{
		BaseResource: dao.BaseResource{
			ID:  appaws.Str(qe.QueryExecutionId),
			ARN: "",
		},
		Item: qe,
	}
}

// Query returns the query string (truncated).
func (r *QueryExecutionResource) Query() string {
	q := appaws.Str(r.Item.Query)
	if len(q) > 100 {
		return q[:100] + "..."
	}
	return q
}

// State returns the query state.
func (r *QueryExecutionResource) State() string {
	if r.Item.Status != nil {
		return string(r.Item.Status.State)
	}
	return ""
}

// WorkGroup returns the workgroup name.
func (r *QueryExecutionResource) WorkGroup() string {
	return appaws.Str(r.Item.WorkGroup)
}

// Database returns the database name.
func (r *QueryExecutionResource) Database() string {
	if r.Item.QueryExecutionContext != nil {
		return appaws.Str(r.Item.QueryExecutionContext.Database)
	}
	return ""
}

// SubmissionTime returns when the query was submitted.
func (r *QueryExecutionResource) SubmissionTime() *time.Time {
	if r.Item.Status != nil {
		return r.Item.Status.SubmissionDateTime
	}
	return nil
}

// CompletionTime returns when the query completed.
func (r *QueryExecutionResource) CompletionTime() *time.Time {
	if r.Item.Status != nil {
		return r.Item.Status.CompletionDateTime
	}
	return nil
}

// DataScannedBytes returns the data scanned in bytes.
func (r *QueryExecutionResource) DataScannedBytes() int64 {
	if r.Item.Statistics != nil {
		return appaws.Int64(r.Item.Statistics.DataScannedInBytes)
	}
	return 0
}

// ExecutionTimeMs returns the execution time in milliseconds.
func (r *QueryExecutionResource) ExecutionTimeMs() int64 {
	if r.Item.Statistics != nil {
		return appaws.Int64(r.Item.Statistics.EngineExecutionTimeInMillis)
	}
	return 0
}

// StateChangeReason returns the state change reason.
func (r *QueryExecutionResource) StateChangeReason() string {
	if r.Item.Status != nil {
		return appaws.Str(r.Item.Status.StateChangeReason)
	}
	return ""
}

// OutputLocation returns the output location.
func (r *QueryExecutionResource) OutputLocation() string {
	if r.Item.ResultConfiguration != nil {
		return appaws.Str(r.Item.ResultConfiguration.OutputLocation)
	}
	return ""
}
