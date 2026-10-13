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

type CaseKind int

const (
	KindSyntaxError CaseKind = iota
	KindLogicError
	KindNominal
	KindBurnout
	KindStress
	KindMemory
)

type TestCaseDef struct {
	ID              string
	File            string
	Name            string
	Category        model.Category
	Description     string
	Kind            CaseKind
	Args            []string
	Timeout         time.Duration
	ExpectBurnout   bool
	ExpectNoBurnout bool
	ExpectError     bool
}

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
	var defs []TestCaseDef

	// --- 1. Argument Syntax Cases (codexion_syntax.c) ---
	syntaxCases := []struct {
		id, name, desc string
		args           []string
	}{
		{"syntax_coder_alpha", "Non-numeric coders count ('banana')", "Invalid number_of_coders string must be rejected", []string{"banana", "200", "300", "400", "500", "5", "10", "fifo"}},
		{"syntax_burnout_alpha", "Non-numeric burnout time ('banana')", "Invalid time_to_burnout string must be rejected", []string{"10", "banana", "300", "400", "500", "5", "10", "fifo"}},
		{"syntax_compile_alpha", "Non-numeric compile time ('banana')", "Invalid time_to_compile string must be rejected", []string{"10", "200", "banana", "400", "500", "5", "10", "fifo"}},
		{"syntax_debug_alpha", "Non-numeric debug time ('banana')", "Invalid time_to_debug string must be rejected", []string{"10", "200", "300", "banana", "500", "5", "10", "fifo"}},
		{"syntax_refactor_alpha", "Non-numeric refactor time ('banana')", "Invalid time_to_refactor string must be rejected", []string{"10", "200", "300", "400", "banana", "5", "10", "fifo"}},
		{"syntax_compiles_alpha", "Non-numeric compiles count ('banana')", "Invalid number_of_compiles string must be rejected", []string{"10", "200", "300", "400", "500", "banana", "10", "fifo"}},
		{"syntax_cooldown_alpha", "Non-numeric dongle cooldown ('banana')", "Invalid dongle_cooldown string must be rejected", []string{"10", "200", "300", "400", "500", "5", "banana", "fifo"}},
		{"syntax_scheduler_invalid", "Invalid scheduler name ('banana')", "Invalid scheduler string must be rejected", []string{"10", "200", "300", "400", "500", "5", "10", "banana"}},
		{"syntax_float_arg", "Floating point argument ('800.5')", "Floating point argument must be safely rejected", []string{"4", "800.5", "200", "200", "200", "5", "10", "fifo"}},
		{"syntax_overflow", "Integer overflow (999999999999999999999999999999)", "Number exceeding integer limits must be safely rejected", []string{"4", "999999999999999999999999999999", "200", "200", "200", "5", "10", "fifo"}},
	}
	for _, sc := range syntaxCases {
		defs = append(defs, TestCaseDef{
			ID:          "codexion_" + sc.id,
			File:        "codexion_syntax.c",
			Name:        sc.name,
			Category:    model.CategoryEdgeCase,
			Description: sc.desc,
			Kind:        KindSyntaxError,
			Args:        sc.args,
			Timeout:     3 * time.Second,
			ExpectError: true,
		})
	}

	// --- 2. Argument Logic Cases (codexion_logic.c) ---
	logicCases := []struct {
		id, name, desc string
		args           []string
	}{
		{"logic_negative_duration", "Negative duration parameter (-400ms)", "Negative duration must be rejected", []string{"10", "200", "300", "-400", "500", "5", "10", "edf"}},
		{"logic_too_many_args", "Too many arguments provided (9 args)", "Command line with extra arguments must be rejected", []string{"too", "10", "200", "300", "400", "500", "5", "10", "edf"}},
		{"logic_zero_coders", "Zero coders count ('0')", "0 coders must be rejected", []string{"0", "800", "200", "200", "200", "5", "10", "fifo"}},
		{"logic_negative_coders", "Negative coders count ('-5')", "Negative number of coders must be rejected", []string{"-5", "800", "200", "200", "200", "5", "10", "fifo"}},
		{"logic_caps_scheduler", "Uppercase scheduler name ('FIFO')", "Uppercase scheduler must be rejected (subject requires 'fifo')", []string{"4", "800", "200", "200", "200", "5", "10", "FIFO"}},
		{"logic_empty_arg", "Empty string argument ('')", "Empty string parameter must be rejected", []string{"4", "", "200", "200", "200", "5", "10", "fifo"}},
		{"logic_leading_spaces", "Leading spaces in argument (' 200')", "Leading space in number parameter must be rejected", []string{"4", "800", " 200", "200", "200", "5", "10", "fifo"}},
		{"logic_trailing_tab", "Trailing tab in argument ('200\\t')", "Trailing tab in number parameter must be rejected", []string{"4", "800", "200\t", "200", "200", "5", "10", "fifo"}},
		{"logic_bad_scheduler", "Unsupported scheduler ('round_robin')", "Unsupported scheduler name must be rejected", []string{"4", "800", "200", "200", "200", "5", "10", "round_robin"}},
		{"logic_missing_args", "Missing required scheduler argument", "Command line missing 8th argument must be rejected", []string{"10", "200", "300", "400", "500", "5", "10"}},
		{"logic_no_args", "No arguments provided at all", "Invoking binary with 0 arguments must display usage and exit", []string{}},
	}
	for _, lc := range logicCases {
		defs = append(defs, TestCaseDef{
			ID:          "codexion_" + lc.id,
			File:        "codexion_logic.c",
			Name:        lc.name,
			Category:    model.CategoryEdgeCase,
			Description: lc.desc,
			Kind:        KindLogicError,
			Args:        lc.args,
			Timeout:     3 * time.Second,
			ExpectError: true,
		})
	}

	// --- 3. Nominal Simulation Cases (codexion_nominal.c) ---
	nominalCases := []struct {
		id, name, desc string
		args           []string
		timeout        time.Duration
	}{
		{"nom_1_basic_fifo", "4 coders, FIFO scheduler (basic nominal)", "4 coders, balanced cycles under FIFO scheduler", []string{"4", "800", "200", "200", "200", "5", "10", "fifo"}, 6 * time.Second},
		{"nom_2_basic_edf", "4 coders, EDF scheduler (basic nominal)", "4 coders, balanced cycles under EDF scheduler", []string{"4", "800", "200", "200", "200", "5", "10", "edf"}, 6 * time.Second},
		{"nom_3_success_fifo", "10 coders, 5 compiles, FIFO", "10 coders, 5 compiles required each, verifies clean termination", []string{"10", "10000", "100", "100", "100", "5", "50", "fifo"}, 8 * time.Second},
		{"nom_4_large_edf", "20 coders, 500ms actions, EDF", "20 coders with 500ms actions under EDF scheduler", []string{"20", "5000", "500", "500", "500", "10", "100", "edf"}, 12 * time.Second},
		{"nom_5_low_cooldown", "5 coders, 1ms dongle cooldown", "5 coders with minimal 1ms cooldown to test fast handovers", []string{"5", "2000", "100", "100", "100", "20", "1", "fifo"}, 8 * time.Second},
		{"nom_6_long_actions", "3 coders, 600ms long action phases", "3 coders with long action phases (600ms each)", []string{"3", "4000", "600", "600", "600", "2", "50", "fifo"}, 8 * time.Second},
		{"nom_7_zero_compile", "0 compiles required: instant clean exit", "0 compiles required: coders must not compile and cleanly exit", []string{"5", "1000", "200", "200", "200", "0", "10", "fifo"}, 3 * time.Second},
		{"nom_8_tight_timings", "Tight margins (410ms burnout vs 300ms cycle)", "Tight timing margins under EDF: tests timer precision", []string{"4", "410", "100", "100", "100", "5", "5", "edf"}, 6 * time.Second},
	}
	for _, nc := range nominalCases {
		defs = append(defs, TestCaseDef{
			ID:              "codexion_" + nc.id,
			File:            "codexion_nominal.c",
			Name:            nc.name,
			Category:        model.CategoryMandatory,
			Description:     nc.desc,
			Kind:            KindNominal,
			Args:            nc.args,
			Timeout:         nc.timeout,
			ExpectNoBurnout: true,
		})
	}

	// --- 4. Burnout & Starvation Cases (codexion_burnout.c) ---
	burnoutCases := []struct {
		id, name, desc string
		args           []string
		timeout        time.Duration
	}{
		{"burn_starvation_fifo", "Neighbor starvation (FIFO)", "3 coders with 600ms compile starves neighbors, burnout expected (FIFO)", []string{"3", "1000", "600", "10", "10", "5", "100", "fifo"}, 5 * time.Second},
		{"burn_starvation_edf", "Neighbor starvation (EDF)", "3 coders with 600ms compile starves neighbors, burnout expected (EDF)", []string{"3", "1000", "600", "10", "10", "5", "100", "edf"}, 5 * time.Second},
		{"burn_one_coder_fifo", "1 single coder on table (FIFO)", "1 coder cannot acquire 2 dongles; must starve and burn out (FIFO)", []string{"1", "1000", "200", "200", "200", "5", "50", "fifo"}, 4 * time.Second},
		{"burn_one_coder_edf", "1 single coder on table (EDF)", "1 coder cannot acquire 2 dongles; must starve and burn out (EDF)", []string{"1", "1000", "200", "200", "200", "5", "50", "edf"}, 4 * time.Second},
		{"burn_immediate", "Immediate burnout (1ms burnout time)", "1ms burnout: coder must burn out immediately (<=10ms delay)", []string{"2", "1", "200", "200", "200", "5", "10", "fifo"}, 3 * time.Second},
		{"burn_cooldown_hell", "High cooldown starvation (2000ms cooldown)", "2000ms cooldown: dongles unavailable in time, burnout expected", []string{"2", "1000", "100", "100", "100", "5", "2000", "fifo"}, 5 * time.Second},
	}
	for _, bc := range burnoutCases {
		defs = append(defs, TestCaseDef{
			ID:            "codexion_" + bc.id,
			File:          "codexion_burnout.c",
			Name:          bc.name,
			Category:      model.CategoryMandatory,
			Description:   bc.desc,
			Kind:          KindBurnout,
			Args:          bc.args,
			Timeout:       bc.timeout,
			ExpectBurnout: true,
		})
	}

	// --- 5. Concurrency & Contention Stress Cases (codexion_stress.c) ---
	stressCases := []struct {
		id, name, desc  string
		args            []string
		timeout         time.Duration
		expectNoBurnout bool
	}{
		{"stress_100_coders", "100 simultaneous coders concurrency stress", "Concurrency stress test with 100 simultaneous threads", []string{"100", "10000", "66", "24", "87", "10", "10", "fifo"}, 10 * time.Second, true},
		{"stress_300_coders", "300 coders scalability under EDF", "High thread count scalability under EDF scheduler", []string{"300", "10000", "100", "100", "100", "5", "10", "edf"}, 12 * time.Second, true},
		{"stress_999_threads", "999 threads extreme contention test", "Extreme thread contention (999 coders) to catch data races", []string{"999", "1000", "200", "200", "200", "5", "50", "fifo"}, 10 * time.Second, false},
	}
	for _, st := range stressCases {
		defs = append(defs, TestCaseDef{
			ID:              "codexion_" + st.id,
			File:            "codexion_stress.c",
			Name:            st.name,
			Category:        model.CategoryBenchmark,
			Description:     st.desc,
			Kind:            KindStress,
			Args:            st.args,
			Timeout:         st.timeout,
			ExpectNoBurnout: st.expectNoBurnout,
		})
	}

	// --- 6. Memory & Data Race Safety (codexion_memory.c) ---
	defs = append(defs, TestCaseDef{
		ID:              "codexion_memory_leak",
		File:            "codexion_memory.c",
		Name:            "Memory leak check (macOS leaks)",
		Category:        model.CategoryMemory,
		Description:     "Runs 4 coders simulation and verifies zero memory leaks on completion",
		Kind:            KindMemory,
		Args:            []string{"4", "800", "100", "100", "100", "2", "10", "fifo"},
		Timeout:         8 * time.Second,
		ExpectNoBurnout: true,
	})

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
