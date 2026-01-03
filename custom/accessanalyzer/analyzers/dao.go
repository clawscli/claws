package analyzers

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/accessanalyzer"
	"github.com/aws/aws-sdk-go-v2/service/accessanalyzer/types"

	appaws "github.com/clawscli/claws/internal/aws"
	"github.com/clawscli/claws/internal/dao"
	apperrors "github.com/clawscli/claws/internal/errors"
)

// AnalyzerDAO provides data access for IAM Access Analyzer analyzers.
type AnalyzerDAO struct {
	dao.BaseDAO
	client *accessanalyzer.Client
}

// NewAnalyzerDAO creates a new AnalyzerDAO.
func NewAnalyzerDAO(ctx context.Context) (dao.DAO, error) {
	cfg, err := appaws.NewConfig(ctx)
	if err != nil {
		return nil, apperrors.Wrap(err, "new "+ServiceResourcePath+" dao")
	}
	return &AnalyzerDAO{
		BaseDAO: dao.NewBaseDAO("accessanalyzer", "analyzers"),
		client:  accessanalyzer.NewFromConfig(cfg),
	}, nil
}

// List returns all Access Analyzer analyzers.
func (d *AnalyzerDAO) List(ctx context.Context) ([]dao.Resource, error) {
	analyzers, err := appaws.Paginate(ctx, func(token *string) ([]types.AnalyzerSummary, *string, error) {
		output, err := d.client.ListAnalyzers(ctx, &accessanalyzer.ListAnalyzersInput{
			NextToken: token,
		})
		if err != nil {
			return nil, nil, apperrors.Wrap(err, "list analyzers")
		}
		return output.Analyzers, output.NextToken, nil
	})
	if err != nil {
		return nil, err
	}

	resources := make([]dao.Resource, len(analyzers))
	for i, analyzer := range analyzers {
		resources[i] = NewAnalyzerResource(analyzer)
	}
	return resources, nil
}

// Get returns a specific analyzer by name.
func (d *AnalyzerDAO) Get(ctx context.Context, name string) (dao.Resource, error) {
	output, err := d.client.GetAnalyzer(ctx, &accessanalyzer.GetAnalyzerInput{
		AnalyzerName: &name,
	})
	if err != nil {
		return nil, apperrors.Wrapf(err, "get analyzer %s", name)
	}
	return NewAnalyzerResourceFromDetail(*output.Analyzer), nil
}

// Delete deletes an analyzer by name.
func (d *AnalyzerDAO) Delete(ctx context.Context, name string) error {
	_, err := d.client.DeleteAnalyzer(ctx, &accessanalyzer.DeleteAnalyzerInput{
		AnalyzerName: &name,
	})
	if err != nil {
		return apperrors.Wrapf(err, "delete analyzer %s", name)
	}
	return nil
}

// AnalyzerResource wraps an Access Analyzer analyzer.
type AnalyzerResource struct {
	dao.BaseResource
	Summary *types.AnalyzerSummary
	Detail  *types.AnalyzerSummary
}

// NewAnalyzerResource creates a new AnalyzerResource from summary.
func NewAnalyzerResource(analyzer types.AnalyzerSummary) *AnalyzerResource {
	return &AnalyzerResource{
		BaseResource: dao.BaseResource{
			ID:  appaws.Str(analyzer.Name),
			ARN: appaws.Str(analyzer.Arn),
		},
		Summary: &analyzer,
	}
}

// NewAnalyzerResourceFromDetail creates a new AnalyzerResource from detail.
func NewAnalyzerResourceFromDetail(analyzer types.AnalyzerSummary) *AnalyzerResource {
	return &AnalyzerResource{
		BaseResource: dao.BaseResource{
			ID:  appaws.Str(analyzer.Name),
			ARN: appaws.Str(analyzer.Arn),
		},
		Detail: &analyzer,
	}
}

// Name returns the analyzer name.
func (r *AnalyzerResource) Name() string {
	return r.ID
}

// Type returns the analyzer type.
func (r *AnalyzerResource) Type() string {
	if r.Summary != nil {
		return string(r.Summary.Type)
	}
	if r.Detail != nil {
		return string(r.Detail.Type)
	}
	return ""
}

// Status returns the analyzer status.
func (r *AnalyzerResource) Status() string {
	if r.Summary != nil {
		return string(r.Summary.Status)
	}
	if r.Detail != nil {
		return string(r.Detail.Status)
	}
	return ""
}

// CreatedAt returns when the analyzer was created.
func (r *AnalyzerResource) CreatedAt() *time.Time {
	if r.Summary != nil {
		return r.Summary.CreatedAt
	}
	if r.Detail != nil {
		return r.Detail.CreatedAt
	}
	return nil
}

// LastResourceAnalyzed returns the last resource analyzed.
func (r *AnalyzerResource) LastResourceAnalyzed() string {
	if r.Summary != nil {
		return appaws.Str(r.Summary.LastResourceAnalyzed)
	}
	if r.Detail != nil {
		return appaws.Str(r.Detail.LastResourceAnalyzed)
	}
	return ""
}

// LastResourceAnalyzedAt returns when the last resource was analyzed.
func (r *AnalyzerResource) LastResourceAnalyzedAt() *time.Time {
	if r.Summary != nil {
		return r.Summary.LastResourceAnalyzedAt
	}
	if r.Detail != nil {
		return r.Detail.LastResourceAnalyzedAt
	}
	return nil
}

// StatusReason returns the status reason.
func (r *AnalyzerResource) StatusReason() string {
	if r.Summary != nil && r.Summary.StatusReason != nil {
		return string(r.Summary.StatusReason.Code)
	}
	if r.Detail != nil && r.Detail.StatusReason != nil {
		return string(r.Detail.StatusReason.Code)
	}
	return ""
}

// Tags returns the analyzer tags.
func (r *AnalyzerResource) Tags() map[string]string {
	if r.Summary != nil {
		return r.Summary.Tags
	}
	if r.Detail != nil {
		return r.Detail.Tags
	}
	return nil
}

// Configuration returns the analyzer configuration.
func (r *AnalyzerResource) Configuration() types.AnalyzerConfiguration {
	if r.Summary != nil {
		return r.Summary.Configuration
	}
	if r.Detail != nil {
		return r.Detail.Configuration
	}
	return nil
}
