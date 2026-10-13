package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/tristan-gscn/repaco/internal/model"
)

func (m Model) View() string {
	width := m.width
	if width < 80 {
		width = 80
	}
	height := m.height
	if height < 24 {
		height = 24
	}

	header := m.renderHeader(width)
	footer := m.renderFooter(width)

	contentHeight := height - lipgloss.Height(header) - lipgloss.Height(footer) - 2
	if contentHeight < 10 {
		contentHeight = 10
	}

	leftWidth := int(float64(width)*0.45) - 2
	rightWidth := width - leftWidth - 4

	leftPane := m.renderListPane(leftWidth, contentHeight)
	rightPane := m.renderDetailPane(rightWidth, contentHeight)

	body := lipgloss.JoinHorizontal(lipgloss.Top, leftPane, rightPane)

	return lipgloss.JoinVertical(lipgloss.Left, header, body, footer)
}

func (m Model) renderHeader(width int) string {
	projectName := "42 Project"
	if m.detection != nil && m.detection.Project.Name != "" {
		projectName = m.detection.Project.Name
	}

	title := HeaderTitleStyle.Render("REPACO") + " " + HeaderSubStyle.Render(projectName)

	badges := fmt.Sprintf("%s %s %s %s %s",
		lipgloss.NewStyle().Foreground(SuccessColor).Bold(true).Render(fmt.Sprintf("✓ %d", m.stats.Passed)),
		lipgloss.NewStyle().Foreground(ErrorColor).Bold(true).Render(fmt.Sprintf("✗ %d", m.stats.Failed)),
		lipgloss.NewStyle().Foreground(CrashColor).Bold(true).Render(fmt.Sprintf("💥 %d", m.stats.Crashed)),
		lipgloss.NewStyle().Foreground(WarningColor).Bold(true).Render(fmt.Sprintf("⏱ %d", m.stats.Timeout)),
		lipgloss.NewStyle().Foreground(InfoColor).Bold(true).Render(fmt.Sprintf("💧 %d", m.stats.Leaked)),
	)

	filterTag := ""
	if m.filterOnlyFailed {
		filterTag = lipgloss.NewStyle().Background(ErrorColor).Foreground(lipgloss.Color("#FFFFFF")).Bold(true).Render(" [FILTER: FAILED ONLY] ")
	}

	gap := width - lipgloss.Width(title) - lipgloss.Width(badges) - lipgloss.Width(filterTag) - 4
	if gap < 1 {
		gap = 1
	}

	topBar := lipgloss.JoinHorizontal(lipgloss.Center, title, strings.Repeat(" ", gap), filterTag, badges)
	statusLine := MutedTextStyle.Render(m.statusMsg)

	return lipgloss.JoinVertical(lipgloss.Left, topBar, statusLine)
}

func (m Model) renderListPane(width, height int) string {
	visible := m.getVisibleResults()
	var b strings.Builder

	if len(visible) == 0 {
		b.WriteString(MutedTextStyle.Render("No tests to display."))
	} else {
		// Pagination/scrolling
		start := 0
		if m.cursor >= height {
			start = m.cursor - height + 1
		}
		end := start + height
		if end > len(visible) {
			end = len(visible)
		}

		for i := start; i < end; i++ {
			r := visible[i]
			icon := getStatusIcon(r.Status)

			line := fmt.Sprintf(" %s %-20s", icon, truncate(r.Case.Name, width-12))
			if i == m.cursor {
				b.WriteString(SelectedRowStyle.Width(width - 2).Render(line))
			} else {
				b.WriteString(NormalRowStyle.Width(width - 2).Render(line))
			}
			b.WriteString("\n")
		}
	}

	style := PanelStyle.Width(width).Height(height)
	if m.activePane == 0 {
		style = ActivePanelStyle.Width(width).Height(height)
	}

	return style.Render(b.String())
}

