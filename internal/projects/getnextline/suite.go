package getnextline

import (
	"context"
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/tristan-gscn/repaco/internal/builder"
	"github.com/tristan-gscn/repaco/internal/config"
	"github.com/tristan-gscn/repaco/internal/detector"
	"github.com/tristan-gscn/repaco/internal/model"
	"github.com/tristan-gscn/repaco/internal/projects"
	"github.com/tristan-gscn/repaco/internal/runner"
	"github.com/tristan-gscn/repaco/pkg/diff"
)

//go:embed harness.c.template
var harnessSource string

type Suite struct {
	harnesses map[string]string // key: "bs42", "bs1", "bs1024", "bs9999", "bs10m", "default", "bonus"
	hasBonus  bool
}

func init() {
	projects.Register(&Suite{})
}

func (s *Suite) ID() model.ProjectID {
	return model.ProjectGetNextLine
}

func (s *Suite) Metadata() model.ProjectMetadata {
	return detector.KnownProjects[model.ProjectGetNextLine]
}

func (s *Suite) Prepare(ctx context.Context, dir string, cfg *config.Config, b *builder.Builder) (*model.TestResult, error) {
	s.harnesses = make(map[string]string)

	// 1. Check required mandatory source files
	required := []string{"get_next_line.h", "get_next_line.c", "get_next_line_utils.c"}
	for _, req := range required {
		p := filepath.Join(dir, req)
		if _, err := os.Stat(p); os.IsNotExist(err) {
			return &model.TestResult{
				Case: model.TestCase{
					ID:          "gnl_prep_missing",
					Module:      "get_next_line",
					File:        "get_next_line.c",
					Name:        "Source verification",
					Category:    model.CategoryBuild,
					Description: "Check required GNL files",
				},
				Status:      model.StatusFailed,
				Explanation: fmt.Sprintf("Required file %s is missing in %s", req, dir),
				Advice:      fmt.Sprintf("Ensure %s is present at the root of the repository.", req),
			}, nil
		}
	}

	// 2. Check bonus files
	bonusFiles := []string{"get_next_line_bonus.h", "get_next_line_bonus.c", "get_next_line_utils_bonus.c"}
	s.hasBonus = true
	for _, bf := range bonusFiles {
		if _, err := os.Stat(filepath.Join(dir, bf)); os.IsNotExist(err) {
			s.hasBonus = false
			break
		}
	}

	// 3. Write C harness
	workDir := filepath.Join(dir, ".repaco")
	if err := os.MkdirAll(workDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create .repaco directory: %w", err)
	}

	harnessSrcPath := filepath.Join(workDir, "gnl_harness.c")
	if err := os.WriteFile(harnessSrcPath, []byte(harnessSource), 0644); err != nil {
		return nil, fmt.Errorf("failed to write harness source: %w", err)
	}

	includeDirs := []string{dir}
	mandatorySources := []string{
		filepath.Join(dir, "get_next_line.c"),
		filepath.Join(dir, "get_next_line_utils.c"),
	}

	// 4. Compile configs: bs42, bs1, bs1024, bs9999, bs10m, default
	configs := []struct {
		key   string
		flags []string
	}{
		{"bs42", []string{"-fsanitize=address,undefined", "-g", "-D", "BUFFER_SIZE=42"}},
		{"bs1", []string{"-fsanitize=address,undefined", "-g", "-D", "BUFFER_SIZE=1"}},
		{"bs1024", []string{"-fsanitize=address,undefined", "-g", "-D", "BUFFER_SIZE=1024"}},
		{"bs9999", []string{"-fsanitize=address,undefined", "-g", "-D", "BUFFER_SIZE=9999"}},
		{"bs10m", []string{"-fsanitize=address,undefined", "-g", "-D", "BUFFER_SIZE=10000000"}},
		{"default", []string{"-fsanitize=address,undefined", "-g"}},
	}

	for _, c := range configs {
		binPath := filepath.Join(workDir, "gnl_"+c.key)
		out, err := b.CompileHarness(ctx, harnessSrcPath, binPath, includeDirs, c.flags, mandatorySources)
		if err != nil {
			return &model.TestResult{
				Case: model.TestCase{
					ID:          "gnl_prep_compile_" + c.key,
					Module:      "get_next_line_buffers",
					File:        "get_next_line_buffers.c",
					Name:        fmt.Sprintf("Compilation check (%s)", c.key),
					Category:    model.CategoryBuild,
					Description: "Verifies compilation with -Wall -Wextra -Werror and AddressSanitizer",
				},
				Status:      model.StatusFailed,
				Explanation: fmt.Sprintf("Compilation failed for %s:\n%s", c.key, out),
				Advice:      "Fix C warnings and compilation errors.",
			}, nil
		}
		s.harnesses[c.key] = binPath
	}

	// 5. Compile bonus if available
	if s.hasBonus {
		bonusSources := []string{
			filepath.Join(dir, "get_next_line_bonus.c"),
			filepath.Join(dir, "get_next_line_utils_bonus.c"),
		}
		bonusBinPath := filepath.Join(workDir, "gnl_bonus")
		bonusFlags := []string{"-fsanitize=address,undefined", "-g", "-D", "BUFFER_SIZE=42", "-DHAS_BONUS=1"}
		out, err := b.CompileHarness(ctx, harnessSrcPath, bonusBinPath, includeDirs, bonusFlags, bonusSources)
		if err != nil {
			return &model.TestResult{
				Case: model.TestCase{
					ID:          "gnl_prep_compile_bonus",
					Module:      "get_next_line_bonus",
					File:        "get_next_line_bonus.c",
					Name:        "Bonus compilation check",
					Category:    model.CategoryBonus,
					Description: "Compiles get_next_line_bonus files",
				},
				Status:      model.StatusFailed,
				Explanation: fmt.Sprintf("Bonus compilation failed:\n%s", out),
				Advice:      "Check bonus prototypes and includes.",
			}, nil
		}
		s.harnesses["bonus"] = bonusBinPath
	}

	return &model.TestResult{
		Case: model.TestCase{
			ID:          "gnl_prep_ok",
			Module:      "get_next_line",
			File:        "get_next_line.c",
			Name:        "GNL compilation and prerequisites",
			Category:    model.CategoryBuild,
			Description: "All BUFFER_SIZE configurations compiled cleanly with AddressSanitizer",
		},
		Status: model.StatusPassed,
	}, nil
}

