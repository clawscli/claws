package findings

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/securityhub"
	"github.com/aws/aws-sdk-go-v2/service/securityhub/types"

	appaws "github.com/clawscli/claws/internal/aws"
	"github.com/clawscli/claws/internal/dao"
	apperrors "github.com/clawscli/claws/internal/errors"
)

// FindingDAO provides data access for Security Hub findings.
type FindingDAO struct {
	dao.BaseDAO
	client *securityhub.Client
}

// NewFindingDAO creates a new FindingDAO.
func NewFindingDAO(ctx context.Context) (dao.DAO, error) {
	cfg, err := appaws.NewConfig(ctx)
	if err != nil {
		return nil, apperrors.Wrap(err, "new "+ServiceResourcePath+" dao")
	}
	return &FindingDAO{
		BaseDAO: dao.NewBaseDAO("securityhub", "findings"),
		client:  securityhub.NewFromConfig(cfg),
	}, nil
}

// List returns Security Hub findings (first page only for backwards compatibility).
// For paginated access, use ListPage instead.
func (d *FindingDAO) List(ctx context.Context) ([]dao.Resource, error) {
	resources, _, err := d.ListPage(ctx, 100, "")
	return resources, err
}

// ListPage returns a page of Security Hub findings.
// Implements dao.PaginatedDAO interface.
func (d *FindingDAO) ListPage(ctx context.Context, pageSize int, pageToken string) ([]dao.Resource, string, error) {
	maxResults := int32(pageSize)
	if maxResults > 100 {
		maxResults = 100 // AWS API max
	}

	input := &securityhub.GetFindingsInput{
		MaxResults: &maxResults,
	}
	if pageToken != "" {
		input.NextToken = &pageToken
	}

	output, err := d.client.GetFindings(ctx, input)
	if err != nil {
		return nil, "", apperrors.Wrap(err, "get security hub findings")
	}

	resources := make([]dao.Resource, len(output.Findings))
	for i, finding := range output.Findings {
		resources[i] = NewFindingResource(finding)
	}

	nextToken := ""
	if output.NextToken != nil {
		nextToken = *output.NextToken
	}

	return resources, nextToken, nil
}

// Get returns a specific finding by ID.
func (d *FindingDAO) Get(ctx context.Context, id string) (dao.Resource, error) {
	output, err := d.client.GetFindings(ctx, &securityhub.GetFindingsInput{
		Filters: &types.AwsSecurityFindingFilters{
			Id: []types.StringFilter{
				{
					Value:      &id,
					Comparison: types.StringFilterComparisonEquals,
				},
			},
		},
	})
	if err != nil {
		return nil, apperrors.Wrapf(err, "get finding %s", id)
	}
	if len(output.Findings) == 0 {
		return nil, fmt.Errorf("finding not found: %s", id)
	}
	return NewFindingResource(output.Findings[0]), nil
}

// Delete is not supported for findings.
func (d *FindingDAO) Delete(ctx context.Context, id string) error {
	return fmt.Errorf("delete not supported for security hub findings")
}

// FindingResource wraps a Security Hub finding.
type FindingResource struct {
	dao.BaseResource
	Item types.AwsSecurityFinding
}

// NewFindingResource creates a new FindingResource.
func NewFindingResource(finding types.AwsSecurityFinding) *FindingResource {
	return &FindingResource{
		BaseResource: dao.BaseResource{
			ID:  appaws.Str(finding.Id),
			ARN: appaws.Str(finding.Id),
		},
		Item: finding,
	}
}

// Title returns the finding title.
func (r *FindingResource) Title() string {
	return appaws.Str(r.Item.Title)
}

// Severity returns the severity label.
func (r *FindingResource) Severity() string {
	if r.Item.Severity != nil {
		return string(r.Item.Severity.Label)
	}
	return ""
}

// SeverityScore returns the severity score.
func (r *FindingResource) SeverityScore() float64 {
	if r.Item.Severity != nil && r.Item.Severity.Normalized != nil {
		return float64(*r.Item.Severity.Normalized)
	}
	return 0
}

