package codexion

import (
	"context"
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
)

type Suite struct {
	binPath string
	cases   map[string]TestCaseDef
}

func init() {
	projects.Register(&Suite{})
}

func (s *Suite) ID() model.ProjectID {
	return model.ProjectCodexion
}

func (s *Suite) Metadata() model.ProjectMetadata {
	return detector.KnownProjects[model.ProjectCodexion]
}

func (s *Suite) getBinPath(dir string) string {
	if s.binPath != "" && filepath.IsAbs(s.binPath) {
		return s.binPath
	}
	abs, err := filepath.Abs(filepath.Join(dir, "codexion"))
	if err == nil {
		return abs
	}
	return filepath.Join(dir, "codexion")
}

func (s *Suite) Prepare(ctx context.Context, dir string, cfg *config.Config, b *builder.Builder) (*model.TestResult, error) {
	binPath := s.getBinPath(dir)
	s.binPath = binPath

	// 1. Check if Makefile exists
	makefilePath := filepath.Join(dir, "Makefile")
	if _, err := os.Stat(makefilePath); err != nil {
		if _, errBin := os.Stat(binPath); errBin == nil {
			return &model.TestResult{
				Case: model.TestCase{
					ID:          "codexion_prebuilt",
					Module:      "Binary",
					File:        "codexion",
					Name:        "Pre-existing binary",
					Category:    model.CategoryBuild,
					Description: "Using existing codexion binary without Makefile",
				},
				Status: model.StatusPassed,
			}, nil
		}
		return &model.TestResult{
			Case: model.TestCase{
				ID:          "codexion_build_makefile_missing",
				Module:      "Makefile",
				File:        "Makefile",
				Name:        "Makefile existence",
				Category:    model.CategoryBuild,
				Description: "Verifies that Makefile is present at repository root",
			},
			Status:      model.StatusFailed,
			Explanation: "No Makefile found in " + dir,
			Advice:      "Add a Makefile at the project root with 'all', 'clean', 'fclean', 're' rules.",
		}, nil
	}

	// 2. Compile codexion
	out, err := b.ExecuteMake(ctx, dir, "all", 30*time.Second)
	if err != nil {
		return &model.TestResult{
			Case: model.TestCase{
				ID:          "codexion_build_compile",
				Module:      "Makefile",
				File:        "Makefile",
				Name:        "make all compilation",
				Category:    model.CategoryBuild,
				Description: "Compiles codexion binary with -Wall -Wextra -Werror",
			},
			Status:      model.StatusFailed,
			Explanation: fmt.Sprintf("Failed to compile codexion:\n%s", out),
			Advice:      "Fix compilation errors or missing pthread libraries.",
		}, nil
	}

	if _, err := os.Stat(binPath); err != nil {
		return &model.TestResult{
			Case: model.TestCase{
				ID:          "codexion_build_binary_missing",
				Module:      "Makefile",
				File:        "Makefile",
				Name:        "Target executable existence",
				Category:    model.CategoryBuild,
				Description: "Verifies that 'make all' produces the 'codexion' binary",
			},
			Status:      model.StatusFailed,
			Explanation: "make completed but 'codexion' binary was not found in " + dir,
			Advice:      "Ensure Makefile target NAME is set to 'codexion'.",
		}, nil
	}

	return &model.TestResult{
		Case: model.TestCase{
			ID:          "codexion_build_success",
			Module:      "Makefile",
			File:        "Makefile",
			Name:        "Build & Makefile verification",
			Category:    model.CategoryBuild,
			Description: "codexion built successfully with -Wall -Wextra -Werror",
		},
		Status: model.StatusPassed,
	}, nil
}

func (s *Suite) Plan(ctx context.Context, dir string, cfg *config.Config) ([]model.TestCase, error) {
	defs := GetDefaultCases()

	s.cases = make(map[string]TestCaseDef)
	var plan []model.TestCase
	for _, d := range defs {
		s.cases[d.ID] = d
		plan = append(plan, model.TestCase{
			ID:          d.ID,
			Module:      d.File,
			File:        d.File,
			Name:        d.Name,
			Description: d.Description,
			Category:    d.Category,
			Timeout:     d.Timeout,
		})
	}

	return plan, nil
}

