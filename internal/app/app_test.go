package app

import (
	"context"
	"fmt"
	"testing"

	tea "charm.land/bubbletea/v2"

	navmsg "github.com/clawscli/claws/internal/msg"
	"github.com/clawscli/claws/internal/registry"
	"github.com/clawscli/claws/internal/view"
)

// MockView is a simple view for testing
type MockView struct {
	name        string
	hasInput    bool
	escReceived bool
}

func (m *MockView) Init() tea.Cmd                     { return nil }
func (m *MockView) View() tea.View                    { return tea.NewView(m.name) }
func (m *MockView) ViewString() string                { return m.name }
func (m *MockView) SetSize(width, height int) tea.Cmd { return nil }
func (m *MockView) StatusLine() string                { return m.name }
func (m *MockView) HasActiveInput() bool              { return m.hasInput }
func (m *MockView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyPressMsg); ok && keyMsg.String() == "esc" {
		m.escReceived = true
		m.hasInput = false // Close input on esc
	}
	return m, nil
}

type RefreshableMockView struct {
	MockView
	canRefresh bool
}

func (m *RefreshableMockView) CanRefresh() bool { return m.canRefresh }

func (m *RefreshableMockView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	return m, nil
}

func newTestApp(t *testing.T) *App {
	t.Helper()
	ctx := context.Background()
	reg := registry.New()
	app := New(ctx, reg, nil)
	app.width = 100
	app.height = 50
	return app
}

func TestEscInDetailView(t *testing.T) {
	app := newTestApp(t)

	// Set up view stack: ServiceBrowser -> ResourceBrowser -> DetailView
	serviceBrowser := &MockView{name: "ServiceBrowser"}
	resourceBrowser := &MockView{name: "ResourceBrowser"}
	detailView := &MockView{name: "DetailView"}

	app.viewStack = []view.View{serviceBrowser, resourceBrowser}
	app.currentView = detailView

	// Press esc
	escMsg := tea.KeyPressMsg{Code: tea.KeyEscape}
	t.Logf("Before esc: currentView=%s, viewStack=%d", app.currentView.StatusLine(), len(app.viewStack))

	app.Update(escMsg)

	t.Logf("After esc: currentView=%s, viewStack=%d", app.currentView.StatusLine(), len(app.viewStack))

	// Should now be on ResourceBrowser
	if app.currentView.StatusLine() != "ResourceBrowser" {
		t.Errorf("Expected currentView to be ResourceBrowser, got %s", app.currentView.StatusLine())
	}

	// ViewStack should have 1 item (ServiceBrowser)
	if len(app.viewStack) != 1 {
		t.Errorf("Expected viewStack length 1, got %d", len(app.viewStack))
	}

	// Press esc again
	app.Update(escMsg)

	t.Logf("After 2nd esc: currentView=%s, viewStack=%d", app.currentView.StatusLine(), len(app.viewStack))

	// Should now be on ServiceBrowser
	if app.currentView.StatusLine() != "ServiceBrowser" {
		t.Errorf("Expected currentView to be ServiceBrowser, got %s", app.currentView.StatusLine())
	}

	// ViewStack should be empty
	if len(app.viewStack) != 0 {
		t.Errorf("Expected viewStack length 0, got %d", len(app.viewStack))
	}
}

func TestEscInFilterMode(t *testing.T) {
	app := newTestApp(t)

	resourceBrowser := &MockView{name: "ResourceBrowser", hasInput: true}
	serviceBrowser := &MockView{name: "ServiceBrowser"}

	app.viewStack = []view.View{serviceBrowser}
	app.currentView = resourceBrowser

	// Press esc - should close filter, NOT go back
	escMsg := tea.KeyPressMsg{Code: tea.KeyEscape}
	t.Logf("Before esc: currentView=%s, hasInput=%v, viewStack=%d",
		app.currentView.StatusLine(), resourceBrowser.hasInput, len(app.viewStack))

	app.Update(escMsg)

	t.Logf("After esc: currentView=%s, escReceived=%v, hasInput=%v, viewStack=%d",
		app.currentView.StatusLine(), resourceBrowser.escReceived, resourceBrowser.hasInput, len(app.viewStack))

	// Should still be on ResourceBrowser
	if app.currentView.StatusLine() != "ResourceBrowser" {
		t.Errorf("Expected currentView to be ResourceBrowser, got %s", app.currentView.StatusLine())
	}

	// Esc should have been received by the view
	if !resourceBrowser.escReceived {
		t.Errorf("Expected escReceived to be true")
	}

	// hasInput should now be false
	if resourceBrowser.hasInput {
		t.Errorf("Expected hasInput to be false after esc")
	}

	// ViewStack should still have 1 item
	if len(app.viewStack) != 1 {
		t.Errorf("Expected viewStack length 1, got %d", len(app.viewStack))
	}
}

