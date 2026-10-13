package libft

import (
	"context"
	_ "embed"
	"fmt"
	"os"
	"os/exec"
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
	return model.ProjectLibft
}

func (s *Suite) Metadata() model.ProjectMetadata {
	return detector.KnownProjects[model.ProjectLibft]
}

func (s *Suite) Prepare(ctx context.Context, dir string, cfg *config.Config, b *builder.Builder) (*model.TestResult, error) {
	// 1. Check Makefile compliance
	check, err := b.CheckMakefile(ctx, dir, "libft.a")
	if err != nil {
		return &model.TestResult{
			Case: model.TestCase{
				ID:          "libft_build_makefile",
				Module:      "Makefile",
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
				ID:          "libft_build_rules",
				Module:      "Makefile",
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
				ID:          "libft_build_relink",
				Module:      "Makefile",
				Name:        "Relink verification",
				Category:    model.CategoryBuild,
				Description: "Makefile must not relink when no files are modified",
			},
			Status:      model.StatusFailed,
			Explanation: "Makefile recompiled object files or archive even though no sources changed.",
			Advice:      "Ensure object targets have proper prerequisites and 'all' depends on $(NAME).",
		}, nil
	}

	// 2. Compile libft.a (or update if sources were modified)
	libPath := filepath.Join(dir, "libft.a")
	makeOut, err := b.ExecuteMake(ctx, dir, "all", 30*time.Second)
	if err != nil {
		return &model.TestResult{
			Case: model.TestCase{
				ID:          "libft_build_compile",
				Module:      "Makefile",
				Name:        "make all compilation",
				Category:    model.CategoryBuild,
				Description: "Compiles libft.a archive",
			},
			Status:      model.StatusFailed,
			Explanation: fmt.Sprintf("Failed to compile libft.a:\n%s", makeOut),
			Advice:      "Fix compilation errors or missing dependencies.",
		}, nil
	}
	if _, err := os.Stat(libPath); os.IsNotExist(err) {
		return &model.TestResult{
			Case: model.TestCase{
				ID:          "libft_build_compile",
				Module:      "Makefile",
				Name:        "make all compilation",
				Category:    model.CategoryBuild,
				Description: "Compiles libft.a archive",
			},
			Status:      model.StatusFailed,
			Explanation: "Rule 'all' completed but libft.a was not created.",
			Advice:      "Ensure $(NAME) produces libft.a.",
		}, nil
	}

	// If bonus files exist, build bonus archive so libft.a contains both mandatory & bonus objects
	bonusFile := filepath.Join(dir, "ft_lstnew_bonus.c")
	if _, err := os.Stat(bonusFile); err == nil {
		_ = os.Remove(libPath)
		_, _ = b.ExecuteMake(ctx, dir, "bonus", 30*time.Second)
	}

	// 3. Write and compile C harness
	workDir := filepath.Join(dir, ".repaco")
	if err := os.MkdirAll(workDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create .repaco directory: %w", err)
	}

	harnessSrcPath := filepath.Join(workDir, "libft_harness.c")
	if err := os.WriteFile(harnessSrcPath, []byte(harnessSource), 0644); err != nil {
		return nil, fmt.Errorf("failed to write harness source: %w", err)
	}

	s.harnessPath = filepath.Join(workDir, "libft_harness")
	extraFlags := []string{"-fsanitize=address,undefined", "-g"}

	// Check if bonus functions were compiled into libft.a
	nmCmd := exec.CommandContext(ctx, "nm", libPath)
	if nmOut, err := nmCmd.Output(); err == nil {
		if strings.Contains(string(nmOut), "ft_lstnew") {
			extraFlags = append(extraFlags, "-DHAS_BONUS=1")
		}
	}

	linkLibs := []string{"-L" + dir, "-lft"}
	includeDirs := []string{dir}

	harnessOut, err := b.CompileHarness(ctx, harnessSrcPath, s.harnessPath, includeDirs, extraFlags, linkLibs)
	if err != nil {
		return &model.TestResult{
			Case: model.TestCase{
				ID:          "libft_build_harness",
				Module:      "Makefile",
				Name:        "Test harness compilation",
				Category:    model.CategoryBuild,
				Description: "Compiles test harness linking against libft.a with AddressSanitizer",
			},
			Status:      model.StatusFailed,
			Explanation: fmt.Sprintf("Failed to link against libft.a:\n%s", harnessOut),
			Advice:      "Check that libft.a exports all prototypes declared in libft.h without missing symbols.",
		}, nil
	}

	return &model.TestResult{
		Case: model.TestCase{
			ID:          "libft_build_success",
			Module:      "Makefile",
			Name:        "Build & Makefile verification",
			Category:    model.CategoryBuild,
			Description: "libft.a built cleanly with -Wall -Wextra -Werror, without relink",
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
			Advice:      "Check for null pointer dereference, out-of-bounds array access, or uninitialized memory.",
		}, nil
	}

	// 2. Process Timed out (Infinite loop)
	if res.TimedOut {
		return &model.TestResult{
			Case:        tc,
			Status:      model.StatusTimeout,
			Duration:    res.Duration,
			Explanation: res.Explanation,
			Advice:      "Check loop termination conditions (e.g., pointer increment or boundary checks).",
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
			Advice:      "Ensure all memory allocated with malloc() is properly freed before returning.",
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
	_ = os.RemoveAll(workDir)
	return nil
}
