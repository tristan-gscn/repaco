package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/tristan-gscn/repaco/internal/model"
)

func (m Model) View() string {
	width := m.width
	if width < 60 {
		width = 60
	}
	height := m.height
	if height < 12 {
		height = 12
	}

	if m.isDiscovery {
		return m.renderDiscoveryView(width, height)
	}

	header := m.renderHeader(width)
	bottomBar := m.renderBottomBar(width)

	headerH := lipgloss.Height(header)
	bottomH := lipgloss.Height(bottomBar)

	contentH := height - headerH - bottomH
	if contentH < 4 {
		contentH = 4
	}

	leftTotalW := int(float64(width) * 0.35)
	if leftTotalW < 26 {
		leftTotalW = 26
	}
	if leftTotalW > width-30 {
		leftTotalW = width - 30
	}
	rightTotalW := width - leftTotalW

	leftInnerW := leftTotalW - 2
	rightInnerW := rightTotalW - 2
	innerH := contentH - 2

	leftPane := m.renderTestTreePane(leftInnerW, innerH)
	rightPane := m.renderDetailConsolePane(rightInnerW, innerH)

	body := lipgloss.JoinHorizontal(lipgloss.Top, leftPane, rightPane)

	return lipgloss.JoinVertical(lipgloss.Left, header, body, bottomBar)
}

// renderDiscoveryView renders a full-screen, adaptive, interactive discovery UI when no project is detected.
func (m Model) renderDiscoveryView(width, height int) string {
	// 1. Header (1 line)
	title := ProjectTagStyle.Render("REPACO") + " " +
		HeaderStyle.Render("PROJECT DISCOVERY — No 42 Project Detected")

	dirTag := MutedTextStyle.Render("Directory: " + m.targetDir)
	gapH := width - lipgloss.Width(title) - lipgloss.Width(dirTag) - 2
	if gapH < 1 {
		gapH = 1
	}
	topHeader := HeaderStyle.Width(width).Render(
		lipgloss.JoinHorizontal(lipgloss.Center, title, strings.Repeat(" ", gapH), dirTag),
	)

	// 2. Bottom status bar (1 line)
	bottomBar := FooterBarStyle.Width(width).Render(
		MutedTextStyle.Render("Select a project with UP/DOWN to inspect required files."),
	)

	headerH := lipgloss.Height(topHeader)
	bottomH := lipgloss.Height(bottomBar)
	contentH := height - headerH - bottomH
	if contentH < 4 {
		contentH = 4
	}

	leftTotalW := int(float64(width) * 0.35)
	if leftTotalW < 26 {
		leftTotalW = 26
	}
	rightTotalW := width - leftTotalW

	leftInnerW := leftTotalW - 2
	rightInnerW := rightTotalW - 2
	innerH := contentH - 2

	// 3. Left Pane: List of Supported Projects
	var leftB strings.Builder
	leftB.WriteString(" " + SectionLabelStyle.Render("SUPPORTED 42 PROJECTS") + "\n\n")

	for i, p := range m.knownProjects {
		line := fmt.Sprintf("  %s", p.Name)
		if i == m.discoveryCursor {
			rem := leftInnerW - lipgloss.Width(line)
			if rem < 0 {
				rem = 0
			}
			leftB.WriteString(SelectedRowStyle.Render(line+strings.Repeat(" ", rem)) + "\n")
		} else {
			leftB.WriteString(NormalRowStyle.Render(line) + "\n")
		}
	}

	leftFitted := wrapAndFit(leftB.String(), leftInnerW, innerH)
	leftPane := ActivePanelStyle.Width(leftInnerW).Height(innerH).Render(leftFitted)

	// 4. Right Pane: Selected Project Inspection & Missing Files
	var rightB strings.Builder
	if m.discoveryCursor >= 0 && m.discoveryCursor < len(m.knownProjects) {
		selected := m.knownProjects[m.discoveryCursor]

		rightB.WriteString(" " + DetailTitleStyle.Render(selected.Name) + "\n")
		rightB.WriteString(" " + MutedTextStyle.Render(selected.Description) + "\n\n")

		if selected.BinaryName != "" {
			rightB.WriteString(" " + SectionLabelStyle.Render("Expected Target:") + " " + selected.BinaryName + " (Executable)\n\n")
		} else if selected.LibraryName != "" {
			rightB.WriteString(" " + SectionLabelStyle.Render("Expected Target:") + " " + selected.LibraryName + " (Static Library)\n\n")
		}

		rightB.WriteString(" " + SectionLabelStyle.Render("File Signature Checklist in Current Directory:") + "\n")
		for _, req := range selected.RequiredFiles {
			if m.checkFileExists(req) {
				rightB.WriteString(fmt.Sprintf("   %s %s\n", BadgePass.Render("✔"), req))
			} else {
				rightB.WriteString(fmt.Sprintf("   %s %s (missing)\n", BadgeFail.Render("✖"), req))
			}
		}

		if len(selected.BonusFiles) > 0 {
			rightB.WriteString("\n " + SectionLabelStyle.Render("Bonus Files:") + "\n")
			for _, b := range selected.BonusFiles {
				if m.checkFileExists(b) {
					rightB.WriteString(fmt.Sprintf("   %s %s\n", BadgePass.Render("✔"), b))
				} else {
					rightB.WriteString(fmt.Sprintf("   %s %s (not found)\n", MutedTextStyle.Render("○"), b))
				}
			}
		}

		rightB.WriteString("\n " + AdviceStyle.Render("💡 How to run repaco on this project:") + "\n")
		rightB.WriteString(fmt.Sprintf("   1. Ensure you are in the root directory containing the required source files.\n"))
		rightB.WriteString(fmt.Sprintf("   2. Run: repaco /path/to/%s\n", strings.ToLower(selected.Name)))
	}

	rightFitted := wrapAndFit(rightB.String(), rightInnerW, innerH)
	rightPane := PanelStyle.Width(rightInnerW).Height(innerH).Render(rightFitted)

	body := lipgloss.JoinHorizontal(lipgloss.Top, leftPane, rightPane)
	return lipgloss.JoinVertical(lipgloss.Left, topHeader, body, bottomBar)
}

