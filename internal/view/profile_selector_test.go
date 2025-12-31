package view

import (
	"context"
	"testing"

	tea "charm.land/bubbletea/v2"
)

func testProfiles() []profileItem {
	return []profileItem{
		{id: "default", display: "default", isSSO: false},
		{id: "dev", display: "dev", isSSO: false},
		{id: "prod-sso", display: "prod-sso", isSSO: true},
	}
}

func TestProfileSelectorMouseHover(t *testing.T) {
	ctx := context.Background()

	selector := NewProfileSelector(ctx)
	selector.SetSize(100, 50)

	selector.Update(profilesLoadedMsg{profiles: testProfiles()})

	initialCursor := selector.selector.Cursor()

	motionMsg := tea.MouseMotionMsg{X: 10, Y: 3}
	selector.Update(motionMsg)

	t.Logf("Cursor after hover: %d (was %d)", selector.selector.Cursor(), initialCursor)
}

func TestProfileSelectorMouseClick(t *testing.T) {
	ctx := context.Background()

	selector := NewProfileSelector(ctx)
	selector.SetSize(100, 50)

	selector.Update(profilesLoadedMsg{profiles: testProfiles()})

	clickMsg := tea.MouseClickMsg{X: 10, Y: 3, Button: tea.MouseLeft}
	_, cmd := selector.Update(clickMsg)

	t.Logf("Command after click: %v", cmd)
}

func TestProfileSelectorEmptyFilter(t *testing.T) {
	ctx := context.Background()

	selector := NewProfileSelector(ctx)
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
	ctx := context.Background()

	selector := NewProfileSelector(ctx)
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
	ctx := context.Background()

	selector := NewProfileSelector(ctx)
	selector.SetSize(100, 50)

	profiles := []profileItem{
		{id: "default", display: "default", isSSO: false},
		{id: "prod-sso", display: "prod-sso", isSSO: true},
	}
	selector.Update(profilesLoadedMsg{profiles: profiles})

	var ssoProfile *profileItem
	for i := range selector.profiles {
		if selector.profiles[i].isSSO {
			ssoProfile = &selector.profiles[i]
			break
		}
	}

	if ssoProfile == nil {
		t.Fatal("Expected to find SSO profile")
	}
	if ssoProfile.id != "prod-sso" {
		t.Errorf("Expected SSO profile 'prod-sso', got %q", ssoProfile.id)
	}

	var nonSSOProfile *profileItem
	for i := range selector.profiles {
		if !selector.profiles[i].isSSO {
			nonSSOProfile = &selector.profiles[i]
			break
		}
	}

	if nonSSOProfile == nil {
		t.Fatal("Expected to find non-SSO profile")
	}
	if nonSSOProfile.isSSO {
		t.Error("Expected non-SSO profile to have isSSO=false")
	}
}

func TestProfileSelectorToggle(t *testing.T) {
	ctx := context.Background()

	selector := NewProfileSelector(ctx)
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
