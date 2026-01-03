package view

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"

	appaws "github.com/clawscli/claws/internal/aws"
	"github.com/clawscli/claws/internal/ui"
)

const (
	panelCost = iota
	panelOperations
	panelSecurity
	panelOptimization
)

const (
	minPanelWidth  = 30
	minPanelHeight = 6
	panelGap       = 1

	dashboardMaxRecords = 100

	targetCost         = "ce/costs"
	targetOperations   = "health/events"
	targetSecurity     = "securityhub/findings"
	targetOptimization = "trustedadvisor/recommendations"

	costValueWidth     = 9
	costPadding        = 2
	minCostBarWidth    = 8
	minCostNameWidth   = 15
	costNameWidthRatio = 60

	bulletIndentWidth = 4
)

func renderPanel(title, content string, width, height int, t *ui.Theme, hovered bool) string {
	titleStyle := ui.TitleStyle()
	boxHeight := max(height-1, 3)

	borderColor := t.Border
	if hovered {
		borderColor = t.Primary
	}

	borderStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(borderColor).
		Padding(0, 1).
		Width(width).
		Height(boxHeight)

	return lipgloss.JoinVertical(lipgloss.Left,
		titleStyle.Render(title),
		borderStyle.Render(content))
}

func renderBar(value, maxVal float64, width int, t *ui.Theme) string {
	if maxVal <= 0 || width <= 0 {
		return ""
	}
	ratio := min(value/maxVal, 1.0)
	filled := min(max(int(ratio*float64(width)), 0), width)

	barStyle := ui.AccentStyle()
	emptyStyle := ui.MutedStyle()

	return barStyle.Render(strings.Repeat("█", filled)) +
		emptyStyle.Render(strings.Repeat("░", width-filled))
}

func (d *DashboardView) renderCostContent(contentWidth, contentHeight int, t *ui.Theme, focusRow int) string {
	s := d.styles
	var lines []string

	if d.costLoading {
		lines = append(lines, d.spinner.View()+" loading...")
	} else if d.costErr != nil {
		lines = append(lines, s.dim.Render("Cost: N/A"))
	} else {
		lines = append(lines, "MTD: "+appaws.FormatMoney(d.costMTD, ""))

		if len(d.costTop) > 0 {
			maxCost := d.costTop[0].cost
			available := contentWidth - costValueWidth - costPadding
			nameWidth := available * costNameWidthRatio / 100
			barWidth := available - nameWidth
			nameWidth = max(nameWidth, minCostNameWidth)
			barWidth = max(barWidth, minCostBarWidth)
			maxServices := max(contentHeight-2, 3)
			showCount := min(len(d.costTop), maxServices)

			for i := range showCount {
				c := d.costTop[i]
				bar := renderBar(c.cost, maxCost, barWidth, t)
				name := TruncateString(c.service, nameWidth)
				line := fmt.Sprintf("%-*s %s %8.0f", nameWidth, name, bar, c.cost)
				if i == focusRow {
					line = s.highlight.Render(line)
				}
				lines = append(lines, line)
			}
		}

		if d.anomalyLoading {
			lines = append(lines, "Anomalies: "+d.spinner.View())
		} else if d.anomalyErr != nil {
			lines = append(lines, "Anomalies: "+s.dim.Render("N/A"))
		} else if d.anomalyCount > 0 {
			lines = append(lines, "Anomalies: "+s.warning.Render(fmt.Sprintf("%d", d.anomalyCount)))
		} else {
			lines = append(lines, "Anomalies: "+s.success.Render("0"))
		}
	}

	return strings.Join(lines, "\n")
}