func (m Model) renderHeader(width int) string {
	projectName := "42 Project"
	if m.detection != nil && m.detection.Project.Name != "" {
		projectName = m.detection.Project.Name
	}

	title := ProjectTagStyle.Render("REPACO") + " " + HeaderStyle.Render(projectName)

	badges := fmt.Sprintf("%s  %s  %s  %s  %s",
		BadgePass.Render(fmt.Sprintf("✔ %d", m.stats.Passed)),
		BadgeFail.Render(fmt.Sprintf("✖ %d", m.stats.Failed)),
		BadgeCrash.Render(fmt.Sprintf("💥 %d", m.stats.Crashed)),
		BadgeWarning.Render(fmt.Sprintf("⏱ %d", m.stats.Timeout)),
		BadgeLeak.Render(fmt.Sprintf("💧 %d", m.stats.Leaked)),
	)

	gap := width - lipgloss.Width(title) - lipgloss.Width(badges) - 2
	if gap < 1 {
		gap = 1
	}

	return HeaderStyle.Width(width).Render(
		lipgloss.JoinHorizontal(lipgloss.Center, title, strings.Repeat(" ", gap), badges),
	)
}

func (m Model) renderTestTreePane(innerW, innerH int) string {
	visible := m.results
	var b strings.Builder

	b.WriteString(" " + SectionLabelStyle.Render(fmt.Sprintf("TEST TREE (%d)", len(visible))) + "\n")

	if len(visible) == 0 {
		b.WriteString("\n " + MutedTextStyle.Render("Waiting for test results..."))
	} else {
		start := 0
		if m.cursor >= innerH-1 {
			start = m.cursor - (innerH - 2)
		}
		end := start + (innerH - 1)
		if end > len(visible) {
			end = len(visible)
		}

		maxNameLen := innerW - 6
		if maxNameLen < 6 {
			maxNameLen = 6
		}

		for i := start; i < end; i++ {
			r := visible[i]
			iconRune := getStatusRune(r.Status)
			iconColor := getStatusColor(r.Status)
			name := truncate(r.Case.Name, maxNameLen)

			if i == m.cursor {
				// Style entire line including background behind text and icon without any ANSI reset gap
				iconPart := lipgloss.NewStyle().Background(JBSelection).Foreground(iconColor).Bold(true).Render(iconRune)
				textPart := lipgloss.NewStyle().Background(JBSelection).Foreground(lipgloss.Color("#FFFFFF")).Bold(true).Render(name)
				prefix := lipgloss.NewStyle().Background(JBSelection).Render("  ")
				sep := lipgloss.NewStyle().Background(JBSelection).Render(" ")

				rowContent := prefix + iconPart + sep + textPart
				rem := innerW - lipgloss.Width(rowContent)
				if rem > 0 {
					rowContent += lipgloss.NewStyle().Background(JBSelection).Render(strings.Repeat(" ", rem))
				}
				b.WriteString(rowContent + "\n")
			} else {
				iconPart := lipgloss.NewStyle().Foreground(iconColor).Bold(true).Render(iconRune)
				textPart := NormalRowStyle.Render(name)
				b.WriteString(fmt.Sprintf("  %s %s\n", iconPart, textPart))
			}
		}
	}

	fitted := wrapAndFit(b.String(), innerW, innerH)
	return ActivePanelStyle.Width(innerW).Height(innerH).Render(fitted)
}

