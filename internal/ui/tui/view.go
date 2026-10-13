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

	if m.isNearbyMode {
		return m.renderNearbyProjectsView(width, height)
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

// renderNearbyProjectsView renders an interactive project selector when 42 projects are discovered in nearby folders.
func (m Model) renderNearbyProjectsView(width, height int) string {
	// 1. Header (1 line)
	title := ProjectTagStyle.Render("REPACO") + " " +
		HeaderStyle.Render("PROJECTS FOUND NEARBY — Select a project to test")

	dirTag := MutedTextStyle.Render("Location: " + m.targetDir)
	gapH := width - lipgloss.Width(title) - lipgloss.Width(dirTag) - 2
	if gapH < 1 {
		gapH = 1
	}
	topHeader := HeaderStyle.Width(width).Render(
		lipgloss.JoinHorizontal(lipgloss.Center, title, strings.Repeat(" ", gapH), dirTag),
	)

	// 2. Bottom status bar (1 line)
	bottomBar := FooterBarStyle.Width(width).Render(
		MutedTextStyle.Render("Press UP/DOWN to choose a project • Press [Enter] to test • [ESC/q] to quit"),
	)

	headerH := lipgloss.Height(topHeader)
	bottomH := lipgloss.Height(bottomBar)
	contentH := height - headerH - bottomH
	if contentH < 4 {
		contentH = 4
	}

	leftTotalW := int(float64(width) * 0.35)
	if leftTotalW < 28 {
		leftTotalW = 28
	}
	rightTotalW := width - leftTotalW

	leftInnerW := leftTotalW - 2
	rightInnerW := rightTotalW - 2
	innerH := contentH - 2

	// 3. Left Pane: Discovered Projects List
	var leftB strings.Builder
	leftB.WriteString(" " + SectionLabelStyle.Render(fmt.Sprintf("DISCOVERED PROJECTS (%d)", len(m.nearbyProjects))) + "\n\n")

	for i, p := range m.nearbyProjects {
		line := fmt.Sprintf("  📁 %-14s [%s]", truncate(p.RelPath, leftInnerW-18), p.Detection.Project.Name)
		if i == m.nearbyCursor {
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

	// 4. Right Pane: Detailed Inspection of selected nearby project
	var rightB strings.Builder
	if m.nearbyCursor >= 0 && m.nearbyCursor < len(m.nearbyProjects) {
		selected := m.nearbyProjects[m.nearbyCursor]

		rightB.WriteString(" " + DetailTitleStyle.Render(selected.Detection.Project.Name) + "\n")
		rightB.WriteString(" " + MutedTextStyle.Render(selected.Detection.Project.Description) + "\n\n")

		rightB.WriteString(" " + SectionLabelStyle.Render("Target Location:") + "\n")
		rightB.WriteString(fmt.Sprintf("   Path: %s\n", selected.RelPath))
		rightB.WriteString(fmt.Sprintf("   Full: %s\n\n", selected.AbsPath))

		rightB.WriteString(" " + SectionLabelStyle.Render("Configuration & Detection:") + "\n")
		rightB.WriteString(fmt.Sprintf("   Confidence: %.0f%%\n", selected.Detection.Confidence*100))
		if selected.Detection.HasMakefile {
			rightB.WriteString("   Makefile  : ✔ Found\n")
		} else {
			rightB.WriteString("   Makefile  : ○ None required\n")
		}
		if selected.Detection.HasBonus {
			rightB.WriteString("   Bonuses   : ✔ Bonus files present\n")
		}

		if len(selected.Detection.FoundFiles) > 0 {
			rightB.WriteString("\n " + SectionLabelStyle.Render("Key Files Identified:") + "\n")
			limit := 8
			if len(selected.Detection.FoundFiles) < limit {
				limit = len(selected.Detection.FoundFiles)
			}
			for i := 0; i < limit; i++ {
				rightB.WriteString(fmt.Sprintf("   %s %s\n", BadgePass.Render("✔"), selected.Detection.FoundFiles[i]))
			}
			if len(selected.Detection.FoundFiles) > limit {
				rightB.WriteString(fmt.Sprintf("   ... and %d more files\n", len(selected.Detection.FoundFiles)-limit))
			}
		}

		rightB.WriteString("\n " + AdviceStyle.Render("👉 Press [Enter] to run tests on this project now"))
	}

	rightFitted := wrapAndFit(rightB.String(), rightInnerW, innerH)
	rightPane := PanelStyle.Width(rightInnerW).Height(innerH).Render(rightFitted)

	body := lipgloss.JoinHorizontal(lipgloss.Top, leftPane, rightPane)
	return lipgloss.JoinVertical(lipgloss.Left, topHeader, body, bottomBar)
}

// renderDiscoveryView renders a full-screen, adaptive, interactive discovery UI when no project is detected anywhere.
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
	treeRows := m.buildTreeRows()
	var b strings.Builder

	b.WriteString(" " + SectionLabelStyle.Render(fmt.Sprintf("TEST TREE (%d)", len(m.results))) + "\n")

	if len(treeRows) == 0 {
		b.WriteString("\n " + MutedTextStyle.Render("Waiting for test results..."))
	} else {
		cursor := m.cursor
		if cursor >= len(treeRows) {
			cursor = len(treeRows) - 1
		}
		if cursor < 0 {
			cursor = 0
		}

		availH := innerH - 1
		start := 0
		if cursor >= availH {
			start = cursor - (availH - 1)
		}
		end := start + availH
		if end > len(treeRows) {
			end = len(treeRows)
		}

		for i := start; i < end; i++ {
			row := treeRows[i]
			isSelected := (i == cursor)

			if row.IsGroup {
				arrow := "▼"
				if !row.Expanded {
					arrow = "▶"
				}
				statusRune := getStatusRune(row.GroupStatus)
				statusColor := getStatusColor(row.GroupStatus)
				counterStr := fmt.Sprintf("(%d/%d)", row.PassedCount, row.TotalCount)

				maxFnLen := innerW - 6 - len(counterStr) - 2
				if maxFnLen < 4 {
					maxFnLen = 4
				}
				fileName := truncate(row.File, maxFnLen)

				if isSelected {
					arrowPart := lipgloss.NewStyle().Background(JBSelection).Foreground(JBTextMuted).Render("  " + arrow + " ")
					iconPart := lipgloss.NewStyle().Background(JBSelection).Foreground(statusColor).Bold(true).Render(statusRune + " ")
					namePart := lipgloss.NewStyle().Background(JBSelection).Foreground(lipgloss.Color("#FFFFFF")).Bold(true).Render(fileName)
					countPart := lipgloss.NewStyle().Background(JBSelection).Foreground(JBTextMuted).Render(" " + counterStr)

					rowContent := arrowPart + iconPart + namePart + countPart
					rem := innerW - lipgloss.Width(rowContent)
					if rem > 0 {
						rowContent += lipgloss.NewStyle().Background(JBSelection).Render(strings.Repeat(" ", rem))
					}
					b.WriteString(rowContent + "\n")
				} else {
					arrowPart := MutedTextStyle.Render("  " + arrow + " ")
					iconPart := lipgloss.NewStyle().Foreground(statusColor).Bold(true).Render(statusRune + " ")
					namePart := lipgloss.NewStyle().Foreground(JBTextPrimary).Bold(true).Render(fileName)
					countPart := MutedTextStyle.Render(" " + counterStr)
					b.WriteString(arrowPart + iconPart + namePart + countPart + "\n")
				}
			} else {
				statusRune := getStatusRune(row.Result.Status)
				statusColor := getStatusColor(row.Result.Status)

				maxNameLen := innerW - 9
				if maxNameLen < 6 {
					maxNameLen = 6
				}
				testName := truncate(row.Result.Case.Name, maxNameLen)

				if isSelected {
					prefix := lipgloss.NewStyle().Background(JBSelection).Render("      ")
					iconPart := lipgloss.NewStyle().Background(JBSelection).Foreground(statusColor).Bold(true).Render(statusRune + " ")
					namePart := lipgloss.NewStyle().Background(JBSelection).Foreground(lipgloss.Color("#FFFFFF")).Bold(true).Render(testName)

					rowContent := prefix + iconPart + namePart
					rem := innerW - lipgloss.Width(rowContent)
					if rem > 0 {
						rowContent += lipgloss.NewStyle().Background(JBSelection).Render(strings.Repeat(" ", rem))
					}
					b.WriteString(rowContent + "\n")
				} else {
					iconPart := lipgloss.NewStyle().Foreground(statusColor).Bold(true).Render(statusRune + " ")
					namePart := NormalRowStyle.Render(testName)
					b.WriteString("      " + iconPart + namePart + "\n")
				}
			}
		}
	}

	fitted := wrapAndFit(b.String(), innerW, innerH)
	panelStyle := ActivePanelStyle
	if m.focusRight {
		panelStyle = PanelStyle
	}
	return panelStyle.Width(innerW).Height(innerH).Render(fitted)
}

func (m Model) renderDetailConsolePane(innerW, innerH int) string {
	treeRows := m.buildTreeRows()
	var b strings.Builder

	if len(treeRows) == 0 || m.cursor >= len(treeRows) {
		b.WriteString(" " + MutedTextStyle.Render("Select a test in the tree to inspect full execution details."))
	} else {
		row := treeRows[m.cursor]
		if row.IsGroup {
			titleStr := fmt.Sprintf("FILE: %s", row.File)
			if m.focusRight {
				b.WriteString(" " + DetailTitleStyle.Render(titleStr) + "  " + AdviceStyle.Render("[SCROLLING: ▲/▼ • LEFT to return]") + "\n")
			} else {
				b.WriteString(" " + DetailTitleStyle.Render(titleStr) + "  " + MutedTextStyle.Render("[RIGHT to scroll]") + "\n")
			}

			cat := "General"
			if len(row.SubResults) > 0 {
				cat = string(row.SubResults[0].Case.Category)
			}
			b.WriteString(" " + MutedTextStyle.Render(fmt.Sprintf("Category: %s  |  Total Sub-Tests: %d  |  Status: %s",
				cat, row.TotalCount, row.GroupStatus)) + "\n\n")

			if row.PassedCount == row.TotalCount {
				b.WriteString(" " + BadgePass.Render(fmt.Sprintf("✔ All %d sub-tests passed cleanly (0 leaks, 0 crashes)", row.TotalCount)) + "\n\n")
			} else {
				b.WriteString(" " + BadgeFail.Render(fmt.Sprintf("✖ %d of %d sub-tests failed", row.TotalCount-row.PassedCount, row.TotalCount)) + "\n\n")
			}

			b.WriteString(" " + SectionLabelStyle.Render("Sub-tests in this file:") + "\n")
			for _, sub := range row.SubResults {
				sRune := getStatusRune(sub.Status)
				sColor := getStatusColor(sub.Status)
				sIcon := lipgloss.NewStyle().Foreground(sColor).Bold(true).Render(sRune)
				b.WriteString(fmt.Sprintf("   %s %s (%v)\n", sIcon, sub.Case.Name, sub.Duration))
				if sub.Status != model.StatusPassed && sub.Explanation != "" {
					b.WriteString("       " + lipgloss.NewStyle().Foreground(JBRed).Render("→ "+sub.Explanation) + "\n")
				}
			}

			b.WriteString("\n " + AdviceStyle.Render("💡 Press [Space] or [Enter] to collapse/expand • Press [Down] to inspect each sub-test in detail"))
		} else {
			r := row.Result

			titleStr := fmt.Sprintf("TEST: %s", r.Case.Name)
			if m.focusRight {
				b.WriteString(" " + DetailTitleStyle.Render(titleStr) + "  " + AdviceStyle.Render("[SCROLLING: ▲/▼ • LEFT to return]") + "\n")
			} else {
				b.WriteString(" " + DetailTitleStyle.Render(titleStr) + "  " + MutedTextStyle.Render("[RIGHT to scroll]") + "\n")
			}
			b.WriteString(" " + MutedTextStyle.Render(fmt.Sprintf("File: %s  |  Module: %s  |  Duration: %v  |  Status: %s",
				row.File, r.Case.Module, r.Duration, r.Status)) + "\n\n")

			// Status Badge
			if r.Status == model.StatusPassed {
				b.WriteString(" " + BadgePass.Render("✔ Test passed cleanly (all assertions verified, 0 leaks)") + "\n\n")
			} else if r.Status == model.StatusFailed {
				b.WriteString(" " + BadgeFail.Render("✖ Assertion failure detected") + "\n\n")
			} else if r.Status == model.StatusCrash {
				b.WriteString(" " + BadgeCrash.Render("💥 Process crashed with signal: "+r.Signal) + "\n\n")
			} else if r.Status == model.StatusTimeout {
				b.WriteString(" " + BadgeWarning.Render("⏱ Process timed out (infinite loop)") + "\n\n")
			} else if r.Status == model.StatusLeak {
				b.WriteString(" " + BadgeLeak.Render("💧 AddressSanitizer detected memory leak / corruption") + "\n\n")
			}

			if r.Case.Description != "" {
				b.WriteString(" " + SectionLabelStyle.Render("Objective:") + "\n")
				b.WriteString("   " + r.Case.Description + "\n\n")
			}

			if r.Input != "" {
				b.WriteString(" " + SectionLabelStyle.Render("Inputs & Scenarios Tested:") + "\n")
				for _, line := range strings.Split(r.Input, "\n") {
					b.WriteString("   " + line + "\n")
				}
				b.WriteString("\n")
			}

			if r.Expected != "" {
				b.WriteString(" " + SectionLabelStyle.Render("Expected Behavior / Output:") + "\n")
				for _, line := range strings.Split(r.Expected, "\n") {
					b.WriteString("   " + DiffExpectedStyle.Render(line) + "\n")
				}
				b.WriteString("\n")
			}

			if r.Actual != "" {
				b.WriteString(" " + SectionLabelStyle.Render("Actual Output Observed:") + "\n")
				actStyle := DiffExpectedStyle
				if r.Status != model.StatusPassed {
					actStyle = DiffActualStyle
				}
				for _, line := range strings.Split(r.Actual, "\n") {
					b.WriteString("   " + actStyle.Render(line) + "\n")
				}
				b.WriteString("\n")
			}

			if r.CodeSnippet != "" {
				b.WriteString(" " + SectionLabelStyle.Render("Reproduce at home (main.c snippet):") + "\n")
				lines := strings.Split(strings.TrimRight(r.CodeSnippet, "\n"), "\n")
				for _, line := range lines {
					b.WriteString("   " + CodeSnippetStyle.Render(line) + "\n")
				}
				b.WriteString("\n")
			}

			if r.Status != model.StatusPassed && r.Status != model.StatusSkipped {
				if r.Explanation != "" {
					b.WriteString(" " + SectionLabelStyle.Render("Diagnosis & Reason:") + "\n")
					b.WriteString("   " + lipgloss.NewStyle().Foreground(JBRed).Render(r.Explanation) + "\n\n")
				}

				if r.Diff != "" {
					b.WriteString(" " + SectionLabelStyle.Render("Comparison Diff (Expected vs Actual):") + "\n")
					b.WriteString(" " + r.Diff + "\n\n")
				}

				if r.Advice != "" {
					b.WriteString(" " + SectionLabelStyle.Render("Fix Recommendation:") + "\n")
					b.WriteString("   " + AdviceStyle.Render("💡 "+r.Advice) + "\n\n")
				}

				if r.LeaksReport != "" {
					b.WriteString(" " + SectionLabelStyle.Render("Sanitizer / Memory Leak Trace:") + "\n")
					b.WriteString("   " + r.LeaksReport + "\n\n")
				}

				if r.Trace != "" {
					b.WriteString(" " + SectionLabelStyle.Render("Crash Stack Trace:") + "\n")
					b.WriteString("   " + r.Trace + "\n\n")
				}
			}
		}
	}

	fitted, _, _ := wrapAndScroll(b.String(), innerW, innerH, m.detailScroll)
	panelStyle := PanelStyle
	if m.focusRight {
		panelStyle = ActivePanelStyle
	}
	return panelStyle.Width(innerW).Height(innerH).Render(fitted)
}

// renderBottomBar renders the full-width segmented progress bar on line 1, and the info metrics on line 2.
func (m Model) renderBottomBar(width int) string {
	total := len(m.results)
	if m.totalTests > total {
		total = m.totalTests
	}

	failedCount := m.stats.Failed + m.stats.Crashed + m.stats.Leaked + m.stats.Timeout

	var greenLen, redLen, emptyLen int
	var passPercent float64

	if total > 0 {
		passPercent = (float64(m.stats.Passed) / float64(total)) * 100.0
		if m.stats.Passed < total && passPercent > 99.0 {
			passPercent = 99.0
		}
		if m.stats.Passed > 0 && passPercent < 1.0 {
			passPercent = 1.0
		}

		if m.stats.Passed == total {
			// 100% Passed -> Pure Green across the entire bar
			greenLen = width
			redLen = 0
			emptyLen = 0
		} else if failedCount == total {
			// 0% Passed (100% Failed) -> Pure Red across the entire bar
			greenLen = 0
			redLen = width
			emptyLen = 0
		} else if !m.running || len(m.results) >= total {
			// Finished suite: Green on the left, Red on the right, fully covering width
			greenLen = int(float64(m.stats.Passed)/float64(total)*float64(width) + 0.5)
			if m.stats.Passed > 0 && greenLen == 0 {
				greenLen = 1
			}
			if failedCount > 0 && greenLen >= width {
				greenLen = width - 1
			}
			redLen = width - greenLen
			emptyLen = 0
		} else {
			// Still running: Green on the left, Red next, and pending tests in gray background
			greenLen = int(float64(m.stats.Passed) / float64(total) * float64(width))
			redLen = int(float64(failedCount) / float64(total) * float64(width))
			if m.stats.Passed > 0 && greenLen == 0 {
				greenLen = 1
			}
			if failedCount > 0 && redLen == 0 {
				redLen = 1
			}
			if greenLen+redLen > width {
				redLen = width - greenLen
			}
			emptyLen = width - greenLen - redLen
			if emptyLen < 0 {
				emptyLen = 0
			}
		}
	} else {
		emptyLen = width
	}

	var greenPart, redPart, emptyPart string
	if greenLen > 0 {
		greenPart = lipgloss.NewStyle().Background(JBGreen).Render(strings.Repeat(" ", greenLen))
	}
	if redLen > 0 {
		redPart = lipgloss.NewStyle().Background(JBRed).Render(strings.Repeat(" ", redLen))
	}
	if emptyLen > 0 {
		emptyPart = lipgloss.NewStyle().Background(lipgloss.Color("#393B40")).Render(strings.Repeat(" ", emptyLen))
	}

	barLine := greenPart + redPart + emptyPart

	// Line 2: Info row directly below the progress bar
	availW := width - 2 // FooterBarStyle has Padding(0, 1)

	statsText := fmt.Sprintf("%3.0f%% (%d/%d) • %v",
		passPercent,
		m.stats.Passed,
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

// wrapAndScroll wraps text to maxW and slices the result between scrollOffset and scrollOffset+maxH.
func wrapAndScroll(text string, maxW, maxH, scrollOffset int) (string, int, int) {
	if maxW < 1 {
		maxW = 1
	}
	wrapStyle := lipgloss.NewStyle().Width(maxW)
	wrapped := wrapStyle.Render(text)
	lines := strings.Split(wrapped, "\n")
	totalLines := len(lines)

	maxScroll := totalLines - maxH
	if maxScroll < 0 {
		maxScroll = 0
	}
	if scrollOffset > maxScroll {
		scrollOffset = maxScroll
	}
	if scrollOffset < 0 {
		scrollOffset = 0
	}

	end := scrollOffset + maxH
	if end > totalLines {
		end = totalLines
	}
	visible := lines[scrollOffset:end]

	return strings.Join(visible, "\n"), scrollOffset, totalLines
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
