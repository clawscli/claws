package crawlers

import (
	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/render"
)

// CrawlerRenderer renders Glue crawlers.
type CrawlerRenderer struct {
	render.BaseRenderer
}

// NewCrawlerRenderer creates a new CrawlerRenderer.
func NewCrawlerRenderer() render.Renderer {
	return &CrawlerRenderer{
		BaseRenderer: render.BaseRenderer{
			Service:  "glue",
			Resource: "crawlers",
			Cols: []render.Column{
				{Name: "CRAWLER NAME", Width: 35, Getter: func(r dao.Resource) string { return r.GetID() }},
				{Name: "STATE", Width: 12, Getter: getState},
				{Name: "DATABASE", Width: 25, Getter: getDatabase},
				{Name: "LAST STATUS", Width: 12, Getter: getLastStatus},
				{Name: "LAST RUN", Width: 20, Getter: getLastRun},
			},
		},
	}
}

func getState(r dao.Resource) string {
	crawler, ok := r.(*CrawlerResource)
	if !ok {
		return ""
	}
	return crawler.State()
}

func getDatabase(r dao.Resource) string {
	crawler, ok := r.(*CrawlerResource)
	if !ok {
		return ""
	}
	return crawler.DatabaseName()
}

func getLastStatus(r dao.Resource) string {
	crawler, ok := r.(*CrawlerResource)
	if !ok {
		return ""
	}
	return crawler.LastCrawlStatus()
}

func getLastRun(r dao.Resource) string {
	crawler, ok := r.(*CrawlerResource)
	if !ok {
		return ""
	}
	if t := crawler.LastCrawlTime(); t != nil {
		return t.Format("2006-01-02 15:04")
	}
	return ""
}

// RenderDetail renders the detail view for a Glue crawler.
func (r *CrawlerRenderer) RenderDetail(resource dao.Resource) string {
	crawler, ok := resource.(*CrawlerResource)
	if !ok {
		return ""
	}

	d := render.NewDetailBuilder()

	d.Title("Glue Crawler", crawler.Name())

	// Basic Info
	d.Section("Basic Information")
	d.Field("Crawler Name", crawler.Name())
	d.Field("State", crawler.State())
	if desc := crawler.Description(); desc != "" {
		d.Field("Description", desc)
	}

	// Configuration
	d.Section("Configuration")
	d.Field("Target Database", crawler.DatabaseName())
	d.Field("IAM Role", crawler.Role())
	if prefix := crawler.TablePrefix(); prefix != "" {
		d.Field("Table Prefix", prefix)
	}

	// Schedule
	if schedule := crawler.Schedule(); schedule != "" {
		d.Section("Schedule")
		d.Field("Expression", schedule)
	}

	// Last Crawl
	if status := crawler.LastCrawlStatus(); status != "" {
		d.Section("Last Crawl")
		d.Field("Status", status)
		if t := crawler.LastCrawlTime(); t != nil {
			d.Field("Start Time", t.Format("2006-01-02 15:04:05"))
		}
	}

	// Timestamps
	d.Section("Timestamps")
	if t := crawler.CreationTime(); t != nil {
		d.Field("Created", t.Format("2006-01-02 15:04:05"))
	}
	if t := crawler.LastUpdated(); t != nil {
		d.Field("Last Updated", t.Format("2006-01-02 15:04:05"))
	}

	return d.String()
}

// RenderSummary renders summary fields for a Glue crawler.
func (r *CrawlerRenderer) RenderSummary(resource dao.Resource) []render.SummaryField {
	crawler, ok := resource.(*CrawlerResource)
	if !ok {
		return r.BaseRenderer.RenderSummary(resource)
	}

	fields := []render.SummaryField{
		{Label: "Crawler Name", Value: crawler.Name()},
		{Label: "State", Value: crawler.State()},
		{Label: "Database", Value: crawler.DatabaseName()},
	}

	if status := crawler.LastCrawlStatus(); status != "" {
		fields = append(fields, render.SummaryField{Label: "Last Status", Value: status})
	}

	return fields
}