func TestNavigationFlow(t *testing.T) {
	app := newTestApp(t)

	// Start with ServiceBrowser
	serviceBrowser := &MockView{name: "ServiceBrowser"}
	app.currentView = serviceBrowser
	app.viewStack = nil

	t.Logf("Initial: currentView=%s, viewStack=%d", app.currentView.StatusLine(), len(app.viewStack))

	// Simulate navigating to ResourceBrowser
	resourceBrowser := &MockView{name: "ResourceBrowser"}
	navMsg1 := view.NavigateMsg{View: resourceBrowser}
	app.Update(navMsg1)

	t.Logf("After nav to ResourceBrowser: currentView=%s, viewStack=%d",
		app.currentView.StatusLine(), len(app.viewStack))

	if app.currentView.StatusLine() != "ResourceBrowser" {
		t.Errorf("Expected currentView to be ResourceBrowser, got %s", app.currentView.StatusLine())
	}
	if len(app.viewStack) != 1 {
		t.Errorf("Expected viewStack length 1, got %d", len(app.viewStack))
	}

	// Simulate navigating to DetailView
	detailView := &MockView{name: "DetailView"}
	navMsg2 := view.NavigateMsg{View: detailView}
	app.Update(navMsg2)

	t.Logf("After nav to DetailView: currentView=%s, viewStack=%d",
		app.currentView.StatusLine(), len(app.viewStack))

	if app.currentView.StatusLine() != "DetailView" {
		t.Errorf("Expected currentView to be DetailView, got %s", app.currentView.StatusLine())
	}
	if len(app.viewStack) != 2 {
		t.Errorf("Expected viewStack length 2, got %d", len(app.viewStack))
	}

	// Press esc - should go back to ResourceBrowser
	escMsg := tea.KeyPressMsg{Code: tea.KeyEscape}
	app.Update(escMsg)

	t.Logf("After 1st esc: currentView=%s, viewStack=%d",
		app.currentView.StatusLine(), len(app.viewStack))

	if app.currentView.StatusLine() != "ResourceBrowser" {
		t.Errorf("Expected currentView to be ResourceBrowser, got %s", app.currentView.StatusLine())
	}
	if len(app.viewStack) != 1 {
		t.Errorf("Expected viewStack length 1, got %d", len(app.viewStack))
	}

	// Press esc again - should go back to ServiceBrowser
	app.Update(escMsg)

	t.Logf("After 2nd esc: currentView=%s, viewStack=%d",
		app.currentView.StatusLine(), len(app.viewStack))

	if app.currentView.StatusLine() != "ServiceBrowser" {
		t.Errorf("Expected currentView to be ServiceBrowser, got %s", app.currentView.StatusLine())
	}
	if len(app.viewStack) != 0 {
		t.Errorf("Expected viewStack length 0, got %d", len(app.viewStack))
	}
}

func TestAWSContextReadyMsg_Success(t *testing.T) {
	app := newTestApp(t)
	app.awsInitializing = true

	// Simulate successful AWS init
	msg := awsContextReadyMsg{err: nil}
	app.Update(msg)

	if app.awsInitializing {
		t.Error("Expected awsInitializing to be false after success")
	}
	if app.showWarnings {
		t.Error("Expected showWarnings to be false after success")
	}
}

func TestAWSContextReadyMsg_Timeout(t *testing.T) {
	app := newTestApp(t)
	app.awsInitializing = true

	// Simulate timeout error
	msg := awsContextReadyMsg{err: context.DeadlineExceeded}
	app.Update(msg)

	if app.awsInitializing {
		t.Error("Expected awsInitializing to be false after timeout")
	}
	if !app.showWarnings {
		t.Error("Expected showWarnings to be true after timeout")
	}
}

func TestAWSContextReadyMsg_IMDSError(t *testing.T) {
	app := newTestApp(t)
	app.awsInitializing = true

	msg := awsContextReadyMsg{err: fmt.Errorf("operation error ec2imds: GetRegion, exceeded maximum number of attempts")}
	app.Update(msg)

	if app.awsInitializing {
		t.Error("Expected awsInitializing to be false after IMDS error")
	}
	if app.showWarnings {
		t.Error("Expected showWarnings to be false for IMDS errors (suppressed)")
	}
}