func (m Model) renderDetailConsolePane(innerW, innerH int) string {
	visible := m.results
	var b strings.Builder

	if len(visible) == 0 || m.cursor >= len(visible) {
		b.WriteString(" " + MutedTextStyle.Render("Select a test in the tree to inspect full execution details."))
	} else {
		r := visible[m.cursor]

		b.WriteString(" " + DetailTitleStyle.Render(fmt.Sprintf("TEST: %s", r.Case.Name)) + "\n")
		b.WriteString(" " + MutedTextStyle.Render(fmt.Sprintf("Module: %s  |  Category: %s  |  Duration: %v  |  Status: %s",
			r.Case.Module, r.Case.Category, r.Duration, r.Status)) + "\n\n")

		if r.Case.Description != "" {
			b.WriteString(" " + SectionLabelStyle.Render("Objective:") + "\n")
			b.WriteString(" " + r.Case.Description + "\n\n")
		}

		if r.Input != "" {
			b.WriteString(" " + SectionLabelStyle.Render("Inputs & Test Arguments:") + "\n")
			b.WriteString(" " + r.Input + "\n\n")
		}

		if r.Status != model.StatusPassed && r.Status != model.StatusSkipped {
			if r.Explanation != "" {
				b.WriteString(" " + SectionLabelStyle.Render("Diagnosis & Reason:") + "\n")
				b.WriteString(" " + lipgloss.NewStyle().Foreground(JBRed).Render(r.Explanation) + "\n\n")
			}

			if r.Diff != "" {
				b.WriteString(" " + SectionLabelStyle.Render("Comparison Diff (Expected vs Actual):") + "\n")
				b.WriteString(" " + r.Diff + "\n\n")
			}

			if r.Advice != "" {
				b.WriteString(" " + SectionLabelStyle.Render("Fix Recommendation:") + "\n")
				b.WriteString(" " + AdviceStyle.Render("💡 "+r.Advice) + "\n\n")
			}

			if r.LeaksReport != "" {
				b.WriteString(" " + SectionLabelStyle.Render("Sanitizer / Memory Leak Trace:") + "\n")
				b.WriteString(" " + r.LeaksReport + "\n\n")
			}

			if r.Trace != "" {
				b.WriteString(" " + SectionLabelStyle.Render("Crash Stack Trace:") + "\n")
				b.WriteString(" " + r.Trace + "\n\n")
			}
		} else {
			b.WriteString(" " + BadgePass.Render("✔ Test passed cleanly with zero assertions failed and no memory leaks.") + "\n")
		}
	}

	fitted := wrapAndFit(b.String(), innerW, innerH)
	return PanelStyle.Width(innerW).Height(innerH).Render(fitted)
}

