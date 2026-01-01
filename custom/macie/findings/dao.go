package findings

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/macie2"
	"github.com/aws/aws-sdk-go-v2/service/macie2/types"

	appaws "github.com/clawscli/claws/internal/aws"
	"github.com/clawscli/claws/internal/dao"
	apperrors "github.com/clawscli/claws/internal/errors"
)

// FindingDAO provides data access for Macie findings.
type FindingDAO struct {
	dao.BaseDAO
	client *macie2.Client
}

// NewFindingDAO creates a new FindingDAO.
func NewFindingDAO(ctx context.Context) (dao.DAO, error) {
	cfg, err := appaws.NewConfig(ctx)
	if err != nil {
		return nil, apperrors.Wrap(err, "new macie/findings dao")
	}
	return &FindingDAO{
		BaseDAO: dao.NewBaseDAO("macie2", "findings"),
		client:  macie2.NewFromConfig(cfg),
	}, nil
}

// List returns all findings.
func (d *FindingDAO) List(ctx context.Context) ([]dao.Resource, error) {
	bucketName := dao.GetFilterFromContext(ctx, "BucketName")

	// First list finding IDs
	findingIds, err := appaws.Paginate(ctx, func(token *string) ([]string, *string, error) {
		input := &macie2.ListFindingsInput{
			NextToken: token,
		}

		output, err := d.client.ListFindings(ctx, input)
		if err != nil {
			return nil, nil, apperrors.Wrap(err, "list macie findings")
		}
		return output.FindingIds, output.NextToken, nil
	})
	if err != nil {
		return nil, err
	}

	if len(findingIds) == 0 {
		return nil, nil
	}

	// Get findings in batches
	resources := make([]dao.Resource, 0, len(findingIds))
	for i := 0; i < len(findingIds); i += 50 {
		end := i + 50
		if end > len(findingIds) {
			end = len(findingIds)
		}
		batch := findingIds[i:end]

		output, err := d.client.GetFindings(ctx, &macie2.GetFindingsInput{
			FindingIds: batch,
		})
		if err != nil {
			return nil, apperrors.Wrap(err, "get macie findings")
		}

		for _, finding := range output.Findings {
			// Client-side filter by bucket name if specified
			if bucketName != "" {
				if finding.ResourcesAffected != nil &&
					finding.ResourcesAffected.S3Bucket != nil &&
					finding.ResourcesAffected.S3Bucket.Name != nil &&
					*finding.ResourcesAffected.S3Bucket.Name == bucketName {
					resources = append(resources, NewFindingResource(finding))
				}
			} else {
				resources = append(resources, NewFindingResource(finding))
			}
		}
	}

	return resources, nil
}

// Get returns a specific finding.
func (d *FindingDAO) Get(ctx context.Context, id string) (dao.Resource, error) {
	output, err := d.client.GetFindings(ctx, &macie2.GetFindingsInput{
		FindingIds: []string{id},
	})
	if err != nil {
		return nil, apperrors.Wrap(err, "get macie finding")
	}
	if len(output.Findings) == 0 {
		return nil, fmt.Errorf("finding not found: %s", id)
	}
	return NewFindingResource(output.Findings[0]), nil
}

// Delete is not supported for findings (read-only).
func (d *FindingDAO) Delete(ctx context.Context, id string) error {
	return fmt.Errorf("delete not supported for macie findings")
}

// FindingResource wraps a Macie finding.
type FindingResource struct {
	dao.BaseResource
	Finding *types.Finding
}

// NewFindingResource creates a new FindingResource.
func NewFindingResource(finding types.Finding) *FindingResource {
	return &FindingResource{
		BaseResource: dao.BaseResource{
			ID:  appaws.Str(finding.Id),
			ARN: "",
		},
		Finding: &finding,
	}
}

// Title returns the finding title.
func (r *FindingResource) Title() string {
	if r.Finding != nil && r.Finding.Title != nil {
		return *r.Finding.Title
	}
	return ""
}

// Description returns the finding description.
func (r *FindingResource) Description() string {
	if r.Finding != nil && r.Finding.Description != nil {
		return *r.Finding.Description
	}
	return ""
}

// Severity returns the finding severity.
func (r *FindingResource) Severity() string {
	if r.Finding != nil && r.Finding.Severity != nil {
		return string(r.Finding.Severity.Description)
	}
	return ""
}

// SeverityScore returns the finding severity score.
func (r *FindingResource) SeverityScore() int64 {
	if r.Finding != nil && r.Finding.Severity != nil && r.Finding.Severity.Score != nil {
		return *r.Finding.Severity.Score
	}
	return 0
}