func TestProfileRefreshDoneMsg_Success(t *testing.T) {
	app := newTestApp(t)
	app.profileRefreshID = 5
	app.profileRefreshing = true

	msg := profileRefreshDoneMsg{
		refreshID:  5,
		region:     "us-west-2",
		accountIDs: map[string]string{"dev": "123456789012"},
		err:        nil,
	}
	app.Update(msg)

	if app.profileRefreshing {
		t.Error("Expected profileRefreshing to be false after success")
	}
	if app.profileRefreshError != nil {
		t.Error("Expected profileRefreshError to be nil after success")
	}
}

func TestProfileRefreshDoneMsg_StaleIgnored(t *testing.T) {
	app := newTestApp(t)
	app.profileRefreshID = 10
	app.profileRefreshing = true

	msg := profileRefreshDoneMsg{
		refreshID:  5,
		region:     "us-west-2",
		accountIDs: map[string]string{"dev": "123456789012"},
		err:        nil,
	}
	app.Update(msg)

	if !app.profileRefreshing {
		t.Error("Expected profileRefreshing to remain true for stale refresh")
	}
}

func TestProfileRefreshDoneMsg_Error(t *testing.T) {
	app := newTestApp(t)
	app.profileRefreshID = 1
	app.profileRefreshing = true

	msg := profileRefreshDoneMsg{
		refreshID: 1,
		err:       fmt.Errorf("failed to load config"),
	}
	app.Update(msg)

	if app.profileRefreshing {
		t.Error("Expected profileRefreshing to be false after error")
	}
	if app.profileRefreshError == nil {
		t.Error("Expected profileRefreshError to be set after error")
	}
	if app.showWarnings {
		t.Error("Expected showWarnings to remain false")
	}
}

func TestProfileRefreshError_ClearedOnNewRefresh(t *testing.T) {
	app := newTestApp(t)
	app.profileRefreshError = fmt.Errorf("previous error")
	app.currentView = &MockView{name: "Dashboard"}

	msg := navmsg.ProfilesChangedMsg{Selections: nil}
	app.Update(msg)

	if app.profileRefreshError != nil {
		t.Error("Expected profileRefreshError to be cleared on new profile change")
	}
	if !app.profileRefreshing {
		t.Error("Expected profileRefreshing to be true")
	}
}

func TestProfileRefresh_RapidChangesOnlyLatestHonored(t *testing.T) {
	app := newTestApp(t)
	app.currentView = &MockView{name: "Dashboard"}

	app.Update(navmsg.ProfilesChangedMsg{Selections: nil})
	firstID := app.profileRefreshID

	app.Update(navmsg.ProfilesChangedMsg{Selections: nil})
	secondID := app.profileRefreshID

	app.Update(navmsg.ProfilesChangedMsg{Selections: nil})
	thirdID := app.profileRefreshID

	if thirdID != 3 {
		t.Errorf("Expected profileRefreshID to be 3, got %d", thirdID)
	}

	staleMsg := profileRefreshDoneMsg{
		refreshID:  firstID,
		region:     "us-east-1",
		accountIDs: map[string]string{"old": "111111111111"},
	}
	app.Update(staleMsg)

	if !app.profileRefreshing {
		t.Error("Expected profileRefreshing to remain true after stale response")
	}

	anotherStaleMsg := profileRefreshDoneMsg{
		refreshID:  secondID,
		region:     "us-west-1",
		accountIDs: map[string]string{"old2": "222222222222"},
	}
	app.Update(anotherStaleMsg)

	if !app.profileRefreshing {
		t.Error("Expected profileRefreshing to remain true after another stale response")
	}

	latestMsg := profileRefreshDoneMsg{
		refreshID:  thirdID,
		region:     "ap-northeast-1",
		accountIDs: map[string]string{"latest": "333333333333"},
	}
	app.Update(latestMsg)

	if app.profileRefreshing {
		t.Error("Expected profileRefreshing to be false after latest response")
	}
}

