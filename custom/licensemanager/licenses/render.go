package licenses

import (
	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/render"
)

// LicenseRenderer renders License Manager licenses.
// Ensure LicenseRenderer implements render.Navigator
var _ render.Navigator = (*LicenseRenderer)(nil)

type LicenseRenderer struct {
	render.BaseRenderer
}

// NewLicenseRenderer creates a new LicenseRenderer.
func NewLicenseRenderer() render.Renderer {
	return &LicenseRenderer{
		BaseRenderer: render.BaseRenderer{
			Service:  "license-manager",
			Resource: "licenses",
			Cols: []render.Column{
				{Name: "NAME", Width: 35, Getter: getName},
				{Name: "PRODUCT", Width: 25, Getter: getProduct},
				{Name: "ISSUER", Width: 20, Getter: getIssuer},
				{Name: "STATUS", Width: 15, Getter: getStatus},
			},
		},
	}
}

func getName(r dao.Resource) string {
	license, ok := r.(*LicenseResource)
	if !ok {
		return ""
	}
	return license.Name()
}

func getProduct(r dao.Resource) string {
	license, ok := r.(*LicenseResource)
	if !ok {
		return ""
	}
	return license.ProductName()
}

func getIssuer(r dao.Resource) string {
	license, ok := r.(*LicenseResource)
	if !ok {
		return ""
	}
	return license.Issuer()
}

func getStatus(r dao.Resource) string {
	license, ok := r.(*LicenseResource)
	if !ok {
		return ""
	}
	return license.Status()
}

// RenderDetail renders the detail view for a license.
func (r *LicenseRenderer) RenderDetail(resource dao.Resource) string {
	license, ok := resource.(*LicenseResource)
	if !ok {
		return ""
	}

	d := render.NewDetailBuilder()

	d.Title("License", license.Name())

	// Basic Info
	d.Section("Basic Information")
	d.Field("Name", license.Name())
	d.Field("ARN", license.GetARN())
	d.Field("Product", license.ProductName())
	d.Field("Status", license.Status())

	// Parties
	d.Section("Parties")
	d.Field("Issuer", license.Issuer())
	if license.Beneficiary() != "" {
		d.Field("Beneficiary", license.Beneficiary())
	}

	return d.String()
}

// RenderSummary renders summary fields for a license.
func (r *LicenseRenderer) RenderSummary(resource dao.Resource) []render.SummaryField {
	license, ok := resource.(*LicenseResource)
	if !ok {
		return r.BaseRenderer.RenderSummary(resource)
	}

	return []render.SummaryField{
		{Label: "Name", Value: license.Name()},
		{Label: "Product", Value: license.ProductName()},
		{Label: "Status", Value: license.Status()},
	}
}

// Navigations returns available navigations from a license.
func (r *LicenseRenderer) Navigations(resource dao.Resource) []render.Navigation {
	license, ok := resource.(*LicenseResource)
	if !ok {
		return nil
	}
	return []render.Navigation{
		{
			Key:         "g",
			Label:       "Grants",
			Service:     "license-manager",
			Resource:    "grants",
			FilterField: "LicenseArn",
			FilterValue: license.GetARN(),
		},
	}
}