func (d *DashboardView) renderOpsContent(contentWidth, contentHeight int, focusRow int) string {
	s := d.styles
	var lines []string
	alarmCount := len(d.alarms)

	if d.alarmLoading {
		lines = append(lines, "Alarms: "+d.spinner.View())
	} else if d.alarmErr != nil {
		lines = append(lines, s.dim.Render("Alarms: N/A"))
	} else if alarmCount > 0 {
		lines = append(lines, s.danger.Render(fmt.Sprintf("Alarms: %d in ALARM", alarmCount)))
		maxShow := min(alarmCount, contentHeight-3)
		for i := range maxShow {
			line := "  " + s.danger.Render("• ") + TruncateString(d.alarms[i].name, contentWidth-bulletIndentWidth)
			if i == focusRow {
				line = s.highlight.Render(line)
			}
			lines = append(lines, line)
		}
	} else {
		lines = append(lines, "Alarms: "+s.success.Render("0 ✓"))
	}

	if d.healthLoading {
		lines = append(lines, "Health: "+d.spinner.View())
	} else if d.healthErr != nil {
		lines = append(lines, s.dim.Render("Health: N/A"))
	} else if len(d.healthItems) > 0 {
		lines = append(lines, s.warning.Render(fmt.Sprintf("Health: %d open", len(d.healthItems))))
		remaining := contentHeight - len(lines) - 1
		maxShow := min(len(d.healthItems), remaining)
		for i := range maxShow {
			h := d.healthItems[i]
			line := "  " + s.warning.Render("• ") + TruncateString(h.service+": "+h.eventType, contentWidth-bulletIndentWidth)
			if alarmCount+i == focusRow {
				line = s.highlight.Render(line)
			}
			lines = append(lines, line)
		}
	} else {
		lines = append(lines, "Health: "+s.success.Render("0 open ✓"))
	}

	return strings.Join(lines, "\n")
}

func (d *DashboardView) renderSecurityContent(contentWidth, contentHeight int, focusRow int) string {
	s := d.styles
	var lines []string

	if d.secLoading {
		lines = append(lines, d.spinner.View()+" loading...")
	} else if d.secErr != nil {
		lines = append(lines, s.dim.Render("Security: N/A"))
	} else if len(d.secItems) > 0 {
		var critical, high int
		for _, item := range d.secItems {
			if item.severity == "CRITICAL" {
				critical++
			} else if item.severity == "HIGH" {
				high++
			}
		}
		if critical > 0 {
			lines = append(lines, s.danger.Render(fmt.Sprintf("Critical: %d 🔴", critical)))
		}
		if high > 0 {
			lines = append(lines, s.warning.Render(fmt.Sprintf("High: %d 🟠", high)))
		}
		maxShow := min(len(d.secItems), contentHeight-len(lines)-1)
		for i := range maxShow {
			item := d.secItems[i]
			style := s.warning
			if item.severity == "CRITICAL" {
				style = s.danger
			}
			line := "  " + style.Render("• ") + TruncateString(item.title, contentWidth-bulletIndentWidth)
			if i == focusRow {
				line = s.highlight.Render(line)
			}
			lines = append(lines, line)
		}
	} else {
		lines = append(lines, s.success.Render("No critical/high ✓"))
	}

	return strings.Join(lines, "\n")
}

func (d *DashboardView) renderOptimizationContent(contentWidth, contentHeight int, focusRow int) string {
	s := d.styles
	var lines []string

	if d.taLoading {
		lines = append(lines, d.spinner.View()+" loading...")
	} else if d.taErr != nil {
		lines = append(lines, s.dim.Render("Optimization: N/A"))
	} else {
		var errors, warnings int
		for _, item := range d.taItems {
			if item.status == "error" {
				errors++
			} else {
				warnings++
			}
		}
		if errors > 0 {
			lines = append(lines, s.danger.Render(fmt.Sprintf("Errors: %d", errors)))
		}
		if warnings > 0 {
			lines = append(lines, s.warning.Render(fmt.Sprintf("Warnings: %d", warnings)))
		}
		if d.taSavings > 0 {
			lines = append(lines, s.success.Render("Savings: "+appaws.FormatMoney(d.taSavings, "")+"/mo 💰"))
		}
		if len(d.taItems) > 0 {
			maxShow := min(len(d.taItems), contentHeight-len(lines)-1)
			for i := range maxShow {
				item := d.taItems[i]
				style := s.warning
				if item.status == "error" {
					style = s.danger
				}
				line := "  " + style.Render("• ") + TruncateString(item.name, contentWidth-bulletIndentWidth)
				if i == focusRow {
					line = s.highlight.Render(line)
				}
				lines = append(lines, line)
			}
		}
		if len(lines) == 0 {
			lines = append(lines, s.success.Render("All good ✓"))
		}
	}

	return strings.Join(lines, "\n")
}