func TestModalShowAndHide(t *testing.T) {
	app := newTestApp(t)

	serviceBrowser := &MockView{name: "ServiceBrowser"}
	app.currentView = serviceBrowser
	app.viewStack = nil

	modalContent := &MockView{name: "ActionMenu"}
	showMsg := view.ShowModalMsg{Modal: &view.Modal{Content: modalContent}}
	app.Update(showMsg)

	if app.modal == nil {
		t.Error("Expected modal to be set after ShowModalMsg")
	}
	if app.modal.Content.StatusLine() != "ActionMenu" {
		t.Errorf("Expected modal content to be ActionMenu, got %s", app.modal.Content.StatusLine())
	}

	escMsg := tea.KeyPressMsg{Code: tea.KeyEscape}
	app.Update(escMsg)

	if app.modal != nil {
		t.Error("Expected modal to be nil after esc")
	}
	if app.currentView.StatusLine() != "ServiceBrowser" {
		t.Errorf("Expected currentView to remain ServiceBrowser, got %s", app.currentView.StatusLine())
	}
}

func TestModalNavigateClosesModal(t *testing.T) {
	app := newTestApp(t)

	serviceBrowser := &MockView{name: "ServiceBrowser"}
	app.currentView = serviceBrowser
	app.viewStack = nil

	modalContent := &MockView{name: "ActionMenu"}
	app.modal = &view.Modal{Content: modalContent}

	detailView := &MockView{name: "DetailView"}
	navMsg := view.NavigateMsg{View: detailView}
	app.Update(navMsg)

	if app.modal != nil {
		t.Error("Expected modal to be closed after NavigateMsg")
	}
	if app.currentView.StatusLine() != "DetailView" {
		t.Errorf("Expected currentView to be DetailView, got %s", app.currentView.StatusLine())
	}
	if len(app.viewStack) != 1 {
		t.Errorf("Expected viewStack length 1, got %d", len(app.viewStack))
	}
}

func TestKeyOpensModal(t *testing.T) {
	tests := []struct {
		name string
		key  string
	}{
		{"region selector", "R"},
		{"profile selector", "P"},
		{"help view", "?"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := newTestApp(t)
			app.currentView = &MockView{name: "Dashboard"}
			app.viewStack = nil

			app.Update(tea.KeyPressMsg{Code: 0, Text: tt.key})

			if app.modal == nil {
				t.Errorf("Expected modal after %s key", tt.key)
			}
			if app.currentView.StatusLine() != "Dashboard" {
				t.Errorf("Expected currentView Dashboard, got %s", app.currentView.StatusLine())
			}
			if len(app.viewStack) != 0 {
				t.Errorf("Expected empty viewStack, got %d", len(app.viewStack))
			}
		})
	}
}

func TestCommandModeActivation(t *testing.T) {
	app := newTestApp(t)
	app.currentView = &MockView{name: "Dashboard"}

	app.Update(tea.KeyPressMsg{Code: 0, Text: ":"})

	if !app.commandMode {
		t.Error("Expected commandMode=true after ':' key")
	}
	if app.modal != nil {
		t.Error("Expected no modal for command mode")
	}
}

func TestModalClosesWithKey(t *testing.T) {
	tests := []struct {
		name string
		key  tea.KeyPressMsg
	}{
		{"q key", tea.KeyPressMsg{Code: 0, Text: "q"}},
		{"esc key", tea.KeyPressMsg{Code: tea.KeyEscape}},
		{"backspace", tea.KeyPressMsg{Code: tea.KeyBackspace}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := newTestApp(t)
			app.currentView = &MockView{name: "Dashboard"}
			app.modal = &view.Modal{Content: &MockView{name: "TestModal"}}

			app.Update(tt.key)

			if app.modal != nil {
				t.Errorf("Expected modal nil after %s", tt.name)
			}
			if app.currentView.StatusLine() != "Dashboard" {
				t.Errorf("Expected currentView Dashboard, got %s", app.currentView.StatusLine())
			}
		})
	}
}

func TestMessageClosesModal(t *testing.T) {
	tests := []struct {
		name string
		msg  tea.Msg
	}{
		{"RegionChangedMsg", navmsg.RegionChangedMsg{Regions: []string{"us-west-2"}}},
		{"ProfilesChangedMsg", navmsg.ProfilesChangedMsg{Selections: nil}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := newTestApp(t)
			app.currentView = &MockView{name: "Dashboard"}
			app.modal = &view.Modal{Content: &MockView{name: "TestModal"}}

			app.Update(tt.msg)

			if app.modal != nil {
				t.Errorf("Expected modal closed after %s", tt.name)
			}
		})
	}
}

