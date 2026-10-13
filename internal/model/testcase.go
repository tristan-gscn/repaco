package model

import (
	"strings"
	"time"
)

// Status represents the verdict of a test case execution.
type Status string

const (
	StatusPending Status = "PENDING"
	StatusRunning Status = "RUNNING"
	StatusPassed  Status = "PASS"
	StatusFailed  Status = "FAIL"
	StatusCrash   Status = "CRASH"
	StatusTimeout Status = "TIMEOUT"
	StatusLeak    Status = "LEAK"
	StatusSkipped Status = "SKIPPED"
)

// Category groups tests by nature.
type Category string

const (
	CategoryBuild     Category = "BUILD"
	CategoryMandatory Category = "MANDATORY"
	CategoryEdgeCase  Category = "EDGE_CASE"
	CategoryBonus     Category = "BONUS"
	CategoryMemory    Category = "MEMORY"
	CategoryBenchmark Category = "BENCHMARK"
)

// TestCase defines a single test scenario to execute.
type TestCase struct {
	ID          string        `json:"id"`
	Module      string        `json:"module"` // e.g. "ft_split", "buffer_size_1", "100_numbers"
	File        string        `json:"file,omitempty"` // e.g. "ft_split.c", "Makefile"
	Name        string        `json:"name"`
	Description string        `json:"description"`
	Category    Category      `json:"category"`
	Timeout     time.Duration `json:"timeout"`
	Tags        []string      `json:"tags,omitempty"`
}

// FileName returns the corresponding C source file name (e.g. ft_split.c, ft_lstnew_bonus.c, Makefile).
func (tc TestCase) FileName() string {
	if tc.File != "" {
		return tc.File
	}
	if strings.HasSuffix(tc.Module, ".c") || tc.Module == "Makefile" || tc.Module == "Build" {
		return tc.Module
	}
	if tc.Category == CategoryBonus && !strings.HasSuffix(tc.Module, "_bonus") {
		return tc.Module + "_bonus.c"
	}
	return tc.Module + ".c"
}

// TestResult represents the outcome and deep diagnostics of a TestCase.
type TestResult struct {
	Case        TestCase      `json:"case"`
	Status      Status        `json:"status"`
	Duration    time.Duration `json:"duration"`
	Input       string        `json:"input,omitempty"`
	Expected    string        `json:"expected,omitempty"`
	Actual      string        `json:"actual,omitempty"`
	Diff        string        `json:"diff,omitempty"`
	Explanation string        `json:"explanation,omitempty"` // Clear reason why it failed
	Advice      string        `json:"advice,omitempty"`      // Actionable hint on how to fix
	CodeSnippet string        `json:"code_snippet,omitempty"` // Standalone C snippet to reproduce locally
	Trace       string        `json:"trace,omitempty"`       // Stack trace or signal if crashed
	LeaksReport string        `json:"leaks_report,omitempty"`// Sanitizer or leaks trace
	ExitCode    int           `json:"exit_code"`
	Signal      string        `json:"signal,omitempty"`
}
