package tables

import (
	"fmt"

	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/render"
)

// TableRenderer renders Glue tables.
type TableRenderer struct {
	render.BaseRenderer
}

// NewTableRenderer creates a new TableRenderer.
func NewTableRenderer() render.Renderer {
	return &TableRenderer{
		BaseRenderer: render.BaseRenderer{
			Service:  "glue",
			Resource: "tables",
			Cols: []render.Column{
				{Name: "TABLE NAME", Width: 35, Getter: func(r dao.Resource) string { return r.GetID() }},
				{Name: "TYPE", Width: 15, Getter: getTableType},
				{Name: "COLUMNS", Width: 10, Getter: getColumns},
				{Name: "LOCATION", Width: 50, Getter: getLocation},
				{Name: "UPDATED", Width: 20, Getter: getUpdated},
			},
		},
	}
}

func getTableType(r dao.Resource) string {
	table, ok := r.(*TableResource)
	if !ok {
		return ""
	}
	return table.TableType()
}

func getColumns(r dao.Resource) string {
	table, ok := r.(*TableResource)
	if !ok {
		return ""
	}
	return fmt.Sprintf("%d", table.ColumnCount())
}

func getLocation(r dao.Resource) string {
	table, ok := r.(*TableResource)
	if !ok {
		return ""
	}
	loc := table.Location()
	if len(loc) > 47 {
		return loc[:47] + "..."
	}
	return loc
}

func getUpdated(r dao.Resource) string {
	table, ok := r.(*TableResource)
	if !ok {
		return ""
	}
	if t := table.UpdateTime(); t != nil {
		return t.Format("2006-01-02 15:04")
	}
	return ""
}

// RenderDetail renders the detail view for a Glue table.
func (r *TableRenderer) RenderDetail(resource dao.Resource) string {
	table, ok := resource.(*TableResource)
	if !ok {
		return ""
	}

	d := render.NewDetailBuilder()

	d.Title("Glue Table", table.Name())

	// Basic Info
	d.Section("Basic Information")
	d.Field("Table Name", table.Name())
	d.Field("Database", table.DatabaseName)
	if tableType := table.TableType(); tableType != "" {
		d.Field("Table Type", tableType)
	}
	if desc := table.Description(); desc != "" {
		d.Field("Description", desc)
	}

	// Storage
	d.Section("Storage")
	if loc := table.Location(); loc != "" {
		d.Field("Location", loc)
	}
	if input := table.InputFormat(); input != "" {
		d.Field("Input Format", input)
	}
	if output := table.OutputFormat(); output != "" {
		d.Field("Output Format", output)
	}

	// Schema
	d.Section("Schema")
	d.Field("Column Count", fmt.Sprintf("%d", table.ColumnCount()))

	// Timestamps
	d.Section("Timestamps")
	if t := table.CreateTime(); t != nil {
		d.Field("Created", t.Format("2006-01-02 15:04:05"))
	}
	if t := table.UpdateTime(); t != nil {
		d.Field("Updated", t.Format("2006-01-02 15:04:05"))
	}

	return d.String()
}

// RenderSummary renders summary fields for a Glue table.
func (r *TableRenderer) RenderSummary(resource dao.Resource) []render.SummaryField {
	table, ok := resource.(*TableResource)
	if !ok {
		return r.BaseRenderer.RenderSummary(resource)
	}

	fields := []render.SummaryField{
		{Label: "Table Name", Value: table.Name()},
		{Label: "Database", Value: table.DatabaseName},
		{Label: "Columns", Value: fmt.Sprintf("%d", table.ColumnCount())},
	}

	if tableType := table.TableType(); tableType != "" {
		fields = append(fields, render.SummaryField{Label: "Type", Value: tableType})
	}

	if loc := table.Location(); loc != "" {
		fields = append(fields, render.SummaryField{Label: "Location", Value: loc})
	}

	return fields
}
