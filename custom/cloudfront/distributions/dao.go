package distributions

import (
	"context"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/cloudfront"
	"github.com/aws/aws-sdk-go-v2/service/cloudfront/types"

	appaws "github.com/clawscli/claws/internal/aws"
	"github.com/clawscli/claws/internal/dao"
)

// DistributionDAO provides data access for CloudFront distributions
type DistributionDAO struct {
	dao.BaseDAO
	client *cloudfront.Client
}

// NewDistributionDAO creates a new DistributionDAO
func NewDistributionDAO(ctx context.Context) (dao.DAO, error) {
	cfg, err := appaws.NewConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("new cloudfront/distributions dao: %w", err)
	}
	return &DistributionDAO{
		BaseDAO: dao.NewBaseDAO("cloudfront", "distributions"),
		client:  cloudfront.NewFromConfig(cfg),
	}, nil
}

// List returns all CloudFront distributions
func (d *DistributionDAO) List(ctx context.Context) ([]dao.Resource, error) {
	distributions, err := appaws.Paginate(ctx, func(token *string) ([]types.DistributionSummary, *string, error) {
		output, err := d.client.ListDistributions(ctx, &cloudfront.ListDistributionsInput{
			Marker:   token,
			MaxItems: appaws.Int32Ptr(100),
		})
		if err != nil {
			return nil, nil, fmt.Errorf("list distributions: %w", err)
		}
		if output.DistributionList == nil {
			return nil, nil, nil
		}
		// CloudFront uses IsTruncated flag
		var nextToken *string
		if output.DistributionList.IsTruncated != nil && *output.DistributionList.IsTruncated {
			nextToken = output.DistributionList.NextMarker
		}
		return output.DistributionList.Items, nextToken, nil
	})
	if err != nil {
		return nil, err
	}

	resources := make([]dao.Resource, len(distributions))
	for i, item := range distributions {
		resources[i] = NewDistributionResource(item)
	}

	return resources, nil
}

// Get returns a specific CloudFront distribution by ID
func (d *DistributionDAO) Get(ctx context.Context, id string) (dao.Resource, error) {
	input := &cloudfront.GetDistributionInput{
		Id: &id,
	}

	output, err := d.client.GetDistribution(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("get distribution %s: %w", id, err)
	}

	// Convert Distribution to DistributionSummary for consistency
	dist := output.Distribution
	summary := types.DistributionSummary{
		Id:               dist.Id,
		ARN:              dist.ARN,
		Status:           dist.Status,
		DomainName:       dist.DomainName,
		LastModifiedTime: dist.LastModifiedTime,
	}
	if dist.DistributionConfig != nil {
		summary.Enabled = dist.DistributionConfig.Enabled
		summary.PriceClass = dist.DistributionConfig.PriceClass
		summary.HttpVersion = dist.DistributionConfig.HttpVersion
		summary.Origins = dist.DistributionConfig.Origins
		summary.Aliases = dist.DistributionConfig.Aliases
		summary.Comment = dist.DistributionConfig.Comment
		summary.DefaultCacheBehavior = dist.DistributionConfig.DefaultCacheBehavior
		summary.WebACLId = dist.DistributionConfig.WebACLId
	}

	res := NewDistributionResource(summary)
	// Store the invalidation batches from the full distribution
	if dist.InProgressInvalidationBatches != nil {
		res.InProgressInvalidations = *dist.InProgressInvalidationBatches
	}

	// Store additional config fields
	if dist.DistributionConfig != nil {
		cfg := dist.DistributionConfig
		res.ViewerCertificate = cfg.ViewerCertificate
		res.Logging = cfg.Logging
		if cfg.DefaultRootObject != nil {
			res.DefaultRootObject = *cfg.DefaultRootObject
		}
		if cfg.Restrictions != nil && cfg.Restrictions.GeoRestriction != nil {
			res.GeoRestriction = cfg.Restrictions.GeoRestriction
		}
		if cfg.CacheBehaviors != nil && cfg.CacheBehaviors.Quantity != nil {
			res.CacheBehaviorCount = int(*cfg.CacheBehaviors.Quantity)
		}
		if cfg.CustomErrorResponses != nil && cfg.CustomErrorResponses.Quantity != nil {
			res.CustomErrorResponses = int(*cfg.CustomErrorResponses.Quantity)
		}
		if cfg.IsIPV6Enabled != nil {
			res.IsIPV6Enabled = *cfg.IsIPV6Enabled
		}
	}

	return res, nil
}

// Delete deletes a CloudFront distribution (requires disabling first)
func (d *DistributionDAO) Delete(ctx context.Context, id string) error {
	// Get the distribution to get ETag
	getInput := &cloudfront.GetDistributionInput{
		Id: &id,
	}
	getOutput, err := d.client.GetDistribution(ctx, getInput)
	if err != nil {
		return fmt.Errorf("get distribution %s for deletion: %w", id, err)
	}

	input := &cloudfront.DeleteDistributionInput{
		Id:      &id,
		IfMatch: getOutput.ETag,
	}

	_, err = d.client.DeleteDistribution(ctx, input)
	if err != nil {
		return fmt.Errorf("delete distribution %s: %w", id, err)
	}

	return nil
}

