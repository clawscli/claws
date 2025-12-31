package view

import (
	tea "charm.land/bubbletea/v2"

	"github.com/clawscli/claws/internal/aws"
	"github.com/clawscli/claws/internal/render"
)

type ProfileDetailView struct {
	profile      profileItem
	info         aws.ProfileInfo
	hasInfo      bool
	contentCache string
}

func NewProfileDetailView(profile profileItem, info aws.ProfileInfo, hasInfo bool) *ProfileDetailView {
	v := &ProfileDetailView{
		profile: profile,
		info:    info,
		hasInfo: hasInfo,
	}
	v.contentCache = v.buildContent()
	return v
}

func (v *ProfileDetailView) Init() tea.Cmd {
	return nil
}

func (v *ProfileDetailView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "d":
			return v, func() tea.Msg { return HideModalMsg{} }
		}
	}
	return v, nil
}

func (v *ProfileDetailView) View() tea.View {
	return tea.NewView(v.ViewString())
}

func (v *ProfileDetailView) ViewString() string {
	return v.contentCache
}

func (v *ProfileDetailView) SetSize(_, _ int) tea.Cmd {
	return nil
}

func (v *ProfileDetailView) StatusLine() string {
	return "Esc/d/q:close"
}

func (v *ProfileDetailView) buildContent() string {
	d := render.NewDetailBuilder()
	d.Title("Profile", v.profile.display)

	d.Section("Configuration")
	d.Field("Type", v.profile.profileType)
	if v.profile.region != "" {
		d.Field("Region", v.profile.region)
	}

	if !v.hasInfo {
		return d.String()
	}

	if v.info.RoleArn != "" {
		d.Section("Role Assumption")
		d.Field("Role ARN", v.info.RoleArn)
		if v.info.SourceProfile != "" {
			d.Field("Source Profile", v.info.SourceProfile)
		}
	}

	if v.info.IsSSO {
		d.Section("SSO Configuration")
		if v.info.SSOSession != "" {
			d.Field("SSO Session", v.info.SSOSession)
		}
		if v.info.SSOStartURL != "" {
			d.Field("Start URL", v.info.SSOStartURL)
		}
		if v.info.SSORegion != "" {
			d.Field("SSO Region", v.info.SSORegion)
		}
		if v.info.SSOAccountID != "" {
			d.Field("Account ID", v.info.SSOAccountID)
		}
		if v.info.SSORoleName != "" {
			d.Field("Role Name", v.info.SSORoleName)
		}
	}

	if v.info.HasCredentials {
		d.Section("Credentials")
		d.Field("Access Key ID", v.info.AccessKeyID)
	}

	return d.String()
}
