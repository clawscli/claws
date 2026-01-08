package crawlers

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/glue"
	"github.com/aws/aws-sdk-go-v2/service/glue/types"

	appaws "github.com/clawscli/claws/internal/aws"
	"github.com/clawscli/claws/internal/dao"
	apperrors "github.com/clawscli/claws/internal/errors"
)

// CrawlerDAO provides data access for Glue crawlers.
type CrawlerDAO struct {
	dao.BaseDAO
	client *glue.Client
}

// NewCrawlerDAO creates a new CrawlerDAO.
func NewCrawlerDAO(ctx context.Context) (dao.DAO, error) {
	cfg, err := appaws.NewConfig(ctx)
	if err != nil {
		return nil, apperrors.Wrap(err, "new "+ServiceResourcePath+" dao")
	}
	return &CrawlerDAO{
		BaseDAO: dao.NewBaseDAO("glue", "crawlers"),
		client:  glue.NewFromConfig(cfg),
	}, nil
}

// List returns all Glue crawlers.
func (d *CrawlerDAO) List(ctx context.Context) ([]dao.Resource, error) {
	crawlers, err := appaws.Paginate(ctx, func(token *string) ([]types.Crawler, *string, error) {
		output, err := d.client.GetCrawlers(ctx, &glue.GetCrawlersInput{
			NextToken: token,
		})
		if err != nil {
			return nil, nil, apperrors.Wrap(err, "get glue crawlers")
		}
		return output.Crawlers, output.NextToken, nil
	})
	if err != nil {
		return nil, err
	}

	resources := make([]dao.Resource, len(crawlers))
	for i, crawler := range crawlers {
		resources[i] = NewCrawlerResource(crawler)
	}
	return resources, nil
}

// Get returns a specific Glue crawler by name.
func (d *CrawlerDAO) Get(ctx context.Context, id string) (dao.Resource, error) {
	output, err := d.client.GetCrawler(ctx, &glue.GetCrawlerInput{
		Name: &id,
	})
	if err != nil {
		return nil, apperrors.Wrapf(err, "get glue crawler %s", id)
	}
	return NewCrawlerResource(*output.Crawler), nil
}

// Delete deletes a Glue crawler by name.
func (d *CrawlerDAO) Delete(ctx context.Context, id string) error {
	_, err := d.client.DeleteCrawler(ctx, &glue.DeleteCrawlerInput{
		Name: &id,
	})
	if err != nil {
		return apperrors.Wrapf(err, "delete glue crawler %s", id)
	}
	return nil
}

// CrawlerResource wraps a Glue crawler.
type CrawlerResource struct {
	dao.BaseResource
	Item types.Crawler
}

// NewCrawlerResource creates a new CrawlerResource.
func NewCrawlerResource(crawler types.Crawler) *CrawlerResource {
	return &CrawlerResource{
		BaseResource: dao.BaseResource{
			ID:   appaws.Str(crawler.Name),
			ARN:  "",
			Data: crawler,
		},
		Item: crawler,
	}
}

// Name returns the crawler name.
func (r *CrawlerResource) Name() string {
	return appaws.Str(r.Item.Name)
}

// State returns the crawler state.
func (r *CrawlerResource) State() string {
	return string(r.Item.State)
}

// DatabaseName returns the target database name.
func (r *CrawlerResource) DatabaseName() string {
	return appaws.Str(r.Item.DatabaseName)
}

// Description returns the crawler description.
func (r *CrawlerResource) Description() string {
	return appaws.Str(r.Item.Description)
}

// Role returns the IAM role.
func (r *CrawlerResource) Role() string {
	return appaws.Str(r.Item.Role)
}

// Schedule returns the crawler schedule.
func (r *CrawlerResource) Schedule() string {
	if r.Item.Schedule != nil {
		return appaws.Str(r.Item.Schedule.ScheduleExpression)
	}
	return ""
}

// LastCrawlStatus returns the last crawl status.
func (r *CrawlerResource) LastCrawlStatus() string {
	if r.Item.LastCrawl != nil {
		return string(r.Item.LastCrawl.Status)
	}
	return ""
}

// LastCrawlTime returns when the last crawl started.
func (r *CrawlerResource) LastCrawlTime() *time.Time {
	if r.Item.LastCrawl != nil {
		return r.Item.LastCrawl.StartTime
	}
	return nil
}

// CreationTime returns when the crawler was created.
func (r *CrawlerResource) CreationTime() *time.Time {
	return r.Item.CreationTime
}

// LastUpdated returns when the crawler was last updated.
func (r *CrawlerResource) LastUpdated() *time.Time {
	return r.Item.LastUpdated
}

// TablePrefix returns the table prefix.
func (r *CrawlerResource) TablePrefix() string {
	return appaws.Str(r.Item.TablePrefix)
}
