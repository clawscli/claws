package analyzers

import (
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/accessanalyzer/types"
	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/render"
)

// AnalyzerRenderer renders Access Analyzer analyzers.
// Ensure AnalyzerRenderer implements render.Navigator
var _ render.Navigator = (*AnalyzerRenderer)(nil)

type AnalyzerRenderer struct {
	render.BaseRenderer
}

// NewAnalyzerRenderer creates a new AnalyzerRenderer.
func NewAnalyzerRenderer() render.Renderer {
	return &AnalyzerRenderer{
		BaseRenderer: render.BaseRenderer{
			Service:  "accessanalyzer",
			Resource: "analyzers",
			Cols: []render.Column{
				{Name: "NAME", Width: 30, Getter: func(r dao.Resource) string { return r.GetID() }},
				{Name: "TYPE", Width: 18, Getter: getType},
				{Name: "STATUS", Width: 12, Getter: getStatus},
				{Name: "CREATED", Width: 20, Getter: getCreated},
			},
		},
	}
}

func getType(r dao.Resource) string {
	analyzer, ok := r.(*AnalyzerResource)
	if !ok {
		return ""
	}
	return analyzer.Type()
}

func getStatus(r dao.Resource) string {
	analyzer, ok := r.(*AnalyzerResource)
	if !ok {
		return ""
	}
	return analyzer.Status()
}

func getCreated(r dao.Resource) string {
	analyzer, ok := r.(*AnalyzerResource)
	if !ok {
		return ""
	}
	if t := analyzer.CreatedAt(); t != nil {
		return render.FormatAge(*t)
	}
	return ""
}

// RenderDetail renders the detail view for an analyzer.
func (r *AnalyzerRenderer) RenderDetail(resource dao.Resource) string {
	analyzer, ok := resource.(*AnalyzerResource)
	if !ok {
		return ""
	}

	d := render.NewDetailBuilder()

	d.Title("IAM Access Analyzer", analyzer.Name())

	// Basic Info
	d.Section("Basic Information")
	d.Field("Name", analyzer.Name())
	d.Field("ARN", analyzer.GetARN())
	d.Field("Type", analyzer.Type())
	d.Field("Status", analyzer.Status())

	// Status Reason
	if reason := analyzer.StatusReason(); reason != "" {
		d.Field("Status Reason", reason)
	}

	// Configuration
	if cfg := analyzer.Configuration(); cfg != nil {
		d.Section("Configuration")
		// Use type switch to determine configuration type
		switch c := cfg.(type) {
		case *types.AnalyzerConfigurationMemberUnusedAccess:
			d.Field("Configuration Type", "Unused Access Analyzer")
			if c.Value.AnalysisRule != nil && len(c.Value.AnalysisRule.Exclusions) > 0 {
				d.Field("Exclusion Rules", fmt.Sprintf("%d rules", len(c.Value.AnalysisRule.Exclusions)))
			}
			if c.Value.UnusedAccessAge != nil {
				d.Field("Unused Access Age", fmt.Sprintf("%d days", *c.Value.UnusedAccessAge))
			}
		case *types.AnalyzerConfigurationMemberInternalAccess:
			d.Field("Configuration Type", "Internal Access Analyzer")
		default:
			d.Field("Configuration Type", "External Access Analyzer")
		}
	}

	// Analysis Info
	d.Section("Analysis")
	if lastResource := analyzer.LastResourceAnalyzed(); lastResource != "" {
		d.Field("Last Resource Analyzed", lastResource)
	}
	if t := analyzer.LastResourceAnalyzedAt(); t != nil {
		d.Field("Last Analyzed At", t.Format("2006-01-02 15:04:05"))
	}

	// Timestamps
	d.Section("Timestamps")
	if t := analyzer.CreatedAt(); t != nil {
		d.Field("Created", t.Format("2006-01-02 15:04:05"))
	}

	// Tags
	if tags := analyzer.Tags(); len(tags) > 0 {
		d.Section("Tags")
		for k, v := range tags {
			d.Field(k, v)
		}
	}

	return d.String()
}

// RenderSummary renders summary fields for an analyzer.
func (r *AnalyzerRenderer) RenderSummary(resource dao.Resource) []render.SummaryField {
	analyzer, ok := resource.(*AnalyzerResource)
	if !ok {
		return r.BaseRenderer.RenderSummary(resource)
	}

	return []render.SummaryField{
		{Label: "Name", Value: analyzer.Name()},
		{Label: "ARN", Value: analyzer.GetARN()},
		{Label: "Type", Value: analyzer.Type()},
		{Label: "Status", Value: analyzer.Status()},
	}
}

// Navigations returns available navigations from an analyzer.
func (r *AnalyzerRenderer) Navigations(resource dao.Resource) []render.Navigation {
	analyzer, ok := resource.(*AnalyzerResource)
	if !ok {
		return nil
	}
	return []render.Navigation{
		{
			Key:         "f",
			Label:       "Findings",
			Service:     "accessanalyzer",
			Resource:    "findings",
			FilterField: "AnalyzerArn",
			FilterValue: analyzer.GetARN(),
		},
	}
}
