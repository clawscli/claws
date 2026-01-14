package view

import (
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/clawscli/claws/internal/ui"
)

// HelpView shows keybindings and help information
// helpViewStyles holds cached lipgloss styles for performance
type helpViewStyles struct {
	title   lipgloss.Style
	section lipgloss.Style
	key     lipgloss.Style
	desc    lipgloss.Style
}

func newHelpViewStyles() helpViewStyles {
	return helpViewStyles{
		title:   ui.TitleStyle(),
		section: ui.SectionStyle().MarginTop(1),
		key:     ui.SuccessStyle().Width(15),
		desc:    ui.TextStyle(),
	}
}

type HelpView struct {
	styles helpViewStyles
	vp     ViewportState
}

// NewHelpView creates a new HelpView
func NewHelpView() *HelpView {
	return &HelpView{
		styles: newHelpViewStyles(),
	}
}

// Init implements tea.Model
func (h *HelpView) Init() tea.Cmd {
	return nil
}

func (h *HelpView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg.(type) {
	case ThemeChangedMsg:
		h.styles = newHelpViewStyles()
		if h.vp.Ready {
			h.vp.Model.SetContent(h.renderContent())
		}
		return h, nil
	}
	var cmd tea.Cmd
	h.vp.Model, cmd = h.vp.Model.Update(msg)
	return h, cmd
}

