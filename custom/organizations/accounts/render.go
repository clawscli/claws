package accounts

import (
	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/render"
)

// AccountRenderer renders Organizations accounts.
type AccountRenderer struct {
	render.BaseRenderer
}

// NewAccountRenderer creates a new AccountRenderer.
func NewAccountRenderer() render.Renderer {
	return &AccountRenderer{
		BaseRenderer: render.BaseRenderer{
			Service:  "organizations",
			Resource: "accounts",
			Cols: []render.Column{
				{Name: "ACCOUNT ID", Width: 15, Getter: func(r dao.Resource) string { return r.GetID() }},
				{Name: "NAME", Width: 30, Getter: getName},
				{Name: "EMAIL", Width: 35, Getter: getEmail},
				{Name: "STATUS", Width: 12, Getter: getStatus},
			},
		},
	}
}

func getName(r dao.Resource) string {
	account, ok := r.(*AccountResource)
	if !ok {
		return ""
	}
	return account.Name()
}

func getEmail(r dao.Resource) string {
	account, ok := r.(*AccountResource)
	if !ok {
		return ""
	}
	return account.Email()
}

func getStatus(r dao.Resource) string {
	account, ok := r.(*AccountResource)
	if !ok {
		return ""
	}
	return account.Status()
}

// RenderDetail renders the detail view for an account.
func (r *AccountRenderer) RenderDetail(resource dao.Resource) string {
	account, ok := resource.(*AccountResource)
	if !ok {
		return ""
	}

	d := render.NewDetailBuilder()

	d.Title("Organizations Account", account.GetID())

	// Basic Info
	d.Section("Basic Information")
	d.Field("Account ID", account.GetID())
	d.Field("Name", account.Name())
	d.Field("Email", account.Email())
	d.Field("ARN", account.GetARN())
	d.Field("Status", account.Status())

	// Membership
	d.Section("Membership")
	d.Field("Joined Method", account.JoinedMethod())
	if t := account.JoinedTimestamp(); t != nil {
		d.Field("Joined", t.Format("2006-01-02 15:04:05"))
	}

	return d.String()
}

// RenderSummary renders summary fields for an account.
func (r *AccountRenderer) RenderSummary(resource dao.Resource) []render.SummaryField {
	account, ok := resource.(*AccountResource)
	if !ok {
		return r.BaseRenderer.RenderSummary(resource)
	}

	return []render.SummaryField{
		{Label: "Account ID", Value: account.GetID()},
		{Label: "Name", Value: account.Name()},
		{Label: "Email", Value: account.Email()},
		{Label: "Status", Value: account.Status()},
	}
}
