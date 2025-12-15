package view

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
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
	t := ui.Current()
	return helpViewStyles{
		title:   lipgloss.NewStyle().Bold(true).Foreground(t.Primary).MarginBottom(1),
		section: lipgloss.NewStyle().Bold(true).Foreground(t.Secondary).MarginTop(1),
		key:     lipgloss.NewStyle().Foreground(t.Success).Width(15),
		desc:    lipgloss.NewStyle().Foreground(t.Text),
	}
}

type HelpView struct {
	width  int
	height int
	styles helpViewStyles
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

// Update implements tea.Model
func (h *HelpView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	return h, nil
}

// View implements tea.Model
func (h *HelpView) View() string {
	s := h.styles

	var out string
	out += s.title.Render("claws - AWS TUI") + "\n\n"

	// Navigation
	out += s.section.Render("Navigation") + "\n"
	out += s.key.Render("↑/k, ↓/j") + s.desc.Render("Move cursor up/down") + "\n"
	out += s.key.Render("Enter/d") + s.desc.Render("View details / select") + "\n"
	out += s.key.Render("Esc") + s.desc.Render("Go back / cancel") + "\n"
	out += s.key.Render("q") + s.desc.Render("Quit") + "\n"

	// Resource Browser
	out += "\n" + s.section.Render("Resource Browser") + "\n"
	out += s.key.Render("Tab") + s.desc.Render("Next resource type") + "\n"
	out += s.key.Render("Shift+Tab") + s.desc.Render("Previous resource type") + "\n"
	out += s.key.Render("1-9") + s.desc.Render("Switch to resource type") + "\n"
	out += s.key.Render("/") + s.desc.Render("Filter resources") + "\n"
	out += s.key.Render("c") + s.desc.Render("Clear filter") + "\n"
	out += s.key.Render("Ctrl+r") + s.desc.Render("Refresh resources") + "\n"
	out += s.key.Render("a") + s.desc.Render("Show actions menu") + "\n"

	// Filter Syntax
	out += "\n" + s.section.Render("Filter Syntax") + "\n"
	out += s.key.Render("/text") + s.desc.Render("Fuzzy search in all columns") + "\n"

	// Command Mode
	out += "\n" + s.section.Render("Command Mode") + "\n"
	out += s.key.Render(":") + s.desc.Render("Enter command mode") + "\n"
	out += s.key.Render("Tab") + s.desc.Render("Cycle through suggestions") + "\n"
	out += s.key.Render("Shift+Tab") + s.desc.Render("Cycle backward") + "\n"
	out += s.key.Render("Enter") + s.desc.Render("Execute command") + "\n"

	// Tag Commands
	out += "\n" + s.section.Render("Tag Commands") + "\n"
	out += s.key.Render(":tag key=val") + s.desc.Render("Filter current view by tag (exact)") + "\n"
	out += s.key.Render(":tag key") + s.desc.Render("Filter by tag key exists") + "\n"
	out += s.key.Render(":tag key~val") + s.desc.Render("Filter by tag (partial match)") + "\n"
	out += s.key.Render(":tag") + s.desc.Render("Clear tag filter") + "\n"
	out += s.key.Render(":tags") + s.desc.Render("Browse all tagged resources") + "\n"
	out += s.key.Render(":tags Env=prod") + s.desc.Render("Browse with tag filter") + "\n"

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
	out += s.key.Render("?") + s.desc.Render("Show this help") + "\n"

	// Command examples
	out += "\n" + s.section.Render("Command Examples") + "\n"
	out += ui.DimStyle().Render(
		"  :ec2             → EC2 instances\n" +
			"  :ec2/volumes     → EC2 volumes\n" +
			"  :s3              → S3 buckets\n" +
			"  :ec2/sec         → Auto-completes to ec2/security-groups\n" +
			"  :sort Name       → Sort by Name column\n" +
			"  :sort desc Age   → Sort by Age descending\n" +
			"  :tag Env=prod    → Filter current view by tag\n" +
			"  :tags Env=prod   → Browse all resources with tag",
	)

	return out
}

// SetSize implements View
func (h *HelpView) SetSize(width, height int) tea.Cmd {
	h.width = width
	h.height = height
	return nil
}

// StatusLine implements View
func (h *HelpView) StatusLine() string {
	return "Help • Press Esc to go back"
}