// Type returns the finding type.
func (r *FindingResource) Type() string {
	if r.Finding != nil {
		return string(r.Finding.Type)
	}
	return ""
}

// Category returns the finding category.
func (r *FindingResource) Category() string {
	if r.Finding != nil {
		return string(r.Finding.Category)
	}
	return ""
}

// AccountId returns the account ID.
func (r *FindingResource) AccountId() string {
	if r.Finding != nil && r.Finding.AccountId != nil {
		return *r.Finding.AccountId
	}
	return ""
}

// Region returns the region.
func (r *FindingResource) Region() string {
	if r.Finding != nil && r.Finding.Region != nil {
		return *r.Finding.Region
	}
	return ""
}

// Count returns the occurrence count.
func (r *FindingResource) Count() int64 {
	if r.Finding != nil && r.Finding.Count != nil {
		return *r.Finding.Count
	}
	return 0
}

// Archived returns whether the finding is archived.
func (r *FindingResource) Archived() bool {
	if r.Finding != nil && r.Finding.Archived != nil {
		return *r.Finding.Archived
	}
	return false
}

// Sample returns whether the finding is a sample.
func (r *FindingResource) Sample() bool {
	if r.Finding != nil && r.Finding.Sample != nil {
		return *r.Finding.Sample
	}
	return false
}

// BucketName returns the affected S3 bucket name.
func (r *FindingResource) BucketName() string {
	if r.Finding != nil && r.Finding.ResourcesAffected != nil &&
		r.Finding.ResourcesAffected.S3Bucket != nil &&
		r.Finding.ResourcesAffected.S3Bucket.Name != nil {
		return *r.Finding.ResourcesAffected.S3Bucket.Name
	}
	return ""
}

// BucketArn returns the affected S3 bucket ARN.
func (r *FindingResource) BucketArn() string {
	if r.Finding != nil && r.Finding.ResourcesAffected != nil &&
		r.Finding.ResourcesAffected.S3Bucket != nil &&
		r.Finding.ResourcesAffected.S3Bucket.Arn != nil {
		return *r.Finding.ResourcesAffected.S3Bucket.Arn
	}
	return ""
}

// BucketOwner returns the affected S3 bucket owner.
func (r *FindingResource) BucketOwner() string {
	if r.Finding != nil && r.Finding.ResourcesAffected != nil &&
		r.Finding.ResourcesAffected.S3Bucket != nil &&
		r.Finding.ResourcesAffected.S3Bucket.Owner != nil &&
		r.Finding.ResourcesAffected.S3Bucket.Owner.DisplayName != nil {
		return *r.Finding.ResourcesAffected.S3Bucket.Owner.DisplayName
	}
	return ""
}

// ObjectKey returns the affected S3 object key.
func (r *FindingResource) ObjectKey() string {
	if r.Finding != nil && r.Finding.ResourcesAffected != nil &&
		r.Finding.ResourcesAffected.S3Object != nil &&
		r.Finding.ResourcesAffected.S3Object.Key != nil {
		return *r.Finding.ResourcesAffected.S3Object.Key
	}
	return ""
}

// ObjectSize returns the affected S3 object size.
func (r *FindingResource) ObjectSize() int64 {
	if r.Finding != nil && r.Finding.ResourcesAffected != nil &&
		r.Finding.ResourcesAffected.S3Object != nil &&
		r.Finding.ResourcesAffected.S3Object.Size != nil {
		return *r.Finding.ResourcesAffected.S3Object.Size
	}
	return 0
}

// ObjectPath returns the affected S3 object path.
func (r *FindingResource) ObjectPath() string {
	if r.Finding != nil && r.Finding.ResourcesAffected != nil &&
		r.Finding.ResourcesAffected.S3Object != nil &&
		r.Finding.ResourcesAffected.S3Object.Path != nil {
		return *r.Finding.ResourcesAffected.S3Object.Path
	}
	return ""
}

// ObjectStorageClass returns the affected S3 object storage class.
func (r *FindingResource) ObjectStorageClass() string {
	if r.Finding != nil && r.Finding.ResourcesAffected != nil &&
		r.Finding.ResourcesAffected.S3Object != nil {
		return string(r.Finding.ResourcesAffected.S3Object.StorageClass)
	}
	return ""
}

// CreatedAt returns when the finding was created.
func (r *FindingResource) CreatedAt() *time.Time {
	if r.Finding != nil {
		return r.Finding.CreatedAt
	}
	return nil
}

// UpdatedAt returns when the finding was updated.
func (r *FindingResource) UpdatedAt() *time.Time {
	if r.Finding != nil {
		return r.Finding.UpdatedAt
	}
	return nil
}