// DistributionResource represents a CloudFront distribution
type DistributionResource struct {
	dao.BaseResource
	Item                    types.DistributionSummary
	InProgressInvalidations int32 // Stored separately as not in DistributionSummary
	// Extended fields from full Distribution
	ViewerCertificate    *types.ViewerCertificate
	Logging              *types.LoggingConfig
	DefaultRootObject    string
	GeoRestriction       *types.GeoRestriction
	CacheBehaviorCount   int
	CustomErrorResponses int
	IsIPV6Enabled        bool
}

// NewDistributionResource creates a new DistributionResource
func NewDistributionResource(item types.DistributionSummary) *DistributionResource {
	distId := appaws.Str(item.Id)
	arn := appaws.Str(item.ARN)

	return &DistributionResource{
		BaseResource: dao.BaseResource{
			ID:   distId,
			Name: distId,
			ARN:  arn,
			Tags: make(map[string]string),
			Data: item,
		},
		Item: item,
	}
}

// DistributionId returns the distribution ID
func (r *DistributionResource) DistributionId() string {
	return appaws.Str(r.Item.Id)
}

// DomainName returns the CloudFront domain name
func (r *DistributionResource) DomainName() string {
	return appaws.Str(r.Item.DomainName)
}

// Status returns the distribution status
func (r *DistributionResource) Status() string {
	return appaws.Str(r.Item.Status)
}

// Enabled returns whether the distribution is enabled
func (r *DistributionResource) Enabled() bool {
	if r.Item.Enabled != nil {
		return *r.Item.Enabled
	}
	return false
}

// Comment returns the distribution comment/description
func (r *DistributionResource) Comment() string {
	return appaws.Str(r.Item.Comment)
}

// PriceClass returns the price class
func (r *DistributionResource) PriceClass() string {
	return string(r.Item.PriceClass)
}

// HttpVersion returns the HTTP version
func (r *DistributionResource) HttpVersion() string {
	return string(r.Item.HttpVersion)
}

// Aliases returns the CNAMEs (alternate domain names)
func (r *DistributionResource) Aliases() []string {
	if r.Item.Aliases != nil {
		return r.Item.Aliases.Items
	}
	return nil
}

// AliasCount returns the number of aliases
func (r *DistributionResource) AliasCount() int {
	if r.Item.Aliases != nil && r.Item.Aliases.Quantity != nil {
		return int(*r.Item.Aliases.Quantity)
	}
	return 0
}

// Origins returns a summary of origins
func (r *DistributionResource) Origins() []string {
	if r.Item.Origins == nil {
		return nil
	}
	var origins []string
	for _, origin := range r.Item.Origins.Items {
		if origin.DomainName != nil {
			origins = append(origins, *origin.DomainName)
		}
	}
	return origins
}

// OriginCount returns the number of origins
func (r *DistributionResource) OriginCount() int {
	if r.Item.Origins != nil && r.Item.Origins.Quantity != nil {
		return int(*r.Item.Origins.Quantity)
	}
	return 0
}

// DefaultOrigin returns the first/default origin domain
func (r *DistributionResource) DefaultOrigin() string {
	if r.Item.Origins != nil && len(r.Item.Origins.Items) > 0 {
		return appaws.Str(r.Item.Origins.Items[0].DomainName)
	}
	return ""
}

// OriginType returns the type of the default origin
func (r *DistributionResource) OriginType() string {
	origin := r.DefaultOrigin()
	if origin == "" {
		return ""
	}
	if strings.Contains(origin, ".s3.") || strings.HasSuffix(origin, ".s3.amazonaws.com") {
		return "S3"
	}
	if strings.Contains(origin, "elb.") || strings.Contains(origin, "elasticloadbalancing.") {
		return "ALB/ELB"
	}
	if strings.Contains(origin, "execute-api.") {
		return "API Gateway"
	}
	return "Custom"
}

// WebACLId returns the WAF WebACL ID
func (r *DistributionResource) WebACLId() string {
	return appaws.Str(r.Item.WebACLId)
}

// DefaultCacheBehavior returns information about the default cache behavior
func (r *DistributionResource) DefaultCacheBehaviorViewerProtocolPolicy() string {
	if r.Item.DefaultCacheBehavior != nil {
		return string(r.Item.DefaultCacheBehavior.ViewerProtocolPolicy)
	}
	return ""
}

// LastModifiedTime returns the last modified time
func (r *DistributionResource) LastModifiedTime() string {
	if r.Item.LastModifiedTime != nil {
		return r.Item.LastModifiedTime.Format("2006-01-02 15:04:05")
	}
	return ""
}

// InProgressInvalidationBatches returns the number of in-progress invalidation batches
func (r *DistributionResource) InProgressInvalidationBatches() int32 {
	return r.InProgressInvalidations
}
