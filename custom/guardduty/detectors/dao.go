package detectors

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/guardduty"
	"github.com/aws/aws-sdk-go-v2/service/guardduty/types"

	appaws "github.com/clawscli/claws/internal/aws"
	"github.com/clawscli/claws/internal/dao"
	apperrors "github.com/clawscli/claws/internal/errors"
)

// DetectorDAO provides data access for GuardDuty detectors
type DetectorDAO struct {
	dao.BaseDAO
	client *guardduty.Client
}

// NewDetectorDAO creates a new DetectorDAO
func NewDetectorDAO(ctx context.Context) (dao.DAO, error) {
	cfg, err := appaws.NewConfig(ctx)
	if err != nil {
		return nil, apperrors.Wrap(err, "new "+ServiceResourcePath+" dao")
	}
	return &DetectorDAO{
		BaseDAO: dao.NewBaseDAO("guardduty", "detectors"),
		client:  guardduty.NewFromConfig(cfg),
	}, nil
}

// List returns all GuardDuty detectors
func (d *DetectorDAO) List(ctx context.Context) ([]dao.Resource, error) {
	var resources []dao.Resource

	idIter := appaws.PaginateIter(ctx, func(token *string) ([]string, *string, error) {
		output, err := d.client.ListDetectors(ctx, &guardduty.ListDetectorsInput{
			NextToken: token,
		})
		if err != nil {
			return nil, nil, apperrors.Wrap(err, "list detectors")
		}
		return output.DetectorIds, output.NextToken, nil
	})

	for detectorId, err := range idIter {
		if err != nil {
			return nil, err
		}
		detail, err := d.client.GetDetector(ctx, &guardduty.GetDetectorInput{
			DetectorId: &detectorId,
		})
		if err != nil {
			return nil, apperrors.Wrapf(err, "get detector %s", detectorId)
		}
		resources = append(resources, NewDetectorResource(detectorId, detail))
	}

	return resources, nil
}

// Get returns a specific GuardDuty detector by ID
func (d *DetectorDAO) Get(ctx context.Context, id string) (dao.Resource, error) {
	output, err := d.client.GetDetector(ctx, &guardduty.GetDetectorInput{
		DetectorId: &id,
	})
	if err != nil {
		return nil, apperrors.Wrapf(err, "get detector %s", id)
	}

	return NewDetectorResource(id, output), nil
}

// Delete deletes a GuardDuty detector
func (d *DetectorDAO) Delete(ctx context.Context, id string) error {
	_, err := d.client.DeleteDetector(ctx, &guardduty.DeleteDetectorInput{
		DetectorId: &id,
	})
	if err != nil {
		return apperrors.Wrapf(err, "delete detector %s", id)
	}
	return nil
}

// DetectorResource represents a GuardDuty detector
type DetectorResource struct {
	dao.BaseResource
	DetectorId string
	Detail     *guardduty.GetDetectorOutput
}

// NewDetectorResource creates a new DetectorResource
func NewDetectorResource(detectorId string, detail *guardduty.GetDetectorOutput) *DetectorResource {
	return &DetectorResource{
		BaseResource: dao.BaseResource{
			ID:   detectorId,
			Name: detectorId,
			ARN:  "", // ARN not directly available
			Tags: convertTags(detail.Tags),
			Data: detail,
		},
		DetectorId: detectorId,
		Detail:     detail,
	}
}

func convertTags(tags map[string]string) map[string]string {
	if tags == nil {
		return make(map[string]string)
	}
	return tags
}

// Status returns the detector status
func (r *DetectorResource) Status() string {
	if r.Detail != nil {
		return string(r.Detail.Status)
	}
	return ""
}

// FindingPublishingFrequency returns the finding publishing frequency
func (r *DetectorResource) FindingPublishingFrequency() string {
	if r.Detail != nil {
		return string(r.Detail.FindingPublishingFrequency)
	}
	return ""
}

// ServiceRole returns the service role ARN
func (r *DetectorResource) ServiceRole() string {
	if r.Detail != nil {
		return appaws.Str(r.Detail.ServiceRole)
	}
	return ""
}

// CreatedAt returns the creation date
func (r *DetectorResource) CreatedAt() string {
	if r.Detail != nil {
		return appaws.Str(r.Detail.CreatedAt)
	}
	return ""
}

// CreatedAtTime returns the creation date as time.Time
func (r *DetectorResource) CreatedAtTime() *time.Time {
	if r.Detail != nil && r.Detail.CreatedAt != nil {
		// GuardDuty CreatedAt is a string in ISO format
		t, err := time.Parse(time.RFC3339, *r.Detail.CreatedAt)
		if err != nil {
			return nil
		}
		return &t
	}
	return nil
}

// UpdatedAt returns the last updated date
func (r *DetectorResource) UpdatedAt() string {
	if r.Detail != nil {
		return appaws.Str(r.Detail.UpdatedAt)
	}
	return ""
}

// FeaturesStatus returns status of features
func (r *DetectorResource) FeaturesStatus() map[string]types.FeatureStatus {
	result := make(map[string]types.FeatureStatus)
	if r.Detail == nil {
		return result
	}

	for _, feature := range r.Detail.Features {
		if feature.Name != "" {
			result[string(feature.Name)] = feature.Status
		}
	}

	return result
}