// renderContent returns the help content for the viewport
func (h *HelpView) renderContent() string {
	s := h.styles

	var out string
	out += s.title.Render("claws - AWS TUI") + "\n\n"

	// Navigation
	out += s.section.Render("Navigation") + "\n"
	out += s.key.Render("↑/k, ↓/j") + s.desc.Render("Move cursor up/down") + "\n"
	out += s.key.Render("Enter/d") + s.desc.Render("View details / select") + "\n"
	out += s.key.Render("Esc") + s.desc.Render("Go back / cancel") + "\n"
	out += s.key.Render("q") + s.desc.Render("Quit") + "\n"

	// Service Browser
	out += "\n" + s.section.Render("Service Browser") + "\n"
	out += s.key.Render("←/h, →/l") + s.desc.Render("Move within category") + "\n"
	out += s.key.Render("↑/k, ↓/j") + s.desc.Render("Move between categories") + "\n"
	out += s.key.Render("~") + s.desc.Render("Toggle Dashboard ↔ Services") + "\n"
	out += s.key.Render("/") + s.desc.Render("Filter services") + "\n"

	// Resource Browser
	out += "\n" + s.section.Render("Resource Browser") + "\n"
	out += s.key.Render("Tab") + s.desc.Render("Next resource type") + "\n"
	out += s.key.Render("Shift+Tab") + s.desc.Render("Previous resource type") + "\n"
	out += s.key.Render("1-9") + s.desc.Render("Switch to resource type") + "\n"
	out += s.key.Render("/") + s.desc.Render("Filter resources") + "\n"
	out += s.key.Render("c") + s.desc.Render("Clear filter") + "\n"
	out += s.key.Render("Ctrl+r") + s.desc.Render("Refresh resources") + "\n"
	out += s.key.Render("a") + s.desc.Render("Show actions menu") + "\n"
	out += s.key.Render("y") + s.desc.Render("Copy resource ID to clipboard") + "\n"
	out += s.key.Render("Y") + s.desc.Render("Copy resource ARN to clipboard") + "\n"

	// Filter Syntax
	out += "\n" + s.section.Render("Filter Syntax") + "\n"
	out += s.key.Render("/text") + s.desc.Render("Fuzzy search in all columns") + "\n"

	// Command Mode
	out += "\n" + s.section.Render("Command Mode") + "\n"
	out += s.key.Render(":") + s.desc.Render("Enter command mode") + "\n"
	out += s.key.Render(": + Enter") + s.desc.Render("Go to services") + "\n"
	out += s.key.Render(":home") + s.desc.Render("Go to services") + "\n"
	out += s.key.Render(":pulse") + s.desc.Render("Go to dashboard") + "\n"
	out += s.key.Render(":dashboard") + s.desc.Render("Go to dashboard") + "\n"
	out += s.key.Render(":services") + s.desc.Render("Go to services") + "\n"
	out += s.key.Render(":clear-history") + s.desc.Render("Clear navigation history") + "\n"
	out += s.key.Render("Tab") + s.desc.Render("Cycle through suggestions") + "\n"
	out += s.key.Render("Shift+Tab") + s.desc.Render("Cycle backward") + "\n"
	out += s.key.Render("Enter") + s.desc.Render("Execute command") + "\n"
	out += s.key.Render(":q") + s.desc.Render("Quit") + "\n"
	out += s.key.Render(":login") + s.desc.Render("AWS Console login (claws-login profile)") + "\n"
	out += s.key.Render(":login <name>") + s.desc.Render("AWS Console login with profile") + "\n"
	out += s.key.Render(":theme <name>") + s.desc.Render("Change theme (dark/light/nord/dracula/...)") + "\n"
	out += s.key.Render(":autosave") + s.desc.Render("Toggle config persistence (on/off)") + "\n"
	out += s.key.Render(":settings") + s.desc.Render("Show current settings") + "\n"

	// Tag Commands
	out += "\n" + s.section.Render("Tag Commands") + "\n"
	out += s.key.Render(":tag key=val") + s.desc.Render("Filter current view by tag (exact)") + "\n"
	out += s.key.Render(":tag key") + s.desc.Render("Filter by tag key exists") + "\n"
	out += s.key.Render(":tag key~val") + s.desc.Render("Filter by tag (partial match)") + "\n"
	out += s.key.Render(":tag") + s.desc.Render("Clear tag filter") + "\n"
	out += s.key.Render(":tags") + s.desc.Render("Browse all tagged resources") + "\n"
	out += s.key.Render(":tags Env=prod") + s.desc.Render("Browse with tag filter") + "\n"

	// Diff Commands
	out += "\n" + s.section.Render("Compare Resources") + "\n"
	out += s.key.Render("m") + s.desc.Render("Mark resource for comparison") + "\n"
	out += s.key.Render("d") + s.desc.Render("Compare with marked resource (or view detail)") + "\n"
	out += s.key.Render(":diff name") + s.desc.Render("Compare current row with named resource") + "\n"
	out += s.key.Render(":diff a b") + s.desc.Render("Compare two named resources") + "\n"

	// Actions
	out += "\n" + s.section.Render("Actions (EC2 Instances)") + "\n"
	out += s.key.Render("x") + s.desc.Render("SSM Session") + "\n"
	out += s.key.Render("s") + s.desc.Render("SSH") + "\n"
	out += s.key.Render("S") + s.desc.Render("Stop instance") + "\n"
	out += s.key.Render("R") + s.desc.Render("Start instance") + "\n"
	out += s.key.Render("T") + s.desc.Render("Terminate instance (dangerous)") + "\n"

	// Navigation shortcuts
	out += "\n" + s.section.Render("Resource Navigation") + "\n"
	out += ui.DimStyle().Italic(true).
		Render("  Navigation shortcuts are shown in the status line.\n  They change based on the current resource type.\n") + "\n"
	out += s.key.Render("VPC") + s.desc.Render("s:Subnets t:RouteTables i:IGWs n:NATGWs g:SGs e:Instances") + "\n"
	out += s.key.Render("Subnet") + s.desc.Render("v:VPC e:Instances") + "\n"
	out += s.key.Render("Instance") + s.desc.Render("v:VPC u:Subnet g:SecurityGroups") + "\n"
	out += s.key.Render("SecurityGroup") + s.desc.Render("v:VPC e:Instances") + "\n"

	// Global
	out += "\n" + s.section.Render("Global") + "\n"
	out += s.key.Render("R") + s.desc.Render("Switch AWS region") + "\n"
	out += s.key.Render("P") + s.desc.Render("Switch AWS profile") + "\n"
	out += s.key.Render("Ctrl+E") + s.desc.Render("Toggle compact header") + "\n"
	out += s.key.Render("?") + s.desc.Render("Show this help") + "\n"

	// Command examples
	out += "\n" + s.section.Render("Command Examples") + "\n"
	out += ui.DimStyle().Render(
		"  :ec2             → EC2 instances\n" +
			"  :ec2/volumes     → EC2 volumes\n" +
			"  :s3              → S3 buckets\n" +
			"  :ec2/sec         → Auto-completes to ec2/security-groups\n" +
			"  :sort Name       → Sort by Name column\n" +
			"  :tag Env=prod    → Filter current view by tag\n" +
			"  :tags Env=prod   → Browse all resources with tag\n" +
			"  :diff my-func    → Compare current row with my-func\n" +
			"  :login           → AWS Console login\n" +
			"  :theme nord      → Switch to Nord theme",
	)

	return out
}

func (h *HelpView) ViewString() string {
	if !h.vp.Ready {
		return LoadingMessage
	}
	return h.vp.Model.View()
}

// View implements tea.Model
func (h *HelpView) View() tea.View {
	return tea.NewView(h.ViewString())
}

func (h *HelpView) SetSize(width, height int) tea.Cmd {
	h.vp.SetSize(width, height)
	h.vp.Model.SetContent(h.renderContent())
	return nil
}

// StatusLine implements View
func (h *HelpView) StatusLine() string {
	return "Help • Press Esc to go back"
}