func TestModalStackPushPop(t *testing.T) {
	app := newTestApp(t)
	app.currentView = &MockView{name: "Dashboard"}

	parentModal := &view.Modal{Content: &MockView{name: "ParentModal"}}
	app.modal = parentModal

	childModal := &view.Modal{Content: &MockView{name: "ChildModal"}}
	app.Update(view.ShowModalMsg{Modal: childModal})

	if app.modal.Content.StatusLine() != "ChildModal" {
		t.Errorf("Expected ChildModal, got %s", app.modal.Content.StatusLine())
	}
	if len(app.modalStack) != 1 {
		t.Errorf("Expected modalStack length 1, got %d", len(app.modalStack))
	}

	app.Update(tea.KeyPressMsg{Code: tea.KeyEscape})

	if app.modal.Content.StatusLine() != "ParentModal" {
		t.Errorf("Expected ParentModal after esc, got %s", app.modal.Content.StatusLine())
	}
	if len(app.modalStack) != 0 {
		t.Errorf("Expected empty modalStack, got %d", len(app.modalStack))
	}

	app.Update(tea.KeyPressMsg{Code: tea.KeyEscape})

	if app.modal != nil {
		t.Error("Expected modal nil after second esc")
	}
}

func TestShowModalFromNormalState(t *testing.T) {
	app := newTestApp(t)
	app.currentView = &MockView{name: "Dashboard"}
	app.modal = nil
	app.modalStack = nil

	modal := &view.Modal{Content: &MockView{name: "TestModal"}}
	app.Update(view.ShowModalMsg{Modal: modal})

	if app.modal == nil {
		t.Error("Expected modal to be set")
	}
	if app.modal.Content.StatusLine() != "TestModal" {
		t.Errorf("Expected TestModal, got %s", app.modal.Content.StatusLine())
	}
	if len(app.modalStack) != 0 {
		t.Errorf("Expected empty modalStack when showing from normal state, got %d", len(app.modalStack))
	}
}

func TestModalStackClearedOnRegionChange(t *testing.T) {
	app := newTestApp(t)
	app.currentView = &MockView{name: "Dashboard"}

	parentModal := &view.Modal{Content: &MockView{name: "ParentModal"}}
	childModal := &view.Modal{Content: &MockView{name: "ChildModal"}}
	app.modal = childModal
	app.modalStack = []*view.Modal{parentModal}

	app.Update(navmsg.RegionChangedMsg{Regions: []string{"us-west-2"}})

	if app.modal != nil {
		t.Error("Expected modal nil after RegionChangedMsg")
	}
	if len(app.modalStack) != 0 {
		t.Errorf("Expected empty modalStack after RegionChangedMsg, got %d", len(app.modalStack))
	}
}

func TestModalStackClearedOnProfileChange(t *testing.T) {
	app := newTestApp(t)
	app.currentView = &MockView{name: "Dashboard"}

	parentModal := &view.Modal{Content: &MockView{name: "ParentModal"}}
	childModal := &view.Modal{Content: &MockView{name: "ChildModal"}}
	app.modal = childModal
	app.modalStack = []*view.Modal{parentModal}

	app.Update(navmsg.ProfilesChangedMsg{Selections: nil})

	if app.modal != nil {
		t.Error("Expected modal nil after ProfilesChangedMsg")
	}
	if len(app.modalStack) != 0 {
		t.Errorf("Expected empty modalStack after ProfilesChangedMsg, got %d", len(app.modalStack))
	}
}

func TestWarningScreenDismissal(t *testing.T) {
	tests := []struct {
		name string
		key  tea.KeyPressMsg
	}{
		{"enter", tea.KeyPressMsg{Code: tea.KeyEnter}},
		{"space", tea.KeyPressMsg{Code: tea.KeySpace}},
		{"q", tea.KeyPressMsg{Code: 0, Text: "q"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := newTestApp(t)
			app.showWarnings = true
			app.warningsReady = true

			app.Update(tt.key)

			if app.showWarnings {
				t.Errorf("Expected showWarnings=false after %s key", tt.name)
			}
		})
	}
}

func TestProfileChangeStaysOnCurrentRefreshableView(t *testing.T) {
	app := newTestApp(t)

	dashboard := &RefreshableMockView{MockView: MockView{name: "Dashboard"}, canRefresh: true}
	resourceBrowser := &RefreshableMockView{MockView: MockView{name: "ResourceBrowser"}, canRefresh: true}

	app.viewStack = []view.View{dashboard}
	app.currentView = resourceBrowser

	app.Update(navmsg.ProfilesChangedMsg{Selections: nil})

	if app.currentView != resourceBrowser {
		t.Errorf("Expected to stay on ResourceBrowser, got %T", app.currentView)
	}
	if len(app.viewStack) != 1 {
		t.Errorf("Expected viewStack length 1, got %d", len(app.viewStack))
	}
}

