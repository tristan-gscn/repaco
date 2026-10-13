package ftprintf

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
	harnessPath string
}

func init() {
	projects.Register(&Suite{})
}

func (s *Suite) ID() model.ProjectID {
	return model.ProjectFtPrintf
}

func (s *Suite) Metadata() model.ProjectMetadata {
	return detector.KnownProjects[model.ProjectFtPrintf]
}

func (s *Suite) Prepare(ctx context.Context, dir string, cfg *config.Config, b *builder.Builder) (*model.TestResult, error) {
	// 1. Check Makefile compliance
	check, err := b.CheckMakefile(ctx, dir, "libftprintf.a")
	if err != nil {
		return &model.TestResult{
			Case: model.TestCase{
				ID:          "ftprintf_build_makefile",
				Module:      "Makefile",
				File:        "Makefile",
				Name:        "Makefile compliance check",
				Category:    model.CategoryBuild,
				Description: "Verifies mandatory rules, compiler flags, and relink behavior",
			},
			Status:      model.StatusFailed,
			Explanation: fmt.Sprintf("Makefile check failed: %v", err),
			Advice:      "Check that Makefile is present and rules 'all', 'clean', 'fclean', 're' compile cleanly.",
		}, nil
	}

	if len(check.MissingRules) > 0 {
		return &model.TestResult{
			Case: model.TestCase{
				ID:          "ftprintf_build_rules",
				Module:      "Makefile",
				File:        "Makefile",
				Name:        "Mandatory Makefile rules",
				Category:    model.CategoryBuild,
				Description: "Checks presence of rules all, clean, fclean, re",
			},
			Status:      model.StatusFailed,
			Explanation: fmt.Sprintf("Missing mandatory rules: %s", strings.Join(check.MissingRules, ", ")),
			Advice:      "Define rules $(NAME), all, clean, fclean, and re in your Makefile.",
		}, nil
	}

	if check.RelinkDetected {
		return &model.TestResult{
			Case: model.TestCase{
				ID:          "ftprintf_build_relink",
				Module:      "Makefile",
				File:        "Makefile",
				Name:        "Relink verification",
				Category:    model.CategoryBuild,
				Description: "Makefile must not relink when no files are modified",
			},
			Status:      model.StatusFailed,
			Explanation: "Makefile recompiled object files or archive even though no sources changed.",
			Advice:      "Ensure object targets have proper prerequisites and 'all' depends on $(NAME).",
		}, nil
	}

	// 2. Compile libftprintf.a
	libPath := filepath.Join(dir, "libftprintf.a")
	makeOut, err := b.ExecuteMake(ctx, dir, "all", 30*time.Second)
	if err != nil {
		return &model.TestResult{
			Case: model.TestCase{
				ID:          "ftprintf_build_compile",
				Module:      "Makefile",
				File:        "Makefile",
				Name:        "make all compilation",
				Category:    model.CategoryBuild,
				Description: "Compiles libftprintf.a archive",
			},
			Status:      model.StatusFailed,
			Explanation: fmt.Sprintf("Failed to compile libftprintf.a:\n%s", makeOut),
			Advice:      "Fix compilation errors or missing dependencies.",
		}, nil
	}
	if _, err := os.Stat(libPath); os.IsNotExist(err) {
		return &model.TestResult{
			Case: model.TestCase{
				ID:          "ftprintf_build_compile",
				Module:      "Makefile",
				File:        "Makefile",
				Name:        "make all compilation",
				Category:    model.CategoryBuild,
				Description: "Compiles libftprintf.a archive",
			},
			Status:      model.StatusFailed,
			Explanation: "Rule 'all' completed but libftprintf.a was not created.",
			Advice:      "Ensure $(NAME) produces libftprintf.a at the root directory.",
		}, nil
	}

	// 3. Write and compile C harness with AddressSanitizer
	workDir := filepath.Join(dir, ".repaco")
	if err := os.MkdirAll(workDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create .repaco directory: %w", err)
	}

	harnessSrcPath := filepath.Join(workDir, "ftprintf_harness.c")
	if err := os.WriteFile(harnessSrcPath, []byte(harnessSource), 0644); err != nil {
		return nil, fmt.Errorf("failed to write harness source: %w", err)
	}

	s.harnessPath = filepath.Join(workDir, "ftprintf_harness")
	extraFlags := []string{"-fsanitize=address,undefined", "-g"}

	includeDirs := []string{
		dir,
		filepath.Join(dir, "includes"),
		filepath.Join(dir, "inc"),
		filepath.Join(dir, "libft"),
	}
	linkLibs := []string{"-L" + dir, "-lftprintf"}

	harnessOut, err := b.CompileHarness(ctx, harnessSrcPath, s.harnessPath, includeDirs, extraFlags, linkLibs)
	if err != nil {
		return &model.TestResult{
			Case: model.TestCase{
				ID:          "ftprintf_build_harness",
				Module:      "Makefile",
				File:        "Makefile",
				Name:        "Test harness compilation",
				Category:    model.CategoryBuild,
				Description: "Compiles test harness linking against libftprintf.a with AddressSanitizer",
			},
			Status:      model.StatusFailed,
			Explanation: fmt.Sprintf("Failed to link against libftprintf.a:\n%s", harnessOut),
			Advice:      "Check that libftprintf.a exports ft_printf and includes ft_printf.h.",
		}, nil
	}

	return &model.TestResult{
		Case: model.TestCase{
			ID:          "ftprintf_build_success",
			Module:      "Makefile",
			File:        "Makefile",
			Name:        "Build & Makefile verification",
			Category:    model.CategoryBuild,
			Description: "libftprintf.a built cleanly with -Wall -Wextra -Werror, without relink",
		},
		Status: model.StatusPassed,
	}, nil
}

