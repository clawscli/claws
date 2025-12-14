package databases

import (
	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/render"
)

// DatabaseRenderer renders Glue databases.
// Ensure DatabaseRenderer implements render.Navigator
var _ render.Navigator = (*DatabaseRenderer)(nil)

type DatabaseRenderer struct {
	render.BaseRenderer
}

// NewDatabaseRenderer creates a new DatabaseRenderer.
func NewDatabaseRenderer() render.Renderer {
	return &DatabaseRenderer{
		BaseRenderer: render.BaseRenderer{
			Service:  "glue",
			Resource: "databases",
			Cols: []render.Column{
				{Name: "DATABASE NAME", Width: 35, Getter: func(r dao.Resource) string { return r.GetID() }},
				{Name: "DESCRIPTION", Width: 40, Getter: getDescription},
				{Name: "LOCATION", Width: 35, Getter: getLocation},
				{Name: "CREATED", Width: 20, Getter: getCreated},
			},
		},
	}
}

func getDescription(r dao.Resource) string {
	db, ok := r.(*DatabaseResource)
	if !ok {
		return ""
	}
	desc := db.Description()
	if len(desc) > 37 {
		return desc[:37] + "..."
	}
	return desc
}

func getLocation(r dao.Resource) string {
	db, ok := r.(*DatabaseResource)
	if !ok {
		return ""
	}
	loc := db.LocationUri()
	if len(loc) > 32 {
		return loc[:32] + "..."
	}
	return loc
}

func getCreated(r dao.Resource) string {
	db, ok := r.(*DatabaseResource)
	if !ok {
		return ""
	}
	if t := db.CreateTime(); t != nil {
		return t.Format("2006-01-02 15:04")
	}
	return ""
}

// RenderDetail renders the detail view for a Glue database.
func (r *DatabaseRenderer) RenderDetail(resource dao.Resource) string {
	db, ok := resource.(*DatabaseResource)
	if !ok {
		return ""
	}

	d := render.NewDetailBuilder()

	d.Title("Glue Database", db.Name())

	// Basic Info
	d.Section("Basic Information")
	d.Field("Database Name", db.Name())
	d.Field("Catalog ID", db.CatalogId())
	if desc := db.Description(); desc != "" {
		d.Field("Description", desc)
	}

	// Location
	if loc := db.LocationUri(); loc != "" {
		d.Section("Location")
		d.Field("Location URI", loc)
	}

	// Timestamps
	if t := db.CreateTime(); t != nil {
		d.Section("Timestamps")
		d.Field("Created", t.Format("2006-01-02 15:04:05"))
	}

	return d.String()
}

// RenderSummary renders summary fields for a Glue database.
func (r *DatabaseRenderer) RenderSummary(resource dao.Resource) []render.SummaryField {
	db, ok := resource.(*DatabaseResource)
	if !ok {
		return r.BaseRenderer.RenderSummary(resource)
	}

	fields := []render.SummaryField{
		{Label: "Database Name", Value: db.Name()},
		{Label: "Catalog ID", Value: db.CatalogId()},
	}

	if desc := db.Description(); desc != "" {
		fields = append(fields, render.SummaryField{Label: "Description", Value: desc})
	}

	if loc := db.LocationUri(); loc != "" {
		fields = append(fields, render.SummaryField{Label: "Location", Value: loc})
	}

	return fields
}

// Navigations returns available navigations from a database.
func (r *DatabaseRenderer) Navigations(resource dao.Resource) []render.Navigation {
	db, ok := resource.(*DatabaseResource)
	if !ok {
		return nil
	}
	return []render.Navigation{
		{
			Key:         "t",
			Label:       "Tables",
			Service:     "glue",
			Resource:    "tables",
			FilterField: "DatabaseName",
			FilterValue: db.Name(),
		},
	}
}
