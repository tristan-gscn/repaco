package agent

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/tristan-gscn/repaco/internal/model"
)

func TestPrintAgentReport(t *testing.T) {
	res := &model.SuiteResult{
		ProjectID: model.ProjectFtPrintf,
		Duration:  500 * time.Millisecond,
		Stats: model.SummaryStats{
			Total:  2,
			Passed: 1,
			Failed: 1,
		},
		Results: []model.TestResult{
			{
				Case: model.TestCase{
					ID:       "case_pass",
					Module:   "ft_printf_c",
					Name:     "Char Pass",
					Category: model.CategoryMandatory,
				},
				Status:   model.StatusPassed,
				Duration: 10 * time.Millisecond,
			},
			{
				Case: model.TestCase{
					ID:       "case_fail",
					Module:   "ft_printf_c",
					Name:     "Char Fail",
					Category: model.CategoryMandatory,
				},
				Status:      model.StatusFailed,
				Duration:    20 * time.Millisecond,
				Input:       "ft_printf(\"%c\", 'a')",
				Expected:    "ret: 1, output: \"a\"",
				Actual:      "ret: 0, output: \"\"",
				Explanation: "Output content mismatch",
				Advice:      "Write character to stdout",
				CodeSnippet: "ft_printf(\"%c\", 'a');",
			},
		},
	}

	var buf bytes.Buffer
	PrintAgentReport(&buf, res)
	out := buf.String()

	if !strings.Contains(out, "REPACO AGENT REPORT: ft_printf") {
		t.Errorf("expected header, got: %s", out)
	}
	if !strings.Contains(out, "FAILURES (1)") {
		t.Errorf("expected failure section, got: %s", out)
	}
	if !strings.Contains(out, "ft_printf(\"%c\", 'a');") {
		t.Errorf("expected code snippet, got: %s", out)
	}
	if !strings.Contains(out, "STRUCTURED JSON") {
		t.Errorf("expected structured JSON, got: %s", out)
	}
}

func TestPrintJSON(t *testing.T) {
	res := &model.SuiteResult{
		ProjectID: model.ProjectFtPrintf,
		Stats: model.SummaryStats{
			Total:  1,
			Passed: 1,
		},
	}
	var buf bytes.Buffer
	if err := PrintJSON(&buf, res); err != nil {
		t.Fatalf("PrintJSON error: %v", err)
	}
	if !strings.Contains(buf.String(), `"project_id": "ft_printf"`) {
		t.Errorf("unexpected JSON output: %s", buf.String())
	}
}
