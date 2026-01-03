package findings

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/inspector2"
	"github.com/aws/aws-sdk-go-v2/service/inspector2/types"

	appaws "github.com/clawscli/claws/internal/aws"
	"github.com/clawscli/claws/internal/dao"
	apperrors "github.com/clawscli/claws/internal/errors"
)

// FindingDAO provides data access for Inspector2 findings
type FindingDAO struct {
	dao.BaseDAO
	client *inspector2.Client
}

// NewFindingDAO creates a new FindingDAO
func NewFindingDAO(ctx context.Context) (dao.DAO, error) {
	cfg, err := appaws.NewConfig(ctx)
	if err != nil {
		return nil, apperrors.Wrap(err, "new "+ServiceResourcePath+" dao")
	}
	return &FindingDAO{
		BaseDAO: dao.NewBaseDAO("inspector2", "findings"),
		client:  inspector2.NewFromConfig(cfg),
	}, nil
}

// List returns Inspector2 findings (first page only for backwards compatibility).
// For paginated access, use ListPage instead.
func (d *FindingDAO) List(ctx context.Context) ([]dao.Resource, error) {
	resources, _, err := d.ListPage(ctx, 100, "")
	return resources, err
}

// ListPage returns a page of Inspector2 findings.
// Implements dao.PaginatedDAO interface.
func (d *FindingDAO) ListPage(ctx context.Context, pageSize int, pageToken string) ([]dao.Resource, string, error) {
	// By default, list only ACTIVE findings
	filterCriteria := &types.FilterCriteria{
		FindingStatus: []types.StringFilter{
			{
				Comparison: types.StringComparisonEquals,
				Value:      appaws.StringPtr("ACTIVE"),
			},
		},
	}

	maxResults := int32(pageSize)
	if maxResults > 100 {
		maxResults = 100 // AWS API max
	}

	input := &inspector2.ListFindingsInput{
		FilterCriteria: filterCriteria,
		MaxResults:     &maxResults,
	}
	if pageToken != "" {
		input.NextToken = &pageToken
	}

	output, err := d.client.ListFindings(ctx, input)
	if err != nil {
		return nil, "", apperrors.Wrap(err, "list findings")
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

// Get returns a specific Inspector2 finding by ARN
func (d *FindingDAO) Get(ctx context.Context, arn string) (dao.Resource, error) {
	// Inspector2 doesn't have a GetFinding API, so we search with the ARN filter
	filterCriteria := &types.FilterCriteria{
		FindingArn: []types.StringFilter{
			{
				Comparison: types.StringComparisonEquals,
				Value:      &arn,
			},
		},
	}

	output, err := d.client.ListFindings(ctx, &inspector2.ListFindingsInput{
		FilterCriteria: filterCriteria,
		MaxResults:     appaws.Int32Ptr(1),
	})
	if err != nil {
		return nil, apperrors.Wrapf(err, "get finding %s", arn)
	}

	if len(output.Findings) == 0 {
		return nil, fmt.Errorf("finding %s not found", arn)
	}

	return NewFindingResource(output.Findings[0]), nil
}

// Delete is not supported for Inspector2 findings
func (d *FindingDAO) Delete(_ context.Context, _ string) error {
	return fmt.Errorf("delete not supported for Inspector2 findings")
}

// FindingResource represents an Inspector2 finding
type FindingResource struct {
	dao.BaseResource
	Finding types.Finding
}

// NewFindingResource creates a new FindingResource
func NewFindingResource(finding types.Finding) *FindingResource {
	arn := appaws.Str(finding.FindingArn)
	title := appaws.Str(finding.Title)

	return &FindingResource{
		BaseResource: dao.BaseResource{
			ID:   arn,
			Name: title,
			ARN:  arn,
			Tags: make(map[string]string),
			Data: finding,
		},
		Finding: finding,
	}
}

// FindingArn returns the finding ARN
func (r *FindingResource) FindingArn() string {
	return appaws.Str(r.Finding.FindingArn)
}

// Title returns the finding title
func (r *FindingResource) Title() string {
	return appaws.Str(r.Finding.Title)
}

// TitleShort returns a shortened title for display
func (r *FindingResource) TitleShort() string {
	title := r.Title()
	if len(title) > 50 {
		return title[:47] + "..."
	}
	return title
}

// Description returns the finding description
func (r *FindingResource) Description() string {
	return appaws.Str(r.Finding.Description)
}

// Severity returns the finding severity
func (r *FindingResource) Severity() string {
	return string(r.Finding.Severity)
}

// Status returns the finding status
func (r *FindingResource) Status() string {
	return string(r.Finding.Status)
}

// Type returns the finding type
func (r *FindingResource) Type() string {
	return string(r.Finding.Type)
}

// ResourceType returns the affected resource type
func (r *FindingResource) ResourceType() string {
	if len(r.Finding.Resources) > 0 {
		return string(r.Finding.Resources[0].Type)
	}
	return ""
}

// ResourceId returns the affected resource ID
func (r *FindingResource) ResourceId() string {
	if len(r.Finding.Resources) > 0 {
		return appaws.Str(r.Finding.Resources[0].Id)
	}
	return ""
}

// Resources returns all affected resources
func (r *FindingResource) Resources() []types.Resource {
	return r.Finding.Resources
}

// FirstObservedAt returns when the finding was first observed
func (r *FindingResource) FirstObservedAt() string {
	if r.Finding.FirstObservedAt != nil {
		return r.Finding.FirstObservedAt.Format("2006-01-02 15:04:05")
	}
	return ""
}

// FirstObservedAtTime returns when the finding was first observed as time.Time
func (r *FindingResource) FirstObservedAtTime() *time.Time {
	return r.Finding.FirstObservedAt
}

// LastObservedAt returns when the finding was last observed
func (r *FindingResource) LastObservedAt() string {
	if r.Finding.LastObservedAt != nil {
		return r.Finding.LastObservedAt.Format("2006-01-02 15:04:05")
	}
	return ""
}

// UpdatedAt returns when the finding was last updated
func (r *FindingResource) UpdatedAt() string {
	if r.Finding.UpdatedAt != nil {
		return r.Finding.UpdatedAt.Format("2006-01-02 15:04:05")
	}
	return ""
}

// InspectorScore returns the CVSS score if available
func (r *FindingResource) InspectorScore() float64 {
	if r.Finding.InspectorScore != nil {
		return *r.Finding.InspectorScore
	}
	return 0
}

// VulnerabilityId returns the vulnerability ID (CVE, etc.)
func (r *FindingResource) VulnerabilityId() string {
	if r.Finding.PackageVulnerabilityDetails != nil {
		return appaws.Str(r.Finding.PackageVulnerabilityDetails.VulnerabilityId)
	}
	return ""
}

// VendorSeverity returns the vendor-provided severity
func (r *FindingResource) VendorSeverity() string {
	if r.Finding.PackageVulnerabilityDetails != nil {
		return appaws.Str(r.Finding.PackageVulnerabilityDetails.VendorSeverity)
	}
	return ""
}

// Remediation returns the remediation recommendation
func (r *FindingResource) Remediation() string {
	if r.Finding.Remediation != nil && r.Finding.Remediation.Recommendation != nil {
		return appaws.Str(r.Finding.Remediation.Recommendation.Text)
	}
	return ""
}