func (s *Suite) Plan(ctx context.Context, dir string, cfg *config.Config) ([]model.TestCase, error) {
	var plan []model.TestCase

	// 1. Standard Scenarios (get_next_line.c)
	standardCases := []struct {
		id, name, desc string
	}{
		{"gnl_single_line_newline", "Single line with newline", "Tests reading a single line ending with \\n"},
		{"gnl_single_line_no_newline", "Single line without newline", "Tests reading a line without a trailing \\n"},
		{"gnl_multiple_lines", "Standard multi-line file", "Tests reading consecutive lines from file"},
		{"gnl_no_trailing_newline", "File without trailing newline", "Tests reading multi-line file ending without \\n"},
		{"gnl_empty_file", "Empty file (0 bytes)", "Tests reading an empty file returning NULL on first call"},
		{"gnl_empty_line_only", "Single newline only (\\n)", "Tests file containing just \\n returning \\n then NULL"},
		{"gnl_multiple_empty_lines", "Multiple consecutive newlines", "Tests file with consecutive empty lines (\\n\\n\\n)"},
		{"gnl_long_line", "Long line (> 4000 characters)", "Tests buffer stitching on lines larger than buffer"},
		{"gnl_short_lines", "Short lines (single character)", "Tests reading 1-character lines"},
		{"gnl_calls_after_eof", "Repeated calls after EOF", "Tests calling get_next_line repeatedly after EOF returning NULL"},
		{"gnl_pipe_read", "Reading from pipe stream", "Tests reading from pipe file descriptor (stdin simulation)"},
	}
	for _, sc := range standardCases {
		plan = append(plan, model.TestCase{
			ID:          sc.id,
			Module:      "get_next_line",
			File:        "get_next_line.c",
			Name:        sc.name,
			Description: sc.desc,
			Category:    model.CategoryMandatory,
			Timeout:     3 * time.Second,
		})
	}

	// 2. Buffer Sizes (get_next_line_buffers.c)
	bufferCases := []struct {
		id, name, desc string
	}{
		{"gnl_bs_1_multiline", "BUFFER_SIZE=1 multi-line reading", "Tests reading multiple lines 1 byte at a time"},
		{"gnl_bs_1_no_newline", "BUFFER_SIZE=1 without trailing \\n", "Tests EOF without newline reading 1 byte at a time"},
		{"gnl_bs_1_long_line", "BUFFER_SIZE=1 long line (> 4000 chars)", "Tests buffer stitching 1 byte at a time on long lines"},
		{"gnl_bs_1024_multiline", "BUFFER_SIZE=1024 multi-line reading", "Tests standard 1KB buffer size reading"},
		{"gnl_bs_9999_multiline", "BUFFER_SIZE=9999 multi-line reading", "Tests large buffer size reading whole lines in one chunk"},
		{"gnl_bs_9999_short", "BUFFER_SIZE=9999 short lines", "Tests buffer holding multiple lines in single read"},
		{"gnl_bs_10m_multiline", "BUFFER_SIZE=10000000 very large buffer", "Tests huge buffer size (10MB) memory allocation"},
		{"gnl_bs_default_compile", "Default BUFFER_SIZE from header", "Tests compilation and execution without -D BUFFER_SIZE"},
	}
	for _, bc := range bufferCases {
		plan = append(plan, model.TestCase{
			ID:          bc.id,
			Module:      "get_next_line_buffers",
			File:        "get_next_line_buffers.c",
			Name:        bc.name,
			Description: bc.desc,
			Category:    model.CategoryMandatory,
			Timeout:     3 * time.Second,
		})
	}

	// 3. Edge Cases (get_next_line_edge_cases.c)
	edgeCases := []struct {
		id, name, desc string
	}{
		{"gnl_edge_negative_fd", "Negative file descriptor", "Tests get_next_line(-1) and get_next_line(-42) returning NULL"},
		{"gnl_edge_closed_fd", "Closed file descriptor", "Tests get_next_line on a closed fd returning NULL"},
		{"gnl_edge_large_fd", "Out-of-range file descriptor", "Tests get_next_line(10240) returning NULL"},
	}
	for _, ec := range edgeCases {
		plan = append(plan, model.TestCase{
			ID:          ec.id,
			Module:      "get_next_line_edge_cases",
			File:        "get_next_line_edge_cases.c",
			Name:        ec.name,
			Description: ec.desc,
			Category:    model.CategoryEdgeCase,
			Timeout:     2 * time.Second,
		})
	}

	// 4. Bonus Cases (get_next_line_bonus.c)
	if s.hasBonus {
		plan = append(plan, model.TestCase{
			ID:          "gnl_bonus_interleaved",
			Module:      "get_next_line_bonus",
			File:        "get_next_line_bonus.c",
			Name:        "Interleaved reads on 3 file descriptors",
			Description: "Tests simultaneous independent reading across fd 3, 4, 5 without losing state",
			Category:    model.CategoryBonus,
			Timeout:     3 * time.Second,
		})
	}

	return plan, nil
}

