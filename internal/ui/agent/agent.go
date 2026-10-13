package agent

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/tristan-gscn/repaco/internal/model"
)

// PrintAgentReport writes an AI-agent-friendly diagnostic report in GitHub-flavored Markdown.
func PrintAgentReport(w io.Writer, res *model.SuiteResult) {
	if res == nil {
		fmt.Fprintln(w, "Error: nil suite result")
		return
	}

	outcome := "✅ PASSED"
	if !res.Stats.IsPerfect() {
		outcome = "❌ FAILED"
	}

	fmt.Fprintf(w, "# 🤖 REPACO AGENT REPORT: %s\n\n", res.ProjectID)
	fmt.Fprintf(w, "- **Outcome**: %s\n", outcome)
	fmt.Fprintf(w, "- **Pass Rate**: %.1f%% (%d/%d passed)\n", res.Stats.SuccessRate(), res.Stats.Passed, res.Stats.Total)
	fmt.Fprintf(w, "- **Duration**: %v\n", res.Duration.Round(100*1000))
	fmt.Fprintf(w, "- **Metrics**: %d Passed, %d Failed, %d Crashed, %d Leaked, %d Timeouts\n\n",
		res.Stats.Passed, res.Stats.Failed, res.Stats.Crashed, res.Stats.Leaked, res.Stats.Timeout)

	var failures []model.TestResult
	var passes []model.TestResult

	for _, r := range res.Results {
		if r.Status == model.StatusPassed {
			passes = append(passes, r)
		} else {
			failures = append(failures, r)
		}
	}

	// 1. Failures Section
	if len(failures) > 0 {
		fmt.Fprintf(w, "## ❌ FAILURES (%d)\n\n", len(failures))
		for i, f := range failures {
			fn := f.Case.FileName()
			fmt.Fprintf(w, "### %d. [%s] %s (`%s`)\n", i+1, f.Case.ID, f.Case.Name, fn)
			fmt.Fprintf(w, "- **Status**: `%s`\n", f.Status)
			fmt.Fprintf(w, "- **Category**: `%s`\n", f.Case.Category)

			if f.Input != "" {
				fmt.Fprintf(w, "- **Input**: `%s`\n", f.Input)
			}
			if f.Expected != "" {
				fmt.Fprintf(w, "- **Expected**: `%s`\n", f.Expected)
			}
			if f.Actual != "" {
				fmt.Fprintf(w, "- **Actual**: `%s`\n", f.Actual)
			}
			if f.Explanation != "" {
				fmt.Fprintf(w, "- **Diagnosis**: %s\n", f.Explanation)
			}
			if f.Advice != "" {
				fmt.Fprintf(w, "- **Recommendation**: %s\n", f.Advice)
			}
			if f.LeaksReport != "" {
				fmt.Fprintf(w, "- **Sanitizer Leak Report**:\n```\n%s\n```\n", f.LeaksReport)
			}
			if f.Trace != "" {
				fmt.Fprintf(w, "- **Crash Trace**:\n```\n%s\n```\n", f.Trace)
			}
			if f.Diff != "" {
				fmt.Fprintf(w, "- **Diff**:\n```diff\n%s\n```\n", f.Diff)
			}
			if f.CodeSnippet != "" {
				fmt.Fprintf(w, "- **Reproduce in C**:\n```c\n%s\n```\n", f.CodeSnippet)
			}
			fmt.Fprintln(w)
		}
	}

	// 2. Passed Section (compact)
	if len(passes) > 0 {
		fmt.Fprintf(w, "## ✔️ PASSED TESTS (%d)\n\n", len(passes))
		for _, p := range passes {
			fmt.Fprintf(w, "- `[%s]` %s (`%s`) [%v]\n", p.Case.ID, p.Case.Name, p.Case.FileName(), p.Duration.Round(10*1000))
		}
		fmt.Fprintln(w)
	}

	// 3. Compact JSON Block for programmatic ingestion
	data, err := json.MarshalIndent(res, "", "  ")
	if err == nil {
		fmt.Fprintln(w, "## 📋 STRUCTURED JSON")
		fmt.Fprintln(w, "```json")
		fmt.Fprintln(w, string(data))
		fmt.Fprintln(w, "```")
	}
}

// PrintJSON writes the raw JSON payload to w.
func PrintJSON(w io.Writer, res *model.SuiteResult) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(res)
}
