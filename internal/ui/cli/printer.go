package cli

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/tristan-gscn/repaco/internal/model"
)

var (
	passStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("#10B981")).Bold(true)
	failStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("#EF4444")).Bold(true)
	crashStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#DC2626")).Bold(true)
	timeoutStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#F59E0B")).Bold(true)
	leakStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("#3B82F6")).Bold(true)
	dimStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("#6B7280"))
	titleStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#8B5CF6")).Bold(true)
	boxStyle     = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(0, 1)
)

// Printer writes formatted test progress and results to an io.Writer.
type Printer struct {
	w io.Writer
}

// NewPrinter creates a printer defaulting to os.Stdout.
func NewPrinter() *Printer {
	return &Printer{w: os.Stdout}
}

// PrintDetection displays detected project info.
func (p *Printer) PrintDetection(d *model.DetectionResult) {
	fmt.Fprintf(p.w, "%s %s (confidence: %.0f%%)\n",
		titleStyle.Render("==> Detected 42 Project:"),
		lipgloss.NewStyle().Bold(true).Render(d.Project.Name),
		d.Confidence*100,
	)
	if d.Project.Description != "" {
		fmt.Fprintf(p.w, "    %s\n", dimStyle.Render(d.Project.Description))
	}
	if len(d.MissingFiles) > 0 {
		fmt.Fprintf(p.w, "    %s %s\n",
			failStyle.Render("Missing files:"),
			strings.Join(d.MissingFiles, ", "),
		)
	}
	fmt.Fprintln(p.w)
}

// PrintTestResult writes a single test result row.
func (p *Printer) PrintTestResult(res *model.TestResult) {
	var statusBadge string
	switch res.Status {
	case model.StatusPassed:
		statusBadge = passStyle.Render("  PASS ")
	case model.StatusFailed:
		statusBadge = failStyle.Render("  FAIL ")
	case model.StatusCrash:
		statusBadge = crashStyle.Render(" CRASH ")
	case model.StatusTimeout:
		statusBadge = timeoutStyle.Render("  TIME ")
	case model.StatusLeak:
		statusBadge = leakStyle.Render("  LEAK ")
	default:
		statusBadge = dimStyle.Render(fmt.Sprintf("%7s", res.Status))
	}

	cat := dimStyle.Render(fmt.Sprintf("[%s]", res.Case.Category))
	dur := dimStyle.Render(fmt.Sprintf("(%v)", res.Duration.Round(100)))

	fmt.Fprintf(p.w, "%s %-32s %-16s %s\n", statusBadge, res.Case.Name, cat, dur)

	if res.Status != model.StatusPassed && res.Status != model.StatusSkipped {
		if res.Explanation != "" {
			fmt.Fprintf(p.w, "         %s %s\n", failStyle.Render("Reason:"), res.Explanation)
		}
		if res.Diff != "" {
			fmt.Fprintf(p.w, "         %s\n%s\n", titleStyle.Render("Diff:"), indent(res.Diff, "           "))
		}
		if res.Advice != "" {
			fmt.Fprintf(p.w, "         %s %s\n", passStyle.Render("Fix:"), res.Advice)
		}
		if res.LeaksReport != "" {
			fmt.Fprintf(p.w, "         %s\n%s\n", leakStyle.Render("Sanitizer Trace:"), indent(res.LeaksReport, "           "))
		}
	}
}

// PrintSummary outputs final session metrics.
func (p *Printer) PrintSummary(summary *model.SuiteResult) {
	s := summary.Stats
	fmt.Fprintln(p.w)
	fmt.Fprintln(p.w, titleStyle.Render("═══════════════════════════════════════════════════════════════════"))
	fmt.Fprintf(p.w, " %s | Total: %d | Time: %v\n",
		lipgloss.NewStyle().Bold(true).Render(string(summary.ProjectID)),
		s.Total,
		summary.Duration.Round(100),
	)
	fmt.Fprintf(p.w, " %s  %s  %s  %s  %s\n",
		passStyle.Render(fmt.Sprintf("✓ Passed: %d", s.Passed)),
		failStyle.Render(fmt.Sprintf("✗ Failed: %d", s.Failed)),
		crashStyle.Render(fmt.Sprintf("💥 Crashes: %d", s.Crashed)),
		timeoutStyle.Render(fmt.Sprintf("⏱ Timeout: %d", s.Timeout)),
		leakStyle.Render(fmt.Sprintf("💧 Leaks: %d", s.Leaked)),
	)
	fmt.Fprintf(p.w, " Score: %.1f%%\n", s.SuccessRate())
	if s.IsPerfect() {
		fmt.Fprintln(p.w, passStyle.Render(" 🎉 PERFECT PROJECT! 100% SUCCESS."))
	}
	fmt.Fprintln(p.w, titleStyle.Render("═══════════════════════════════════════════════════════════════════"))
}

func indent(str, prefix string) string {
	lines := strings.Split(str, "\n")
	for i, l := range lines {
		lines[i] = prefix + l
	}
	return strings.Join(lines, "\n")
}
