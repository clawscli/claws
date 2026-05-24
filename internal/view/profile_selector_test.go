package view

import (
	"context"
	"io"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	awsui "github.com/clawscli/claws/internal/aws"
	"github.com/clawscli/claws/internal/config"
	navmsg "github.com/clawscli/claws/internal/msg"
)

func testProfiles() []profileItem {
	return []profileItem{
		{id: "default", display: "default", isSSO: false},
		{id: "dev", display: "dev", isSSO: false},
		{id: "prod-sso", display: "prod-sso", isSSO: true},
	}
}

func TestProfileSelectorMouseHover(t *testing.T) {
	selector := NewProfileSelector()
	selector.SetSize(100, 50)

	selector.Update(profilesLoadedMsg{profiles: testProfiles()})

	initialCursor := selector.selector.Cursor()

	motionMsg := tea.MouseMotionMsg{X: 10, Y: 3}
	selector.Update(motionMsg)

	t.Logf("Cursor after hover: %d (was %d)", selector.selector.Cursor(), initialCursor)
}

func TestProfileSelectorMouseClick(t *testing.T) {
	selector := NewProfileSelector()
	selector.SetSize(100, 50)

	selector.Update(profilesLoadedMsg{profiles: testProfiles()})

	clickMsg := tea.MouseClickMsg{X: 10, Y: 3, Button: tea.MouseLeft}
	_, cmd := selector.Update(clickMsg)

	t.Logf("Command after click: %v", cmd)
}

func TestProfileSelectorEmptyFilter(t *testing.T) {
	selector := NewProfileSelector()
	selector.SetSize(100, 50)

	selector.Update(profilesLoadedMsg{profiles: testProfiles()})

	selector.Update(tea.KeyPressMsg{Code: '/', Text: "/"})
	for _, r := range "zzz-nonexistent" {
		selector.Update(tea.KeyPressMsg{Code: r, Text: string(r)})
	}
	selector.Update(tea.KeyPressMsg{Code: tea.KeyEnter})

	if selector.selector.FilteredLen() != 0 {
		t.Errorf("Expected 0 filtered profiles, got %d", selector.selector.FilteredLen())
	}
	if selector.selector.Cursor() != -1 {
		t.Errorf("Expected cursor -1 for empty filter, got %d", selector.selector.Cursor())
	}

	selector.Update(tea.KeyPressMsg{Code: 'c', Text: "c"})

	if selector.selector.FilteredLen() != 3 {
		t.Errorf("Expected 3 filtered profiles after clear, got %d", selector.selector.FilteredLen())
	}
	if selector.selector.Cursor() < 0 {
		t.Errorf("Expected cursor >= 0 after clear, got %d", selector.selector.Cursor())
	}
}

func TestProfileSelectorFilterMatching(t *testing.T) {
	selector := NewProfileSelector()
	selector.SetSize(100, 50)

	profiles := []profileItem{
		{id: "default", display: "default", isSSO: false},
		{id: "dev", display: "dev", isSSO: false},
		{id: "dev-staging", display: "dev-staging", isSSO: false},
		{id: "prod-sso", display: "prod-sso", isSSO: true},
	}
	selector.Update(profilesLoadedMsg{profiles: profiles})

	selector.Update(tea.KeyPressMsg{Code: '/', Text: "/"})
	for _, r := range "dev" {
		selector.Update(tea.KeyPressMsg{Code: r, Text: string(r)})
	}
	selector.Update(tea.KeyPressMsg{Code: tea.KeyEnter})

	if selector.selector.FilteredLen() != 2 {
		t.Errorf("Expected 2 profiles matching 'dev', got %d", selector.selector.FilteredLen())
	}

	selector.Update(tea.KeyPressMsg{Code: 'c', Text: "c"})

	selector.Update(tea.KeyPressMsg{Code: '/', Text: "/"})
	for _, r := range "sso" {
		selector.Update(tea.KeyPressMsg{Code: r, Text: string(r)})
	}
	selector.Update(tea.KeyPressMsg{Code: tea.KeyEnter})

	if selector.selector.FilteredLen() != 1 {
		t.Errorf("Expected 1 profile matching 'sso', got %d", selector.selector.FilteredLen())
	}
}

func TestProfileSelectorSSODetection(t *testing.T) {
	selector := NewProfileSelector()
	selector.SetSize(100, 50)

	profiles := []profileItem{
		{id: "default", display: "default", isSSO: false},
		{id: "prod-sso", display: "prod-sso", isSSO: true},
	}
	selector.Update(profilesLoadedMsg{profiles: profiles})

	var ssoProfile profileItem
	foundSSOProfile := false
	for i := range selector.profiles {
		if selector.profiles[i].isSSO {
			ssoProfile = selector.profiles[i]
			foundSSOProfile = true
			break
		}
	}

	if !foundSSOProfile {
		t.Fatal("Expected to find SSO profile")
	}
	if ssoProfile.id != "prod-sso" {
		t.Errorf("Expected SSO profile 'prod-sso', got %q", ssoProfile.id)
	}

	var nonSSOProfile profileItem
	foundNonSSOProfile := false
	for i := range selector.profiles {
		if !selector.profiles[i].isSSO {
			nonSSOProfile = selector.profiles[i]
			foundNonSSOProfile = true
			break
		}
	}

	if !foundNonSSOProfile {
		t.Fatal("Expected to find non-SSO profile")
	}
	if nonSSOProfile.isSSO {
		t.Error("Expected non-SSO profile to have isSSO=false")
	}
}