func (m Model) renderDetailPane(width, height int) string {
	visible := m.getVisibleResults()
	var b strings.Builder

	if len(visible) == 0 || m.cursor >= len(visible) {
		b.WriteString(MutedTextStyle.Render("Select a test on the left pane to view diagnostic details."))
	} else {
		r := visible[m.cursor]

		b.WriteString(DetailHeaderStyle.Render(fmt.Sprintf("TEST: %s", r.Case.Name)) + "\n")
		b.WriteString(MutedTextStyle.Render(fmt.Sprintf("Category: %s | Duration: %v | Status: %s", r.Case.Category, r.Duration, r.Status)) + "\n\n")

		if r.Case.Description != "" {
			b.WriteString(SectionLabelStyle.Render("Objective:") + "\n")
			b.WriteString(r.Case.Description + "\n\n")
		}

		if r.Input != "" {
			b.WriteString(SectionLabelStyle.Render("Input Arguments:") + "\n")
			b.WriteString(r.Input + "\n\n")
		}

		if r.Status != model.StatusPassed && r.Status != model.StatusSkipped {
			if r.Explanation != "" {
				b.WriteString(SectionLabelStyle.Render("Diagnosis & Reason:") + "\n")
				b.WriteString(lipgloss.NewStyle().Foreground(ErrorColor).Render(r.Explanation) + "\n\n")
			}

			if r.Diff != "" {
				b.WriteString(SectionLabelStyle.Render("Difference (Expected vs Actual):") + "\n")
				b.WriteString(r.Diff + "\n\n")
			}

			if r.Advice != "" {
				b.WriteString(SectionLabelStyle.Render("How to Fix:") + "\n")
				b.WriteString(AdviceStyle.Render("💡 "+r.Advice) + "\n\n")
			}

			if r.LeaksReport != "" {
				b.WriteString(SectionLabelStyle.Render("Sanitizer / Memory Trace:") + "\n")
				b.WriteString(r.LeaksReport + "\n\n")
			}

			if r.Trace != "" {
				b.WriteString(SectionLabelStyle.Render("Crash Trace:") + "\n")
				b.WriteString(r.Trace + "\n\n")
			}
		} else {
			b.WriteString(lipgloss.NewStyle().Foreground(SuccessColor).Bold(true).Render("✓ Test passed cleanly with no memory corruption or errors.") + "\n")
		}
	}

	// Apply scroll
	allLines := strings.Split(b.String(), "\n")
	start := m.detailScroll
	if start >= len(allLines) {
		start = 0
	}
	end := start + height
	if end > len(allLines) {
		end = len(allLines)
	}

	body := strings.Join(allLines[start:end], "\n")

	style := PanelStyle.Width(width).Height(height)
	if m.activePane == 1 {
		style = ActivePanelStyle.Width(width).Height(height)
	}

	return style.Render(body)
}

func (m Model) renderFooter(width int) string {
	keys := []string{
		FooterKeyStyle.Render("q") + " " + FooterDescStyle.Render("quit"),
		FooterKeyStyle.Render("↑/↓ j/k") + " " + FooterDescStyle.Render("navigate"),
		FooterKeyStyle.Render("tab") + " " + FooterDescStyle.Render("switch pane"),
		FooterKeyStyle.Render("f") + " " + FooterDescStyle.Render("filter failures"),
	}
	return lipgloss.NewStyle().MarginTop(1).Render(strings.Join(keys, "  •  "))
}

func getStatusIcon(s model.Status) string {
	switch s {
	case model.StatusPassed:
		return lipgloss.NewStyle().Foreground(SuccessColor).Render("✓")
	case model.StatusFailed:
		return lipgloss.NewStyle().Foreground(ErrorColor).Render("✗")
	case model.StatusCrash:
		return lipgloss.NewStyle().Foreground(CrashColor).Render("💥")
	case model.StatusTimeout:
		return lipgloss.NewStyle().Foreground(WarningColor).Render("⏱")
	case model.StatusLeak:
		return lipgloss.NewStyle().Foreground(InfoColor).Render("💧")
	default:
		return "•"
	}
}

func truncate(str string, length int) string {
	if len(str) <= length {
		return str
	}
	if length <= 3 {
		return str[:length]
	}
	return str[:length-3] + "..."
}
