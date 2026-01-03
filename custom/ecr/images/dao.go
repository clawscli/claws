package images

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/ecr"
	"github.com/aws/aws-sdk-go-v2/service/ecr/types"

	appaws "github.com/clawscli/claws/internal/aws"
	"github.com/clawscli/claws/internal/dao"
	apperrors "github.com/clawscli/claws/internal/errors"
)

// ImageDAO provides data access for ECR images
type ImageDAO struct {
	dao.BaseDAO
	client *ecr.Client
}

// NewImageDAO creates a new ImageDAO
func NewImageDAO(ctx context.Context) (dao.DAO, error) {
	cfg, err := appaws.NewConfig(ctx)
	if err != nil {
		return nil, apperrors.Wrap(err, "new "+ServiceResourcePath+" dao")
	}
	return &ImageDAO{
		BaseDAO: dao.NewBaseDAO("ecr", "images"),
		client:  ecr.NewFromConfig(cfg),
	}, nil
}

// List returns images (first page only for backwards compatibility).
// For paginated access, use ListPage instead.
func (d *ImageDAO) List(ctx context.Context) ([]dao.Resource, error) {
	resources, _, err := d.ListPage(ctx, 100, "")
	return resources, err
}

// ListPage returns a page of ECR images.
// Implements dao.PaginatedDAO interface.
func (d *ImageDAO) ListPage(ctx context.Context, pageSize int, pageToken string) ([]dao.Resource, string, error) {
	// Get repository name from filter context
	repoName := dao.GetFilterFromContext(ctx, "RepositoryName")
	if repoName == "" {
		return nil, "", fmt.Errorf("repository name filter required")
	}

	maxResults := int32(pageSize)
	if maxResults > 1000 {
		maxResults = 1000 // AWS API max
	}

	input := &ecr.DescribeImagesInput{
		RepositoryName: &repoName,
		MaxResults:     &maxResults,
	}
	if pageToken != "" {
		input.NextToken = &pageToken
	}

	output, err := d.client.DescribeImages(ctx, input)
	if err != nil {
		return nil, "", apperrors.Wrap(err, "describe images")
	}

	resources := make([]dao.Resource, len(output.ImageDetails))
	for i, img := range output.ImageDetails {
		resources[i] = NewImageResource(img, repoName)
	}

	nextToken := ""
	if output.NextToken != nil {
		nextToken = *output.NextToken
	}

	return resources, nextToken, nil
}

// Get returns a specific image
func (d *ImageDAO) Get(ctx context.Context, imageDigest string) (dao.Resource, error) {
	repoName := dao.GetFilterFromContext(ctx, "RepositoryName")
	if repoName == "" {
		return nil, fmt.Errorf("repository name filter required")
	}

	input := &ecr.DescribeImagesInput{
		RepositoryName: &repoName,
		ImageIds: []types.ImageIdentifier{
			{ImageDigest: &imageDigest},
		},
	}

	output, err := d.client.DescribeImages(ctx, input)
	if err != nil {
		return nil, apperrors.Wrapf(err, "describe image %s", imageDigest)
	}

	if len(output.ImageDetails) == 0 {
		return nil, fmt.Errorf("image not found: %s", imageDigest)
	}

	return NewImageResource(output.ImageDetails[0], repoName), nil
}

// Delete deletes an image
func (d *ImageDAO) Delete(ctx context.Context, imageDigest string) error {
	repoName := dao.GetFilterFromContext(ctx, "RepositoryName")
	if repoName == "" {
		return fmt.Errorf("repository name filter required")
	}

	input := &ecr.BatchDeleteImageInput{
		RepositoryName: &repoName,
		ImageIds: []types.ImageIdentifier{
			{ImageDigest: &imageDigest},
		},
	}

	_, err := d.client.BatchDeleteImage(ctx, input)
	if err != nil {
		return apperrors.Wrapf(err, "delete image %s", imageDigest)
	}

	return nil
}

// Supports returns supported operations
func (d *ImageDAO) Supports(op dao.Operation) bool {
	switch op {
	case dao.OpList, dao.OpGet, dao.OpDelete:
		return true
	default:
		return false
	}
}