func (s *Suite) Plan(ctx context.Context, dir string, cfg *config.Config) ([]model.TestCase, error) {
	if s.harnessPath == "" {
		return nil, fmt.Errorf("harness not compiled yet")
	}

	r := runner.New(cfg.DefaultTimeout)
	res, err := r.Run(ctx, s.harnessPath, runner.RunOptions{
		Args: []string{"list"},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list tests from harness: %w", err)
	}

	var plan []model.TestCase
	lines := strings.Split(strings.TrimSpace(res.Stdout), "\n")
	for _, l := range lines {
		parts := strings.Split(l, "\t")
		if len(parts) < 5 {
			continue
		}
		tc := model.TestCase{
			ID:          parts[0],
			Module:      parts[1],
			File:        parts[1] + ".c",
			Name:        parts[2],
			Category:    model.Category(parts[3]),
			Description: parts[4],
			Timeout:     3 * time.Second,
		}
		if cfg.FilterModule != "" && cfg.FilterModule != tc.Module {
			continue
		}
		plan = append(plan, tc)
	}

	return plan, nil
}

func (s *Suite) RunCase(ctx context.Context, dir string, tc model.TestCase, cfg *config.Config, b *builder.Builder, r *runner.Runner) (*model.TestResult, error) {
	if s.harnessPath == "" {
		return nil, fmt.Errorf("harness binary not found")
	}

	res, err := r.Run(ctx, s.harnessPath, runner.RunOptions{
		Args:    []string{"run", tc.ID},
		Timeout: tc.Timeout,
	})
	if err != nil {
		return &model.TestResult{
			Case:        tc,
			Status:      model.StatusFailed,
			Explanation: fmt.Sprintf("Process execution error: %v", err),
		}, nil
	}

	// 1. Process Crashed (SIGSEGV, SIGBUS, etc.)
	if res.Crashed {
		return &model.TestResult{
			Case:        tc,
			Status:      model.StatusCrash,
			Duration:    res.Duration,
			Explanation: res.Explanation,
			Signal:      res.Signal,
			Advice:      "Check for null pointer dereference, invalid format string access, or uninitialized memory.",
		}, nil
	}

	// 2. Process Timed out (Infinite loop)
	if res.TimedOut {
		return &model.TestResult{
			Case:        tc,
			Status:      model.StatusTimeout,
			Duration:    res.Duration,
			Explanation: res.Explanation,
			Advice:      "Check loop termination conditions (e.g. format string parsing while (*format)).",
		}, nil
	}

	// 3. Memory leak / corruption reported by AddressSanitizer
	if res.Status == model.StatusLeak {
		return &model.TestResult{
			Case:        tc,
			Status:      model.StatusLeak,
			Duration:    res.Duration,
			Explanation: res.Explanation,
			LeaksReport: res.LeakReport,
			Advice:      "Ensure all memory allocated during conversion (e.g. itoa buffers) is freed before returning.",
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
