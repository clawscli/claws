package findings

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/guardduty"
	"github.com/aws/aws-sdk-go-v2/service/guardduty/types"

	appaws "github.com/clawscli/claws/internal/aws"
	"github.com/clawscli/claws/internal/dao"
	apperrors "github.com/clawscli/claws/internal/errors"
)

// FindingDAO provides data access for GuardDuty findings
type FindingDAO struct {
	dao.BaseDAO
	client *guardduty.Client
}

// NewFindingDAO creates a new FindingDAO
func NewFindingDAO(ctx context.Context) (dao.DAO, error) {
	cfg, err := appaws.NewConfig(ctx)
	if err != nil {
		return nil, apperrors.Wrap(err, "new "+ServiceResourcePath+" dao")
	}
	return &FindingDAO{
		BaseDAO: dao.NewBaseDAO("guardduty", "findings"),
		client:  guardduty.NewFromConfig(cfg),
	}, nil
}

// List returns GuardDuty findings (first page only for backwards compatibility).
// For paginated access, use ListPage instead.
func (d *FindingDAO) List(ctx context.Context) ([]dao.Resource, error) {
	resources, _, err := d.ListPage(ctx, 50, "")
	return resources, err
}

// ListPage returns a page of GuardDuty findings.
// Implements dao.PaginatedDAO interface.
func (d *FindingDAO) ListPage(ctx context.Context, pageSize int, pageToken string) ([]dao.Resource, string, error) {
	// Get detector ID from filter context
	detectorId := dao.GetFilterFromContext(ctx, "DetectorId")
	if detectorId == "" {
		return nil, "", fmt.Errorf("detector ID filter required")
	}

	maxResults := int32(pageSize)
	if maxResults > 50 {
		maxResults = 50 // AWS API max
	}

	listInput := &guardduty.ListFindingsInput{
		DetectorId: &detectorId,
		MaxResults: &maxResults,
	}
	if pageToken != "" {
		listInput.NextToken = &pageToken
	}

	listOutput, err := d.client.ListFindings(ctx, listInput)
	if err != nil {
		return nil, "", apperrors.Wrap(err, "list findings")
	}

	if len(listOutput.FindingIds) == 0 {
		return []dao.Resource{}, "", nil
	}

	// Get finding details
	getInput := &guardduty.GetFindingsInput{
		DetectorId: &detectorId,
		FindingIds: listOutput.FindingIds,
	}

	getOutput, err := d.client.GetFindings(ctx, getInput)
	if err != nil {
		return nil, "", apperrors.Wrap(err, "get findings")
	}

	resources := make([]dao.Resource, 0, len(getOutput.Findings))
	for _, finding := range getOutput.Findings {
		resources = append(resources, NewFindingResource(finding, detectorId))
	}

	nextToken := ""
	if listOutput.NextToken != nil {
		nextToken = *listOutput.NextToken
	}

	return resources, nextToken, nil
}

// Get returns a specific finding
func (d *FindingDAO) Get(ctx context.Context, id string) (dao.Resource, error) {
	detectorId := dao.GetFilterFromContext(ctx, "DetectorId")
	if detectorId == "" {
		return nil, fmt.Errorf("detector ID filter required")
	}

	input := &guardduty.GetFindingsInput{
		DetectorId: &detectorId,
		FindingIds: []string{id},
	}

	output, err := d.client.GetFindings(ctx, input)
	if err != nil {
		return nil, apperrors.Wrapf(err, "get finding %s", id)
	}

	if len(output.Findings) == 0 {
		return nil, fmt.Errorf("finding %s not found", id)
	}

	return NewFindingResource(output.Findings[0], detectorId), nil
}

// Delete archives a GuardDuty finding
func (d *FindingDAO) Delete(ctx context.Context, id string) error {
	detectorId := dao.GetFilterFromContext(ctx, "DetectorId")
	if detectorId == "" {
		return fmt.Errorf("detector ID filter required")
	}

	_, err := d.client.ArchiveFindings(ctx, &guardduty.ArchiveFindingsInput{
		DetectorId: &detectorId,
		FindingIds: []string{id},
	})
	if err != nil {
		return apperrors.Wrapf(err, "archive finding %s", id)
	}
	return nil
}

// Supports returns supported operations
func (d *FindingDAO) Supports(op dao.Operation) bool {
	switch op {
	case dao.OpList, dao.OpGet, dao.OpDelete:
		return true
	default:
		return false
	}
}

// FindingResource represents a GuardDuty finding
type FindingResource struct {
	dao.BaseResource
	Finding    types.Finding
	DetectorId string
}

// NewFindingResource creates a new FindingResource
func NewFindingResource(finding types.Finding, detectorId string) *FindingResource {
	id := appaws.Str(finding.Id)
	title := appaws.Str(finding.Title)
	arn := appaws.Str(finding.Arn)

	return &FindingResource{
		BaseResource: dao.BaseResource{
			ID:   id,
			Name: title,
			ARN:  arn,
			Tags: make(map[string]string),
			Data: finding,
		},
		Finding:    finding,
		DetectorId: detectorId,
	}
}

// FindingId returns the finding ID
func (r *FindingResource) FindingId() string {
	return appaws.Str(r.Finding.Id)
}

// Title returns the finding title
func (r *FindingResource) Title() string {
	return appaws.Str(r.Finding.Title)
}

// TitleShort returns a shortened title
func (r *FindingResource) TitleShort() string {
	title := r.Title()
	if len(title) > 40 {
		return title[:37] + "..."
	}
	return title
}

// Severity returns the severity
func (r *FindingResource) Severity() float64 {
	if r.Finding.Severity != nil {
		return *r.Finding.Severity
	}
	return 0
}

// SeverityLabel returns a human-readable severity label
func (r *FindingResource) SeverityLabel() string {
	sev := r.Severity()
	switch {
	case sev >= 7.0:
		return "HIGH"
	case sev >= 4.0:
		return "MEDIUM"
	default:
		return "LOW"
	}
}

// Type returns the finding type
func (r *FindingResource) Type() string {
	return appaws.Str(r.Finding.Type)
}

// Description returns the description
func (r *FindingResource) Description() string {
	return appaws.Str(r.Finding.Description)
}

// ResourceType returns the affected resource type
func (r *FindingResource) ResourceType() string {
	if r.Finding.Resource != nil {
		return appaws.Str(r.Finding.Resource.ResourceType)
	}
	return ""
}

// Region returns the region
func (r *FindingResource) Region() string {
	return appaws.Str(r.Finding.Region)
}

// AccountId returns the account ID
func (r *FindingResource) AccountId() string {
	return appaws.Str(r.Finding.AccountId)
}

// CreatedAt returns the creation time
func (r *FindingResource) CreatedAt() string {
	return appaws.Str(r.Finding.CreatedAt)
}

// UpdatedAt returns the update time
func (r *FindingResource) UpdatedAt() string {
	return appaws.Str(r.Finding.UpdatedAt)
}

// CreatedAtTime returns the creation time as time.Time
func (r *FindingResource) CreatedAtTime() *time.Time {
	if r.Finding.CreatedAt != nil {
		t, err := time.Parse(time.RFC3339, *r.Finding.CreatedAt)
		if err != nil {
			return nil
		}
		return &t
	}
	return nil
}
