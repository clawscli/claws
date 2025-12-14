package view

import (
	"context"
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/registry"
	"github.com/clawscli/claws/internal/render"
)

// DefaultAutoReloadInterval is the default interval for auto-reload
const DefaultAutoReloadInterval = 3 * time.Second

// View is the interface for all views in the application
type View interface {
	tea.Model

	// SetSize updates the view dimensions
	SetSize(width, height int) tea.Cmd

	// StatusLine returns the status line text for this view
	StatusLine() string
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

// Refreshable is an interface for views that can refresh their data
// Views like ResourceBrowser implement this, while DetailView does not
type Refreshable interface {
	View
	// CanRefresh returns true if this view can meaningfully refresh its data
	CanRefresh() bool
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