// renderBottomBar renders the full-width solid progress bar on line 1, and the info metrics on line 2.
func (m Model) renderBottomBar(width int) string {
	total := len(m.results)
	if m.totalTests > total {
		total = m.totalTests
	}

	var ratio float64
	if total > 0 {
		ratio = float64(len(m.results)) / float64(total)
		if ratio > 1.0 {
			ratio = 1.0
		}
	}

	// JetBrains color behavior: Green as long as all tests pass, immediately turns Red on any failure
	isFailed := m.stats.Failed > 0 || m.stats.Crashed > 0 || m.stats.Leaked > 0
	barColor := JBGreen
	if isFailed {
		barColor = JBRed
	}

	// Line 1: Full-width progress bar spanning 100% of terminal width
	filledLen := int(ratio * float64(width))
	if filledLen > width {
		filledLen = width
	}
	emptyLen := width - filledLen

	filledPart := lipgloss.NewStyle().Background(barColor).Render(strings.Repeat(" ", filledLen))
	emptyPart := lipgloss.NewStyle().Background(lipgloss.Color("#393B40")).Render(strings.Repeat(" ", emptyLen))
	barLine := filledPart + emptyPart

	// Line 2: Info row directly below the progress bar
	availW := width - 2 // FooterBarStyle has Padding(0, 1)

	statsText := fmt.Sprintf("%3.0f%% (%d/%d) • %v",
		ratio*100,
		len(m.results),
		total,
		m.elapsedTime.Round(100*1000),
	)

	statusText := m.statusMsg
	if m.running {
		statusText = fmt.Sprintf("%s %s", m.spinner.View(), m.statusMsg)
	}

	maxStatusLen := availW - lipgloss.Width(statsText) - 2
	if maxStatusLen > 0 {
		statusText = truncate(statusText, maxStatusLen)
	} else {
		statusText = ""
	}

	gap := availW - lipgloss.Width(statsText) - lipgloss.Width(statusText)
	if gap < 1 {
		gap = 1
	}

	infoContent := lipgloss.JoinHorizontal(lipgloss.Center, statsText, strings.Repeat(" ", gap), MutedTextStyle.Render(statusText))
	infoLine := FooterBarStyle.Width(width).Render(infoContent)

	return lipgloss.JoinVertical(lipgloss.Left, barLine, infoLine)
}

// wrapAndFit wraps text to maxW and clamps the result to at most maxH lines.
func wrapAndFit(text string, maxW, maxH int) string {
	if maxW < 1 {
		maxW = 1
	}
	wrapStyle := lipgloss.NewStyle().Width(maxW)
	wrapped := wrapStyle.Render(text)
	lines := strings.Split(wrapped, "\n")
	if len(lines) > maxH {
		lines = lines[:maxH]
	}
	return strings.Join(lines, "\n")
}

func getStatusRune(s model.Status) string {
	switch s {
	case model.StatusPassed:
		return "✔"
	case model.StatusFailed:
		return "✖"
	case model.StatusCrash:
		return "💥"
	case model.StatusTimeout:
		return "⏱"
	case model.StatusLeak:
		return "💧"
	default:
		return "●"
	}
}

func getStatusColor(s model.Status) lipgloss.TerminalColor {
	switch s {
	case model.StatusPassed:
		return JBGreen
	case model.StatusFailed:
		return JBRed
	case model.StatusCrash:
		return lipgloss.Color("#FF4D4D")
	case model.StatusTimeout:
		return JBAmber
	case model.StatusLeak:
		return lipgloss.Color("#4AA8FF")
	default:
		return JBTextMuted
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
