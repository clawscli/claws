package users

import (
	"fmt"
	"sort"

	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/render"
)

// UserRenderer renders Cognito users
// Ensure UserRenderer implements render.Navigator
var _ render.Navigator = (*UserRenderer)(nil)

type UserRenderer struct {
	render.BaseRenderer
}

// NewUserRenderer creates a new UserRenderer
func NewUserRenderer() *UserRenderer {
	return &UserRenderer{
		BaseRenderer: render.BaseRenderer{
			Service:  "cognito",
			Resource: "users",
			Cols: []render.Column{
				{Name: "USERNAME", Width: 30, Getter: getUsername},
				{Name: "EMAIL", Width: 35, Getter: getEmail},
				{Name: "STATUS", Width: 15, Getter: getStatus},
				{Name: "ENABLED", Width: 8, Getter: getEnabled},
				{Name: "AGE", Width: 12, Getter: getAge},
			},
		},
	}
}

func getUsername(r dao.Resource) string {
	if u, ok := r.(*UserResource); ok {
		return u.Username()
	}
	return ""
}

func getEmail(r dao.Resource) string {
	if u, ok := r.(*UserResource); ok {
		return u.Email()
	}
	return ""
}

func getStatus(r dao.Resource) string {
	if u, ok := r.(*UserResource); ok {
		return u.Status()
	}
	return ""
}

func getEnabled(r dao.Resource) string {
	if u, ok := r.(*UserResource); ok {
		if u.Enabled() {
			return "Yes"
		}
		return "No"
	}
	return ""
}

func getAge(r dao.Resource) string {
	if u, ok := r.(*UserResource); ok {
		if t := u.CreatedAtTime(); t != nil {
			return render.FormatAge(*t)
		}
	}
	return "-"
}

// RenderDetail renders detailed user information
func (r *UserRenderer) RenderDetail(resource dao.Resource) string {
	user, ok := resource.(*UserResource)
	if !ok {
		return ""
	}

	d := render.NewDetailBuilder()

	d.Title("Cognito User", user.Username())

	// Basic Info
	d.Section("Basic Information")
	d.Field("Username", user.Username())
	d.Field("User Pool ID", user.UserPoolId)
	d.Field("Status", user.Status())
	d.Field("Enabled", fmt.Sprintf("%v", user.Enabled()))

	// Contact Info
	if email := user.Email(); email != "" {
		d.Section("Contact")
		d.Field("Email", email)
		if phone := user.PhoneNumber(); phone != "" {
			d.Field("Phone", phone)
		}
	}

	// Name
	if name := user.Name(); name != "" {
		d.Section("Name")
		d.Field("Full Name", name)
	} else if givenName := user.GivenName(); givenName != "" {
		d.Section("Name")
		d.Field("Given Name", givenName)
		if familyName := user.FamilyName(); familyName != "" {
			d.Field("Family Name", familyName)
		}
	}

	// All Attributes
	attrs := user.Attributes()
	if len(attrs) > 0 {
		d.Section("All Attributes")
		keys := make([]string, 0, len(attrs))
		for k := range attrs {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			d.Field(k, attrs[k])
		}
	}

	// Timestamps
	d.Section("Timestamps")
	if created := user.CreatedAt(); created != "" {
		d.Field("Created", created)
	}
	if modified := user.LastModifiedDate(); modified != "" {
		d.Field("Last Modified", modified)
	}

	return d.String()
}

// RenderSummary returns summary fields for the header panel
func (r *UserRenderer) RenderSummary(resource dao.Resource) []render.SummaryField {
	user, ok := resource.(*UserResource)
	if !ok {
		return r.BaseRenderer.RenderSummary(resource)
	}

	fields := []render.SummaryField{
		{Label: "Username", Value: user.Username()},
		{Label: "Status", Value: user.Status()},
		{Label: "Enabled", Value: fmt.Sprintf("%v", user.Enabled())},
	}

	if email := user.Email(); email != "" {
		fields = append(fields, render.SummaryField{Label: "Email", Value: email})
	}

	fields = append(fields, render.SummaryField{Label: "User Pool ID", Value: user.UserPoolId})

	if created := user.CreatedAt(); created != "" {
		fields = append(fields, render.SummaryField{Label: "Created", Value: created})
	}

	return fields
}

// Navigations returns navigation shortcuts
func (r *UserRenderer) Navigations(resource dao.Resource) []render.Navigation {
	return nil
}
