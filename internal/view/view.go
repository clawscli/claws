package view

import (
	"context"
	"fmt"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/registry"
	"github.com/clawscli/claws/internal/render"
)

// DefaultAutoReloadInterval is the default interval for auto-reload
const DefaultAutoReloadInterval = 3 * time.Second

// FilterPlaceholder is the placeholder text for filter inputs
const FilterPlaceholder = "filter..."

// LoadingMessage is the standard message shown while loading
const LoadingMessage = "Loading..."

// View is the interface for all views in the application
type View interface {
	tea.Model

	// SetSize updates the view dimensions
	SetSize(width, height int) tea.Cmd

	// StatusLine returns the status line text for this view
	StatusLine() string

	// ViewString returns the view content as a string (for internal composition)
	ViewString() string
}

// InputCapture is an optional interface for views that capture input
type InputCapture interface {
	// HasActiveInput returns true if the view has active input (filter, search, etc.)
	HasActiveInput() bool
}

// NavigateMsg is sent when navigating to a new view
type NavigateMsg struct {
	View       View
	ClearStack bool // If true, clear the view stack (go home)
}

// ErrorMsg is sent when an error occurs
type ErrorMsg struct {
	Err error
}

// LoadingMsg indicates data is being loaded
type LoadingMsg struct{}

// DataLoadedMsg indicates data has been loaded
type DataLoadedMsg struct {
	Data any
}

// RefreshMsg tells the view to reload its data
type RefreshMsg struct{}

// SortMsg tells the current view to sort by the specified column
type SortMsg struct {
	Column    string // Column name to sort by (empty to clear sort)
	Ascending bool   // Sort direction
}

// TagFilterMsg tells the current view to filter by tags
type TagFilterMsg struct {
	Filter string // Tag filter (e.g., "Env=prod", "Env", "Env~prod")
}

// DiffMsg tells the current view to show diff between resources
// If LeftName is empty, use current cursor row as left resource
type DiffMsg struct {
	LeftName  string // Name of left resource (empty = current row)
	RightName string // Name of right resource
}

// Refreshable is an interface for views that can refresh their data
// Views like ResourceBrowser implement this, while DetailView does not
type Refreshable interface {
	View
	// CanRefresh returns true if this view can meaningfully refresh its data
	CanRefresh() bool
}

// IsEscKey returns true if the key message represents an escape key press.
// This handles various terminal escape sequences consistently across views.
// In v2, we use msg.Code and tea.KeyEscape.
func IsEscKey(msg tea.KeyPressMsg) bool {
	return msg.String() == "esc" || msg.Code == tea.KeyEscape
}

// NavigationHelper provides common navigation functionality
type NavigationHelper struct {
	Ctx      context.Context
	Registry *registry.Registry
	Renderer render.Renderer
}

// FormatShortcuts returns a formatted string of navigation shortcuts
func (h *NavigationHelper) FormatShortcuts(resource dao.Resource) string {
	if h.Renderer == nil {
		return ""
	}

	navigator, ok := h.Renderer.(render.Navigator)
	if !ok {
		return ""
	}

	navigations := navigator.Navigations(resource)
	if len(navigations) == 0 {
		return ""
	}

	var parts []string
	for _, nav := range navigations {
		parts = append(parts, fmt.Sprintf("%s:%s", nav.Key, nav.Label))
	}
	return strings.Join(parts, " ")
}

// HandleKey handles navigation key press and returns a command if navigation occurred
func (h *NavigationHelper) HandleKey(key string, resource dao.Resource) tea.Cmd {
	if h.Renderer == nil || h.Registry == nil {
		return nil
	}

	navigator, ok := h.Renderer.(render.Navigator)
	if !ok {
		return nil
	}

	navigations := navigator.Navigations(resource)
	for _, nav := range navigations {
		if nav.Key == key {
			if nav.ViewType != "" {
				return h.createCustomView(nav, resource)
			}

			var newBrowser *ResourceBrowser
			if nav.AutoReload {
				interval := nav.ReloadInterval
				if interval == 0 {
					interval = DefaultAutoReloadInterval
				}
				newBrowser = NewResourceBrowserWithAutoReload(
					h.Ctx,
					h.Registry,
					nav.Service,
					nav.Resource,
					nav.FilterField,
					nav.FilterValue,
					interval,
				)
			} else {
				newBrowser = NewResourceBrowserWithFilter(
					h.Ctx,
					h.Registry,
					nav.Service,
					nav.Resource,
					nav.FilterField,
					nav.FilterValue,
				)
			}
			return func() tea.Msg {
				return NavigateMsg{View: newBrowser}
			}
		}
	}

	return nil
}

func (h *NavigationHelper) createCustomView(nav render.Navigation, resource dao.Resource) tea.Cmd {
	switch nav.ViewType {
	case render.ViewTypeLogView:
		return h.createLogView(resource)
	default:
		return nil
	}
}

func (h *NavigationHelper) createLogView(resource dao.Resource) tea.Cmd {
	var logView *LogView

	type logGroupProvider interface{ LogGroupName() string }
	type logStreamProvider interface{ LogStreamName() string }
	type lastEventProvider interface{ LastEventTimestamp() int64 }

	unwrapped := dao.UnwrapResource(resource)

	if p, ok := unwrapped.(logGroupProvider); ok {
		logGroupName := p.LogGroupName()
		if sp, ok := unwrapped.(logStreamProvider); ok {
			var lastEvent int64
			if lp, ok := unwrapped.(lastEventProvider); ok {
				lastEvent = lp.LastEventTimestamp()
			}
			logView = NewLogViewWithStream(h.Ctx, logGroupName, sp.LogStreamName(), lastEvent)
		} else {
			logView = NewLogView(h.Ctx, logGroupName)
		}
	} else {
		logView = NewLogView(h.Ctx, unwrapped.GetID())
	}

	return func() tea.Msg {
		return NavigateMsg{View: logView}
	}
}