func (s *Suite) RunCase(ctx context.Context, dir string, tc model.TestCase, cfg *config.Config, b *builder.Builder, r *runner.Runner) (*model.TestResult, error) {
	// Determine which harness binary and sub-argument to invoke
	harnessKey := "bs42"
	targetArg := tc.ID

	if strings.HasPrefix(tc.ID, "gnl_bs_1_") {
		harnessKey = "bs1"
		if tc.ID == "gnl_bs_1_multiline" {
			targetArg = "gnl_multiple_lines"
		} else if tc.ID == "gnl_bs_1_no_newline" {
			targetArg = "gnl_no_trailing_newline"
		} else if tc.ID == "gnl_bs_1_long_line" {
			targetArg = "gnl_long_line"
		}
	} else if strings.HasPrefix(tc.ID, "gnl_bs_1024_") {
		harnessKey = "bs1024"
		targetArg = "gnl_multiple_lines"
	} else if strings.HasPrefix(tc.ID, "gnl_bs_9999_") {
		harnessKey = "bs9999"
		if tc.ID == "gnl_bs_9999_short" {
			targetArg = "gnl_short_lines"
		} else {
			targetArg = "gnl_multiple_lines"
		}
	} else if strings.HasPrefix(tc.ID, "gnl_bs_10m_") {
		harnessKey = "bs10m"
		targetArg = "gnl_multiple_lines"
	} else if strings.HasPrefix(tc.ID, "gnl_bs_default_") {
		harnessKey = "default"
		targetArg = "gnl_multiple_lines"
	} else if strings.HasPrefix(tc.ID, "gnl_bonus_") {
		harnessKey = "bonus"
		targetArg = tc.ID
	}

	harnessBin, ok := s.harnesses[harnessKey]
	if !ok || harnessBin == "" {
		return &model.TestResult{
			Case:        tc,
			Status:      model.StatusFailed,
			Explanation: fmt.Sprintf("Harness binary for %s not found", harnessKey),
		}, nil
	}

	res, err := r.Run(ctx, harnessBin, runner.RunOptions{
		Args:    []string{"run", targetArg},
		Timeout: tc.Timeout,
	})
	if err != nil {
		return &model.TestResult{
			Case:        tc,
			Status:      model.StatusFailed,
			Explanation: fmt.Sprintf("Execution error: %v", err),
		}, nil
	}

	// 1. Process Crashed
	if res.Crashed {
		return &model.TestResult{
			Case:        tc,
			Status:      model.StatusCrash,
			Duration:    res.Duration,
			Explanation: res.Explanation,
			Signal:      res.Signal,
			Advice:      "Check for NULL pointer dereference, buffer overflow, or reading from uninitialized memory.",
		}, nil
	}

	// 2. Process Timed Out
	if res.TimedOut {
		return &model.TestResult{
			Case:        tc,
			Status:      model.StatusTimeout,
			Duration:    res.Duration,
			Explanation: res.Explanation,
			Advice:      "Check loop termination conditions (e.g. read return value <= 0 or newline search).",
		}, nil
	}

	// 3. Memory Leak detected by AddressSanitizer
	if res.Status == model.StatusLeak {
		return &model.TestResult{
			Case:        tc,
			Status:      model.StatusLeak,
			Duration:    res.Duration,
			Explanation: res.Explanation,
			LeaksReport: res.LeakReport,
			Advice:      "Ensure all memory allocated during read and line extraction is properly freed.",
		}, nil
	}

	// 4. Parse harness output
	testRes := &model.TestResult{
		Case:     tc,
		Duration: res.Duration,
	}

	stdout := strings.TrimSpace(res.Stdout)
	lines := strings.Split(stdout, "\n")
	var status model.Status = model.StatusFailed
	var input, expected, actual, explanation, advice string
	var codeLines []string
	var inCodeBlock bool

	for _, line := range lines {
		if strings.HasPrefix(line, "STATUS: PASS") {
			status = model.StatusPassed
		} else if strings.HasPrefix(line, "STATUS: FAIL") {
			status = model.StatusFailed
		} else if strings.HasPrefix(line, "INPUT: ") {
			input = strings.TrimPrefix(line, "INPUT: ")
		} else if strings.HasPrefix(line, "EXPECTED: ") {
			expected = strings.TrimPrefix(line, "EXPECTED: ")
		} else if strings.HasPrefix(line, "ACTUAL: ") {
			actual = strings.TrimPrefix(line, "ACTUAL: ")
		} else if strings.HasPrefix(line, "EXPLANATION: ") {
			explanation = strings.TrimPrefix(line, "EXPLANATION: ")
		} else if strings.HasPrefix(line, "ADVICE: ") {
			advice = strings.TrimPrefix(line, "ADVICE: ")
		} else if strings.HasPrefix(line, "CODE_START") {
			inCodeBlock = true
		} else if strings.HasPrefix(line, "CODE_END") {
			inCodeBlock = false
		} else if inCodeBlock {
			codeLines = append(codeLines, line)
		}
	}

	testRes.Status = status
	testRes.Input = input
	testRes.Expected = expected
	testRes.Actual = actual
	testRes.CodeSnippet = strings.Join(codeLines, "\n")
	testRes.Explanation = explanation
	testRes.Advice = advice
	if status != model.StatusPassed && expected != "" && actual != "" {
		testRes.Diff = diff.Compare(expected, actual)
	}

	return testRes, nil
}

func (s *Suite) Cleanup(ctx context.Context, dir string) error {
	workDir := filepath.Join(dir, ".repaco")
	return os.RemoveAll(workDir)
}
