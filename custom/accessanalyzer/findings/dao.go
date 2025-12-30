package findings

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/accessanalyzer"
	"github.com/aws/aws-sdk-go-v2/service/accessanalyzer/types"

	appaws "github.com/clawscli/claws/internal/aws"
	"github.com/clawscli/claws/internal/dao"
)

// FindingDAO provides data access for Access Analyzer findings.
type FindingDAO struct {
	dao.BaseDAO
	client *accessanalyzer.Client
}

// NewFindingDAO creates a new FindingDAO.
func NewFindingDAO(ctx context.Context) (dao.DAO, error) {
	cfg, err := appaws.NewConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("new accessanalyzer/findings dao: %w", err)
	}
	return &FindingDAO{
		BaseDAO: dao.NewBaseDAO("accessanalyzer", "findings"),
		client:  accessanalyzer.NewFromConfig(cfg),
	}, nil
}

// List returns all findings for an analyzer.
func (d *FindingDAO) List(ctx context.Context) ([]dao.Resource, error) {
	analyzerArn := dao.GetFilterFromContext(ctx, "AnalyzerArn")
	if analyzerArn == "" {
		return nil, fmt.Errorf("analyzer ARN filter required")
	}

	findings, err := appaws.Paginate(ctx, func(token *string) ([]types.FindingSummary, *string, error) {
		output, err := d.client.ListFindings(ctx, &accessanalyzer.ListFindingsInput{
			AnalyzerArn: &analyzerArn,
			NextToken:   token,
		})
		if err != nil {
			return nil, nil, fmt.Errorf("list findings: %w", err)
		}
		return output.Findings, output.NextToken, nil
	})
	if err != nil {
		return nil, err
	}

	resources := make([]dao.Resource, len(findings))
	for i, finding := range findings {
		resources[i] = NewFindingResource(finding, analyzerArn)
	}
	return resources, nil
}

// Get returns a specific finding by ID.
func (d *FindingDAO) Get(ctx context.Context, id string) (dao.Resource, error) {
	analyzerArn := dao.GetFilterFromContext(ctx, "AnalyzerArn")
	if analyzerArn == "" {
		return nil, fmt.Errorf("analyzer ARN filter required")
	}

	output, err := d.client.GetFinding(ctx, &accessanalyzer.GetFindingInput{
		AnalyzerArn: &analyzerArn,
		Id:          &id,
	})
	if err != nil {
		return nil, fmt.Errorf("get finding %s: %w", id, err)
	}
	return NewFindingResourceFromDetail(*output.Finding, analyzerArn), nil
}

// Delete is not supported for findings (archive instead).
func (d *FindingDAO) Delete(ctx context.Context, id string) error {
	return fmt.Errorf("delete not supported for findings; use archive action instead")
}

// FindingResource wraps an Access Analyzer finding.
type FindingResource struct {
	dao.BaseResource
	Summary     *types.FindingSummary
	Detail      *types.Finding
	AnalyzerArn string
}

// NewFindingResource creates a new FindingResource from summary.
func NewFindingResource(finding types.FindingSummary, analyzerArn string) *FindingResource {
	return &FindingResource{
		BaseResource: dao.BaseResource{
			ID: appaws.Str(finding.Id),
		},
		Summary:     &finding,
		AnalyzerArn: analyzerArn,
	}
}

// NewFindingResourceFromDetail creates a new FindingResource from detail.
func NewFindingResourceFromDetail(finding types.Finding, analyzerArn string) *FindingResource {
	return &FindingResource{
		BaseResource: dao.BaseResource{
			ID: appaws.Str(finding.Id),
		},
		Detail:      &finding,
		AnalyzerArn: analyzerArn,
	}
}

// FindingId returns the finding ID.
func (r *FindingResource) FindingId() string {
	return r.ID
}

// ResourceType returns the resource type.
func (r *FindingResource) ResourceType() string {
	if r.Summary != nil {
		return string(r.Summary.ResourceType)
	}
	if r.Detail != nil {
		return string(r.Detail.ResourceType)
	}
	return ""
}

// Resource returns the resource ARN.
func (r *FindingResource) Resource() string {
	if r.Summary != nil {
		return appaws.Str(r.Summary.Resource)
	}
	if r.Detail != nil {
		return appaws.Str(r.Detail.Resource)
	}
	return ""
}

// Status returns the finding status.
func (r *FindingResource) Status() string {
	if r.Summary != nil {
		return string(r.Summary.Status)
	}
	if r.Detail != nil {
		return string(r.Detail.Status)
	}
	return ""
}

// CreatedAt returns when the finding was created.
func (r *FindingResource) CreatedAt() *time.Time {
	if r.Summary != nil {
		return r.Summary.CreatedAt
	}
	if r.Detail != nil {
		return r.Detail.CreatedAt
	}
	return nil
}

// UpdatedAt returns when the finding was last updated.
func (r *FindingResource) UpdatedAt() *time.Time {
	if r.Summary != nil {
		return r.Summary.UpdatedAt
	}
	if r.Detail != nil {
		return r.Detail.UpdatedAt
	}
	return nil
}

// IsPublic returns whether the resource is publicly accessible.
func (r *FindingResource) IsPublic() bool {
	if r.Summary != nil {
		return r.Summary.IsPublic != nil && *r.Summary.IsPublic
	}
	if r.Detail != nil {
		return r.Detail.IsPublic != nil && *r.Detail.IsPublic
	}
	return false
}

// ResourceOwnerAccount returns the resource owner account.
func (r *FindingResource) ResourceOwnerAccount() string {
	if r.Summary != nil {
		return appaws.Str(r.Summary.ResourceOwnerAccount)
	}
	if r.Detail != nil {
		return appaws.Str(r.Detail.ResourceOwnerAccount)
	}
	return ""
}

// Principal returns the principal.
func (r *FindingResource) Principal() map[string]string {
	if r.Summary != nil {
		return r.Summary.Principal
	}
	if r.Detail != nil {
		return r.Detail.Principal
	}
	return nil
}

// Action returns the action.
func (r *FindingResource) Action() []string {
	if r.Summary != nil {
		return r.Summary.Action
	}
	if r.Detail != nil {
		return r.Detail.Action
	}
	return nil
}

// Condition returns the condition.
func (r *FindingResource) Condition() map[string]string {
	if r.Summary != nil {
		return r.Summary.Condition
	}
	if r.Detail != nil {
		return r.Detail.Condition
	}
	return nil
}

// Error returns the error (if any).
func (r *FindingResource) Error() string {
	if r.Summary != nil && r.Summary.Error != nil {
		return appaws.Str(r.Summary.Error)
	}
	if r.Detail != nil && r.Detail.Error != nil {
		return appaws.Str(r.Detail.Error)
	}
	return ""
}

// AnalyzedAt returns when the resource was analyzed.
func (r *FindingResource) AnalyzedAt() *time.Time {
	if r.Detail != nil {
		return r.Detail.AnalyzedAt
	}
	return nil
}

// Sources returns the finding sources.
func (r *FindingResource) Sources() []types.FindingSource {
	if r.Summary != nil {
		return r.Summary.Sources
	}
	if r.Detail != nil {
		return r.Detail.Sources
	}
	return nil
}

// ResourceControlPolicyRestriction returns the RCP restriction.
func (r *FindingResource) ResourceControlPolicyRestriction() string {
	if r.Detail != nil {
		return string(r.Detail.ResourceControlPolicyRestriction)
	}
	return ""
}