func TestRegionChangeStaysOnCurrentRefreshableView(t *testing.T) {
	app := newTestApp(t)

	dashboard := &RefreshableMockView{MockView: MockView{name: "Dashboard"}, canRefresh: true}
	resourceBrowser := &RefreshableMockView{MockView: MockView{name: "ResourceBrowser"}, canRefresh: true}

	app.viewStack = []view.View{dashboard}
	app.currentView = resourceBrowser

	app.Update(navmsg.RegionChangedMsg{Regions: []string{"us-east-1"}})

	if app.currentView != resourceBrowser {
		t.Errorf("Expected to stay on ResourceBrowser, got %T", app.currentView)
	}
	if len(app.viewStack) != 1 {
		t.Errorf("Expected viewStack length 1, got %d", len(app.viewStack))
	}
}

func TestProfileChangeFromNonRefreshableViewStaysOnCurrentView(t *testing.T) {
	app := newTestApp(t)

	dashboard := &RefreshableMockView{MockView: MockView{name: "Dashboard"}, canRefresh: true}
	resourceBrowser := &RefreshableMockView{MockView: MockView{name: "ResourceBrowser"}, canRefresh: true}
	detailView := &MockView{name: "DetailView"}

	app.viewStack = []view.View{dashboard, resourceBrowser}
	app.currentView = detailView

	app.Update(navmsg.ProfilesChangedMsg{Selections: nil})

	if app.currentView != detailView {
		t.Errorf("Expected to stay on DetailView, got %T", app.currentView)
	}
	if len(app.viewStack) != 2 {
		t.Errorf("Expected viewStack length 2, got %d", len(app.viewStack))
	}
}

func TestRegionChangeFromNonRefreshableViewStaysOnCurrentView(t *testing.T) {
	app := newTestApp(t)

	dashboard := &RefreshableMockView{MockView: MockView{name: "Dashboard"}, canRefresh: true}
	resourceBrowser := &RefreshableMockView{MockView: MockView{name: "ResourceBrowser"}, canRefresh: true}
	detailView := &MockView{name: "DetailView"}

	app.viewStack = []view.View{dashboard, resourceBrowser}
	app.currentView = detailView

	app.Update(navmsg.RegionChangedMsg{Regions: []string{"us-west-2"}})

	if app.currentView != detailView {
		t.Errorf("Expected to stay on DetailView, got %T", app.currentView)
	}
	if len(app.viewStack) != 2 {
		t.Errorf("Expected viewStack length 2, got %d", len(app.viewStack))
	}
}

func TestNavigateBackWithEmptyStack(t *testing.T) {
	app := newTestApp(t)
	app.currentView = &MockView{name: "Dashboard"}
	app.viewStack = nil

	cmd := app.navigateBack()

	if cmd != nil {
		t.Error("Expected nil cmd when stack is empty")
	}
	if app.currentView.StatusLine() != "Dashboard" {
		t.Errorf("Expected currentView unchanged, got %s", app.currentView.StatusLine())
	}
}

func TestRefreshCurrentViewWithNilView(t *testing.T) {
	app := newTestApp(t)
	app.currentView = nil

	_, cmd := app.refreshCurrentView()

	if cmd != nil {
		t.Error("Expected nil cmd when currentView is nil")
	}
}

func TestRefreshCurrentViewSendsRefreshMsgForRefreshableView(t *testing.T) {
	app := newTestApp(t)
	app.currentView = &RefreshableMockView{MockView: MockView{name: "ResourceBrowser"}, canRefresh: true}

	_, cmd := app.refreshCurrentView()

	if cmd == nil {
		t.Fatal("Expected non-nil cmd for refreshable view")
	}
}

func TestRefreshCurrentViewKeepsNonRefreshableViewUnchanged(t *testing.T) {
	app := newTestApp(t)
	nonRefreshable := &MockView{name: "DetailView"}
	app.currentView = nonRefreshable

	_, _ = app.refreshCurrentView()

	if app.currentView != nonRefreshable {
		t.Errorf("Expected currentView unchanged, got %T", app.currentView)
	}
}