func (s *Suite) RunCase(ctx context.Context, dir string, tc model.TestCase, cfg *config.Config, b *builder.Builder, r *runner.Runner) (*model.TestResult, error) {
	def, ok := s.cases[tc.ID]
	if !ok {
		return &model.TestResult{
			Case:        tc,
			Status:      model.StatusFailed,
			Explanation: fmt.Sprintf("Unknown test case ID: %s", tc.ID),
		}, nil
	}

	binPath := s.getBinPath(dir)
	cmdSnippet := fmt.Sprintf("./codexion %s", strings.Join(def.Args, " "))

	// Memory leak case
	if def.Kind == KindMemory {
		out, hasLeaks, err := r.RunMemoryCheck(ctx, binPath, def.Args...)
		if err != nil {
			return &model.TestResult{
				Case:        tc,
				Status:      model.StatusSkipped,
				Explanation: fmt.Sprintf("Memory checker unavailable: %v", err),
			}, nil
		}
		if hasLeaks {
			return &model.TestResult{
				Case:        tc,
				Status:      model.StatusLeak,
				Explanation: "Memory leak detected during codexion simulation",
				Advice:      "Ensure all mutexes, allocated coders, and thread arrays are cleanly freed at exit.",
				LeaksReport: out,
				CodeSnippet: cmdSnippet,
			}, nil
		}
		return &model.TestResult{
			Case:        tc,
			Status:      model.StatusPassed,
			CodeSnippet: cmdSnippet,
		}, nil
	}

	// Standard simulation or error case
	res, err := r.Run(ctx, binPath, runner.RunOptions{
		Args:    def.Args,
		Timeout: def.Timeout,
	})
	if err != nil {
		return &model.TestResult{
			Case:        tc,
			Status:      model.StatusFailed,
			Explanation: fmt.Sprintf("Execution runner error: %v", err),
			CodeSnippet: cmdSnippet,
		}, nil
	}

	if res.TimedOut {
		return &model.TestResult{
			Case:        tc,
			Status:      model.StatusTimeout,
			Explanation: fmt.Sprintf("Test timed out after %v (deadlock or infinite loop)", def.Timeout),
			Advice:      "Check for thread deadlocks, missing unlock on mutexes, or missed termination conditions.",
			Duration:    res.Duration,
			CodeSnippet: cmdSnippet,
			Actual:      previewOutput(res.Stdout),
		}, nil
	}

	if res.Crashed {
		return &model.TestResult{
			Case:        tc,
			Status:      model.StatusCrash,
			Explanation: fmt.Sprintf("Process crashed with %s (exit code %d)", res.Signal, res.ExitCode),
			Advice:      "Run under lldb or with -fsanitize=thread to diagnose crash or data races.",
			Duration:    res.Duration,
			ExitCode:    res.ExitCode,
			Signal:      res.Signal,
			CodeSnippet: cmdSnippet,
			Trace:       res.Stderr,
			Actual:      previewOutput(res.Stdout),
		}, nil
	}

	// Check for ThreadSanitizer or AddressSanitizer output in stderr
	lowStderr := strings.ToLower(res.Stderr)
	if strings.Contains(lowStderr, "threadsanitizer") || strings.Contains(lowStderr, "data race") {
		return &model.TestResult{
			Case:        tc,
			Status:      model.StatusCrash,
			Explanation: "ThreadSanitizer reported a data race",
			Advice:      "All shared variables (dongles, state flags, timestamps) must be protected by mutexes.",
			Duration:    res.Duration,
			CodeSnippet: cmdSnippet,
			Trace:       res.Stderr,
			Actual:      previewOutput(res.Stdout),
		}, nil
	}
	if strings.Contains(lowStderr, "addresssanitizer") {
		return &model.TestResult{
			Case:        tc,
			Status:      model.StatusCrash,
			Explanation: "AddressSanitizer reported a memory error",
			Advice:      "Check for heap-use-after-free or buffer overflows in coder structs.",
			Duration:    res.Duration,
			CodeSnippet: cmdSnippet,
			Trace:       res.Stderr,
			Actual:      previewOutput(res.Stdout),
		}, nil
	}

	// 1. Error Cases (Syntax / Logic)
	if def.ExpectError {
		vRes := ValidateErrorCase(res.ExitCode, res.Stdout, res.Stderr)
		if vRes.Status == model.StatusFailed {
			return &model.TestResult{
				Case:        tc,
				Status:      model.StatusFailed,
				Explanation: vRes.Explanation,
				Advice:      vRes.Advice,
				Duration:    res.Duration,
				ExitCode:    res.ExitCode,
				CodeSnippet: cmdSnippet,
				Expected:    "Non-zero exit code with error message",
				Actual:      fmt.Sprintf("exit code %d\nstdout: %s\nstderr: %s", res.ExitCode, previewOutput(res.Stdout), previewOutput(res.Stderr)),
			}, nil
		}
		return &model.TestResult{
			Case:        tc,
			Status:      model.StatusPassed,
			Duration:    res.Duration,
			ExitCode:    res.ExitCode,
			CodeSnippet: cmdSnippet,
		}, nil
	}

	// 2. Nominal, Burnout, Stress Simulations
	if res.ExitCode != 0 {
		return &model.TestResult{
			Case:        tc,
			Status:      model.StatusFailed,
			Explanation: fmt.Sprintf("Process exited unexpectedly with non-zero exit code %d", res.ExitCode),
			Advice:      "A simulation should exit with 0 upon completion or upon coder burnout.",
			Duration:    res.Duration,
			ExitCode:    res.ExitCode,
			CodeSnippet: cmdSnippet,
			Trace:       res.Stderr,
			Actual:      previewOutput(res.Stdout),
		}, nil
	}

	params, err := ParseCaseArgs(def.Args)
	if err != nil {
		return &model.TestResult{
			Case:        tc,
			Status:      model.StatusFailed,
			Explanation: fmt.Sprintf("Internal test specification error: %v", err),
			CodeSnippet: cmdSnippet,
		}, nil
	}

	rawLines := strings.TrimSpace(res.Stdout)
	if rawLines == "" {
		if params.NumCompiles == 0 {
			return &model.TestResult{
				Case:        tc,
				Status:      model.StatusPassed,
				Duration:    res.Duration,
				CodeSnippet: cmdSnippet,
			}, nil
		}
		return &model.TestResult{
			Case:        tc,
			Status:      model.StatusFailed,
			Explanation: "Process exited with 0 but emitted no logs at all",
			Advice:      "Coders must print their state changes to standard output.",
			Duration:    res.Duration,
			CodeSnippet: cmdSnippet,
			Expected:    "Simulation logs '<timestamp_ms> <coder_id> <state>'",
			Actual:      "<empty output>",
		}, nil
	}

	entries, parseWarnings, parseErr := ParseOutputLogs(res.Stdout, params.NumCoders)
	if parseErr != nil {
		return &model.TestResult{
			Case:        tc,
			Status:      parseErr.Status,
			Explanation: parseErr.Explanation,
			Advice:      parseErr.Advice,
			Expected:    parseErr.Expected,
			Actual:      parseErr.Actual,
			Duration:    res.Duration,
			CodeSnippet: cmdSnippet,
		}, nil
	}

	simRes := ValidateSimulation(params, entries, def.ExpectBurnout, def.ExpectNoBurnout, parseWarnings)
	if simRes.Status == model.StatusFailed {
		return &model.TestResult{
			Case:        tc,
			Status:      model.StatusFailed,
			Explanation: simRes.Explanation,
			Advice:      simRes.Advice,
			Duration:    res.Duration,
			CodeSnippet: cmdSnippet,
			Actual:      previewOutput(res.Stdout),
		}, nil
	}

	return &model.TestResult{
		Case:        tc,
		Status:      model.StatusPassed,
		Duration:    res.Duration,
		CodeSnippet: cmdSnippet,
	}, nil
}

func (s *Suite) Cleanup(ctx context.Context, dir string) error {
	return nil
}

func previewOutput(out string) string {
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) <= 10 {
		return strings.Join(lines, "\n")
	}
	return strings.Join(lines[:8], "\n") + fmt.Sprintf("\n... (%d more lines truncated)", len(lines)-8)
}
