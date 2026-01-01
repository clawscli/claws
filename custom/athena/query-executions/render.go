package queryexecutions

import (
	"fmt"

	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/render"
)

// QueryExecutionRenderer renders Athena query executions.
type QueryExecutionRenderer struct {
	render.BaseRenderer
}

// NewQueryExecutionRenderer creates a new QueryExecutionRenderer.
func NewQueryExecutionRenderer() render.Renderer {
	return &QueryExecutionRenderer{
		BaseRenderer: render.BaseRenderer{
			Service:  "athena",
			Resource: "query-executions",
			Cols: []render.Column{
				{Name: "QUERY ID", Width: 38, Getter: func(r dao.Resource) string { return r.GetID() }},
				{Name: "STATE", Width: 12, Getter: getState},
				{Name: "DATABASE", Width: 20, Getter: getDatabase},
				{Name: "SUBMITTED", Width: 18, Getter: getSubmitted},
				{Name: "DATA SCANNED", Width: 14, Getter: getDataScanned},
			},
		},
	}
}

func getState(r dao.Resource) string {
	qe, ok := r.(*QueryExecutionResource)
	if !ok {
		return ""
	}
	return qe.State()
}

func getDatabase(r dao.Resource) string {
	qe, ok := r.(*QueryExecutionResource)
	if !ok {
		return ""
	}
	return qe.Database()
}

func getSubmitted(r dao.Resource) string {
	qe, ok := r.(*QueryExecutionResource)
	if !ok {
		return ""
	}
	if t := qe.SubmissionTime(); t != nil {
		return t.Format("2006-01-02 15:04")
	}
	return ""
}

func getDataScanned(r dao.Resource) string {
	qe, ok := r.(*QueryExecutionResource)
	if !ok {
		return ""
	}
	bytes := qe.DataScannedBytes()
	if bytes == 0 {
		return ""
	}
	if bytes >= 1024*1024*1024 {
		return fmt.Sprintf("%.2f GB", float64(bytes)/(1024*1024*1024))
	}
	if bytes >= 1024*1024 {
		return fmt.Sprintf("%.2f MB", float64(bytes)/(1024*1024))
	}
	if bytes >= 1024 {
		return fmt.Sprintf("%.2f KB", float64(bytes)/1024)
	}
	return fmt.Sprintf("%d B", bytes)
}

// RenderDetail renders the detail view for an Athena query execution.
func (r *QueryExecutionRenderer) RenderDetail(resource dao.Resource) string {
	qe, ok := resource.(*QueryExecutionResource)
	if !ok {
		return ""
	}

	d := render.NewDetailBuilder()

	d.Title("Athena Query Execution", qe.GetID())

	// Basic Info
	d.Section("Basic Information")
	d.Field("Query ID", qe.GetID())
	d.Field("State", qe.State())
	d.Field("WorkGroup", qe.WorkGroup())
	if db := qe.Database(); db != "" {
		d.Field("Database", db)
	}

	// Query
	d.Section("Query")
	d.Field("SQL", qe.Query())

	// Timing
	d.Section("Timing")
	if t := qe.SubmissionTime(); t != nil {
		d.Field("Submitted", t.Format("2006-01-02 15:04:05"))
	}
	if t := qe.CompletionTime(); t != nil {
		d.Field("Completed", t.Format("2006-01-02 15:04:05"))
	}
	if ms := qe.ExecutionTimeMs(); ms > 0 {
		d.Field("Execution Time", fmt.Sprintf("%d ms", ms))
	}

	// Statistics
	if bytes := qe.DataScannedBytes(); bytes > 0 {
		d.Section("Statistics")
		d.Field("Data Scanned", fmt.Sprintf("%d bytes", bytes))
	}

	// Output
	if loc := qe.OutputLocation(); loc != "" {
		d.Section("Output")
		d.Field("Location", loc)
	}

	// Error
	if reason := qe.StateChangeReason(); reason != "" && qe.State() == "FAILED" {
		d.Section("Error")
		d.Field("Reason", reason)
	}

	return d.String()
}

// RenderSummary renders summary fields for an Athena query execution.
func (r *QueryExecutionRenderer) RenderSummary(resource dao.Resource) []render.SummaryField {
	qe, ok := resource.(*QueryExecutionResource)
	if !ok {
		return r.BaseRenderer.RenderSummary(resource)
	}

	fields := []render.SummaryField{
		{Label: "Query ID", Value: qe.GetID()},
		{Label: "State", Value: qe.State()},
		{Label: "WorkGroup", Value: qe.WorkGroup()},
	}

	if db := qe.Database(); db != "" {
		fields = append(fields, render.SummaryField{Label: "Database", Value: db})
	}

	return fields
}