func TestProfileSelectorToggle(t *testing.T) {
	selector := NewProfileSelector()
	selector.SetSize(100, 50)

	profiles := []profileItem{
		{id: "default", display: "default", isSSO: false},
		{id: "dev", display: "dev", isSSO: false},
	}
	selector.Update(profilesLoadedMsg{profiles: profiles})

	selector.Update(tea.KeyPressMsg{Code: tea.KeySpace})
	if !selector.selector.Selected()["default"] {
		t.Error("Expected 'default' to be selected after toggle")
	}

	selector.Update(tea.KeyPressMsg{Code: tea.KeySpace})
	if selector.selector.Selected()["default"] {
		t.Error("Expected 'default' to be deselected after second toggle")
	}

	selector.Update(tea.KeyPressMsg{Code: tea.KeySpace})
	selector.Update(tea.KeyPressMsg{Code: 'j', Text: "j"})
	selector.Update(tea.KeyPressMsg{Code: tea.KeySpace})

	if !selector.selector.Selected()["default"] || !selector.selector.Selected()["dev"] {
		t.Error("Expected both profiles to be selected")
	}
}

func TestProfileSelectorConsoleLoginSuccessSwitchesAndEmitsProfileChange(t *testing.T) {
	config.Global().UseEnvOnly()
	t.Cleanup(func() { config.Global().UseSDKDefault() })

	selector := NewProfileSelector()
	selector.SetSize(100, 50)
	selector.Update(profilesLoadedMsg{profiles: []profileItem{
		{id: config.ProfileIDEnvOnly, display: "Env/IMDS Only", isSSO: false},
		{id: "dev", display: "dev", isSSO: false},
	}})

	_, cmd := selector.Update(loginResultMsg{profileID: "dev", success: true, isConsoleLogin: true})
	if cmd == nil {
		t.Fatal("Expected console login success to emit profile change command")
	}

	msg := cmd()
	profileMsg, ok := msg.(navmsg.ProfilesChangedMsg)
	if !ok {
		t.Fatalf("command message = %T, want ProfilesChangedMsg", msg)
	}
	if len(profileMsg.Selections) != 1 || profileMsg.Selections[0].ID() != "dev" {
		t.Fatalf("ProfilesChangedMsg selections = %v, want [dev]", profileMsg.Selections)
	}

	if got := config.Global().Selection().ID(); got != "dev" {
		t.Errorf("global selection = %q, want dev", got)
	}
	if selector.selector.Selected()[config.ProfileIDEnvOnly] {
		t.Error("env-only selection should be cleared after console login switches profile")
	}
	if !selector.selector.Selected()["dev"] {
		t.Error("dev profile should be selected after console login")
	}
}

func TestProfileSelectorSSOLoginUsesSDKRunner(t *testing.T) {
	selector := NewProfileSelector()
	selector.SetSize(100, 50)
	selector.Update(profilesLoadedMsg{
		profiles: []profileItem{{id: "prod-sso", display: "prod-sso", isSSO: true}},
		infoMap: map[string]awsui.ProfileInfo{
			"prod-sso": {
				Name:         "prod-sso",
				SSOSession:   "prod-session",
				SSOStartURL:  "https://example.awsapps.com/start",
				SSORegion:    "us-east-1",
				SSOAccountID: "123456789012",
				SSORoleName:  "ReadOnly",
			},
		},
	})

	called := false
	selector.ssoLogin = func(_ context.Context, profile awsui.ProfileInfo, _ io.Writer) (awsui.SSOLoginResult, error) {
		called = true
		if profile.Name != "prod-sso" {
			t.Fatalf("profile.Name = %q, want prod-sso", profile.Name)
		}
		return awsui.SSOLoginResult{Message: "SSO session ready", ExpiresAt: time.Now().Add(time.Hour)}, nil
	}

	_, cmd := selector.Update(tea.KeyPressMsg{Code: 'l', Text: "l"})
	if cmd == nil {
		t.Fatal("expected SSO login command")
	}

	execCmd := &ssoLoginExec{profile: selector.profileInfo["prod-sso"], run: selector.ssoLogin}
	if err := execCmd.Run(); err != nil {
		t.Fatalf("ssoLoginExec.Run() error = %v", err)
	}
	if !called {
		t.Fatal("expected SSO login runner to be called")
	}
	if execCmd.result.Message != "SSO session ready" {
		t.Fatalf("result.Message = %q, want SSO session ready", execCmd.result.Message)
	}
}