// Status returns the workflow status.
func (r *FindingResource) Status() string {
	if r.Item.Workflow != nil {
		return string(r.Item.Workflow.Status)
	}
	return ""
}

// RecordState returns the record state.
func (r *FindingResource) RecordState() string {
	return string(r.Item.RecordState)
}

// ProductName returns the product name.
func (r *FindingResource) ProductName() string {
	return appaws.Str(r.Item.ProductName)
}

// CompanyName returns the company name.
func (r *FindingResource) CompanyName() string {
	return appaws.Str(r.Item.CompanyName)
}

// Description returns the finding description.
func (r *FindingResource) Description() string {
	return appaws.Str(r.Item.Description)
}

// ResourceType returns the affected resource type.
func (r *FindingResource) ResourceType() string {
	if len(r.Item.Resources) > 0 {
		return appaws.Str(r.Item.Resources[0].Type)
	}
	return ""
}

// ResourceId returns the affected resource ID.
func (r *FindingResource) ResourceId() string {
	if len(r.Item.Resources) > 0 {
		return appaws.Str(r.Item.Resources[0].Id)
	}
	return ""
}

// GeneratorId returns the generator ID.
func (r *FindingResource) GeneratorId() string {
	return appaws.Str(r.Item.GeneratorId)
}

// CreatedAt returns when the finding was created.
func (r *FindingResource) CreatedAt() *time.Time {
	if r.Item.CreatedAt != nil {
		if t, err := time.Parse(time.RFC3339, *r.Item.CreatedAt); err == nil {
			return &t
		}
	}
	return nil
}

// UpdatedAt returns when the finding was last updated.
func (r *FindingResource) UpdatedAt() *time.Time {
	if r.Item.UpdatedAt != nil {
		if t, err := time.Parse(time.RFC3339, *r.Item.UpdatedAt); err == nil {
			return &t
		}
	}
	return nil
}

// Compliance returns the compliance status.
func (r *FindingResource) ComplianceStatus() string {
	if r.Item.Compliance != nil {
		return string(r.Item.Compliance.Status)
	}
	return ""
}

// AwsAccountId returns the AWS account ID.
func (r *FindingResource) AwsAccountId() string {
	return appaws.Str(r.Item.AwsAccountId)
}

// Region returns the AWS region.
func (r *FindingResource) Region() string {
	return appaws.Str(r.Item.Region)
}

// Confidence returns the finding confidence (0-100).
func (r *FindingResource) Confidence() int32 {
	if r.Item.Confidence != nil {
		return *r.Item.Confidence
	}
	return 0
}

// Criticality returns the finding criticality (0-100).
func (r *FindingResource) Criticality() int32 {
	if r.Item.Criticality != nil {
		return *r.Item.Criticality
	}
	return 0
}

// ProductArn returns the product ARN.
func (r *FindingResource) ProductArn() string {
	return appaws.Str(r.Item.ProductArn)
}

// Remediation returns the remediation recommendation.
func (r *FindingResource) Remediation() string {
	if r.Item.Remediation != nil && r.Item.Remediation.Recommendation != nil {
		return appaws.Str(r.Item.Remediation.Recommendation.Text)
	}
	return ""
}

// RemediationUrl returns the remediation URL.
func (r *FindingResource) RemediationUrl() string {
	if r.Item.Remediation != nil && r.Item.Remediation.Recommendation != nil {
		return appaws.Str(r.Item.Remediation.Recommendation.Url)
	}
	return ""
}

// SourceUrl returns the source URL.
func (r *FindingResource) SourceUrl() string {
	return appaws.Str(r.Item.SourceUrl)
}

// VerificationState returns the verification state.
func (r *FindingResource) VerificationState() string {
	return string(r.Item.VerificationState)
}

// Types returns the finding types.
func (r *FindingResource) Types() []string {
	return r.Item.Types
}