// ImageResource represents an ECR image
type ImageResource struct {
	dao.BaseResource
	Image          types.ImageDetail
	RepositoryName string
}

// NewImageResource creates a new ImageResource
func NewImageResource(img types.ImageDetail, repoName string) *ImageResource {
	digest := appaws.Str(img.ImageDigest)
	// Use first tag as name if available, otherwise use digest
	name := digest
	if len(img.ImageTags) > 0 {
		name = img.ImageTags[0]
	}

	return &ImageResource{
		BaseResource: dao.BaseResource{
			ID:   digest,
			Name: name,
			ARN:  "",
			Tags: make(map[string]string),
			Data: img,
		},
		Image:          img,
		RepositoryName: repoName,
	}
}

// ImageDigest returns the image digest
func (r *ImageResource) ImageDigest() string {
	return appaws.Str(r.Image.ImageDigest)
}

// ImageTags returns the image tags
func (r *ImageResource) ImageTags() []string {
	return r.Image.ImageTags
}

// FirstTag returns the first tag or empty string
func (r *ImageResource) FirstTag() string {
	if len(r.Image.ImageTags) > 0 {
		return r.Image.ImageTags[0]
	}
	return ""
}

// TagsFormatted returns tags as comma-separated string
func (r *ImageResource) TagsFormatted() string {
	if len(r.Image.ImageTags) == 0 {
		return "<untagged>"
	}
	if len(r.Image.ImageTags) == 1 {
		return r.Image.ImageTags[0]
	}
	// Show first tag + count
	return fmt.Sprintf("%s (+%d)", r.Image.ImageTags[0], len(r.Image.ImageTags)-1)
}

// ImageSizeInBytes returns the image size
func (r *ImageResource) ImageSizeInBytes() int64 {
	if r.Image.ImageSizeInBytes != nil {
		return *r.Image.ImageSizeInBytes
	}
	return 0
}

// ImageSizeFormatted returns the image size formatted
func (r *ImageResource) ImageSizeFormatted() string {
	bytes := r.ImageSizeInBytes()
	if bytes == 0 {
		return "-"
	}
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
	)
	switch {
	case bytes >= GB:
		return fmt.Sprintf("%.2f GB", float64(bytes)/GB)
	case bytes >= MB:
		return fmt.Sprintf("%.2f MB", float64(bytes)/MB)
	case bytes >= KB:
		return fmt.Sprintf("%.2f KB", float64(bytes)/KB)
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}

// PushedAt returns the push timestamp
func (r *ImageResource) PushedAt() string {
	if r.Image.ImagePushedAt != nil {
		return r.Image.ImagePushedAt.Format("2006-01-02 15:04:05")
	}
	return ""
}

// PushedAtTime returns the push timestamp as time.Time
func (r *ImageResource) PushedAtTime() *time.Time {
	return r.Image.ImagePushedAt
}

// ScanStatus returns the scan status
func (r *ImageResource) ScanStatus() string {
	if r.Image.ImageScanStatus != nil {
		return string(r.Image.ImageScanStatus.Status)
	}
	return ""
}

// ScanFindingsCount returns the number of findings
func (r *ImageResource) ScanFindingsCount() int {
	if r.Image.ImageScanFindingsSummary != nil && r.Image.ImageScanFindingsSummary.FindingSeverityCounts != nil {
		total := 0
		for _, count := range r.Image.ImageScanFindingsSummary.FindingSeverityCounts {
			total += int(count)
		}
		return total
	}
	return 0
}

// ArtifactMediaType returns the artifact media type
func (r *ImageResource) ArtifactMediaType() string {
	if r.Image.ArtifactMediaType != nil {
		return *r.Image.ArtifactMediaType
	}
	return ""
}

// ImageManifestMediaType returns the manifest media type
func (r *ImageResource) ImageManifestMediaType() string {
	if r.Image.ImageManifestMediaType != nil {
		return *r.Image.ImageManifestMediaType
	}
	return ""
}
