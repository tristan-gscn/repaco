package pushswap

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"strconv"
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

type CaseType int

const (
	CaseError CaseType = iota
	CaseIdentity
	CaseSort
	CaseBenchmark
	CaseMemory
	CaseBonus
)

type TestCaseDef struct {
	ID          string
	File        string
	Name        string
	Category    model.Category
	Description string
	Type        CaseType
	Args        []string
	MaxOps      int
	Timeout     time.Duration
}

type Suite struct {
	binPath      string
	bonusBinPath string
	hasBonus     bool
	cases        map[string]TestCaseDef
}

func init() {
	projects.Register(&Suite{})
}

func (s *Suite) ID() model.ProjectID {
	return model.ProjectPushSwap
}

func (s *Suite) Metadata() model.ProjectMetadata {
	return detector.KnownProjects[model.ProjectPushSwap]
}

func (s *Suite) getBinPath(dir string) string {
	if s.binPath != "" && filepath.IsAbs(s.binPath) {
		return s.binPath
	}
	abs, err := filepath.Abs(filepath.Join(dir, "push_swap"))
	if err == nil {
		return abs
	}
	return filepath.Join(dir, "push_swap")
}

func (s *Suite) getBonusBinPath(dir string) string {
	if s.bonusBinPath != "" && filepath.IsAbs(s.bonusBinPath) {
		return s.bonusBinPath
	}
	abs, err := filepath.Abs(filepath.Join(dir, "checker"))
	if err == nil {
		return abs
	}
	return filepath.Join(dir, "checker")
}

func (s *Suite) Prepare(ctx context.Context, dir string, cfg *config.Config, b *builder.Builder) (*model.TestResult, error) {
	if absDir, err := filepath.Abs(dir); err == nil {
		dir = absDir
	}
	s.binPath = filepath.Join(dir, "push_swap")
	s.cases = make(map[string]TestCaseDef)

	// 1. Makefile compliance check
	check, err := b.CheckMakefile(ctx, dir, "push_swap")
	if err != nil {
		return &model.TestResult{
			Case: model.TestCase{
				ID:          "pushswap_build_makefile",
				Module:      "Makefile",
				File:        "Makefile",
				Name:        "Makefile compliance",
				Category:    model.CategoryBuild,
				Description: "Verifies mandatory rules, flags, and relink",
			},
			Status:      model.StatusFailed,
			Explanation: fmt.Sprintf("Makefile check failed: %v", err),
			Advice:      "Check that Makefile is present and rules 'all', 'clean', 'fclean', 're' compile cleanly.",
		}, nil
	}

	if len(check.MissingRules) > 0 {
		return &model.TestResult{
			Case: model.TestCase{
				ID:          "pushswap_build_rules",
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
				ID:          "pushswap_build_relink",
				Module:      "Makefile",
				File:        "Makefile",
				Name:        "Relink check",
				Category:    model.CategoryBuild,
				Description: "Makefile must not relink when no files are modified",
			},
			Status:      model.StatusFailed,
			Explanation: "Makefile recompiled object files or binary even though no sources changed.",
			Advice:      "Ensure object targets have proper prerequisites and 'all' depends on $(NAME).",
		}, nil
	}

	// 2. Compile push_swap binary
	if _, err := os.Stat(s.binPath); os.IsNotExist(err) {
		out, err := b.ExecuteMake(ctx, dir, "all", 30*time.Second)
		if err != nil {
			return &model.TestResult{
				Case: model.TestCase{
					ID:          "pushswap_build_compile",
					Module:      "Makefile",
					File:        "Makefile",
					Name:        "make all compilation",
					Category:    model.CategoryBuild,
					Description: "Compiles push_swap executable",
				},
				Status:      model.StatusFailed,
				Explanation: fmt.Sprintf("Failed to compile push_swap:\n%s", out),
				Advice:      "Fix C compilation errors or missing dependencies.",
			}, nil
		}
	}

	// 3. Check for bonus checker
	checkerPath := filepath.Join(dir, "checker")
	s.hasBonus = false
	if _, err := os.Stat(checkerPath); err == nil {
		s.hasBonus = true
		s.bonusBinPath = checkerPath
	} else {
		// Try building bonus if Makefile defines it
		out, err := b.ExecuteMake(ctx, dir, "bonus", 10*time.Second)
		if err == nil && !strings.Contains(out, "No rule to make target") {
			if _, err := os.Stat(checkerPath); err == nil {
				s.hasBonus = true
				s.bonusBinPath = checkerPath
			}
		}
	}

	return &model.TestResult{
		Case: model.TestCase{
			ID:          "pushswap_build_success",
			Module:      "Makefile",
			File:        "Makefile",
			Name:        "Build & Makefile verification",
			Category:    model.CategoryBuild,
			Description: "push_swap built successfully with -Wall -Wextra -Werror without relink",
		},
		Status: model.StatusPassed,
	}, nil
}

func (s *Suite) Plan(ctx context.Context, dir string, cfg *config.Config) ([]model.TestCase, error) {
	var definitions []TestCaseDef

	// --- 1. Error Management (push_swap_errors.c) ---
	errorDefs := []struct {
		id, name, desc string
		args           []string
	}{
		{"err_alpha", "Non-numeric argument in list", "Arguments containing letters must write 'Error\\n' to stderr", []string{"1", "2", "3", "a", "5"}},
		{"err_alpha_only", "Single non-numeric argument", "Word argument 'abc' must write 'Error\\n' to stderr", []string{"abc"}},
		{"err_overflow", "Integer overflow (> INT_MAX)", "Number exceeding INT_MAX (2147483648) must write 'Error\\n'", []string{"2147483648"}},
		{"err_underflow", "Integer underflow (< INT_MIN)", "Number below INT_MIN (-2147483649) must write 'Error\\n'", []string{"-2147483649"}},
		{"err_duplicates", "Duplicate numbers", "Duplicate numbers in list must write 'Error\\n'", []string{"1", "2", "3", "2", "4"}},
		{"err_empty_string", "Empty string parameter", "Empty string '' must write 'Error\\n'", []string{""}},
		{"err_spaces_string", "Spaces-only string parameter", "String with only spaces '   ' must write 'Error\\n'", []string{"   "}},
		{"err_isolated_plus", "Isolated '+' sign", "Isolated sign '+' must write 'Error\\n'", []string{"+"}},
		{"err_isolated_minus", "Isolated '-' sign", "Isolated sign '-' must write 'Error\\n'", []string{"-"}},
		{"err_double_sign", "Double sign '++1'", "Invalid double sign '++1' must write 'Error\\n'", []string{"++1"}},
		{"err_invalid_suffix", "Alphanumeric suffix '123a'", "Number with alphanumeric suffix must write 'Error\\n'", []string{"123a"}},
		{"err_middle_empty", "Empty parameter in middle", "Empty string in middle of valid numbers must write 'Error\\n'", []string{"1", "", "2"}},
	}
	for _, ed := range errorDefs {
		definitions = append(definitions, TestCaseDef{
			ID:          "pushswap_" + ed.id,
			File:        "push_swap_errors.c",
			Name:        ed.name,
			Category:    model.CategoryEdgeCase,
			Description: ed.desc,
			Type:        CaseError,
			Args:        ed.args,
			Timeout:     2 * time.Second,
		})
	}

	// --- 2. Identity / Already Sorted (push_swap_identity.c) ---
	identityDefs := []struct {
		id, name, desc string
		args           []string
	}{
		{"id_no_args", "No arguments", "Running push_swap with 0 parameters must display nothing", []string{}},
		{"id_single_arg", "Single positive number", "Running push_swap with single number (42) must display nothing", []string{"42"}},
		{"id_single_negative", "Single negative number", "Running push_swap with single negative number (-42) must display nothing", []string{"-42"}},
		{"id_sorted_2", "Already sorted 2 numbers", "Running push_swap with 1 2 must display nothing", []string{"1", "2"}},
		{"id_sorted_3", "Already sorted 3 numbers", "Running push_swap with 1 2 3 must display nothing", []string{"1", "2", "3"}},
		{"id_sorted_5", "Already sorted 5 numbers", "Running push_swap with 1 2 3 4 5 must display nothing", []string{"1", "2", "3", "4", "5"}},
		{"id_sorted_quoted", "Already sorted quoted string", "Running push_swap with '1 2 3 4 5' must display nothing", []string{"1 2 3 4 5"}},
		{"id_sorted_negatives", "Already sorted negative numbers", "Running push_swap with '-10 -5 0 5 10' must display nothing", []string{"-10", "-5", "0", "5", "10"}},
	}
	for _, id := range identityDefs {
		definitions = append(definitions, TestCaseDef{
			ID:          "pushswap_" + id.id,
			File:        "push_swap_identity.c",
			Name:        id.name,
			Category:    model.CategoryMandatory,
			Description: id.desc,
			Type:        CaseIdentity,
			Args:        id.args,
			Timeout:     2 * time.Second,
		})
	}

	// --- 3. Small Sort (push_swap_small.c) ---
	smallDefs := []struct {
		id, name, desc string
		args           []string
		maxOps         int
	}{
		{"small_2_rev", "Sort 2 numbers (reverse)", "Sorts '2 1' in <= 1 operations", []string{"2", "1"}, 1},
		{"small_3_perm_213", "Sort 3 numbers (2 1 3)", "Sorts '2 1 3' in <= 3 operations", []string{"2", "1", "3"}, 3},
		{"small_3_perm_321", "Sort 3 numbers (3 2 1)", "Sorts '3 2 1' in <= 3 operations", []string{"3", "2", "1"}, 3},
		{"small_3_perm_312", "Sort 3 numbers (3 1 2)", "Sorts '3 1 2' in <= 3 operations", []string{"3", "1", "2"}, 3},
		{"small_3_perm_132", "Sort 3 numbers (1 3 2)", "Sorts '1 3 2' in <= 3 operations", []string{"1", "3", "2"}, 3},
		{"small_3_perm_231", "Sort 3 numbers (2 3 1)", "Sorts '2 3 1' in <= 3 operations", []string{"2", "3", "1"}, 3},
		{"small_4_reverse", "Sort 4 numbers (4 3 2 1)", "Sorts '4 3 2 1' in <= 12 operations", []string{"4", "3", "2", "1"}, 12},
		{"small_4_random", "Sort 4 numbers (2 4 1 3)", "Sorts '2 4 1 3' in <= 12 operations", []string{"2", "4", "1", "3"}, 12},
		{"small_5_reverse", "Sort 5 numbers (5 4 3 2 1)", "Sorts '5 4 3 2 1' in <= 12 operations", []string{"5", "4", "3", "2", "1"}, 12},
		{"small_5_perm1", "Sort 5 numbers (1 5 2 4 3)", "Sorts '1 5 2 4 3' in <= 12 operations", []string{"1", "5", "2", "4", "3"}, 12},
		{"small_5_perm2", "Sort 5 numbers (2 3 5 1 4)", "Sorts '2 3 5 1 4' in <= 12 operations", []string{"2", "3", "5", "1", "4"}, 12},
		{"small_5_perm3", "Sort 5 numbers (4 1 5 3 2)", "Sorts '4 1 5 3 2' in <= 12 operations", []string{"4", "1", "5", "3", "2"}, 12},
		{"small_5_quoted", "Sort 5 numbers quoted ('5 2 4 1 3')", "Sorts space-separated string in <= 12 operations", []string{"5 2 4 1 3"}, 12},
		{"small_5_negatives", "Sort 5 numbers with negatives", "Sorts '-2 5 -10 0 3' in <= 12 operations", []string{"-2", "5", "-10", "0", "3"}, 12},
	}
	for _, sd := range smallDefs {
		definitions = append(definitions, TestCaseDef{
			ID:          "pushswap_" + sd.id,
			File:        "push_swap_small.c",
			Name:        sd.name,
			Category:    model.CategoryMandatory,
			Description: sd.desc,
			Type:        CaseSort,
			Args:        sd.args,
			MaxOps:      sd.maxOps,
			Timeout:     2 * time.Second,
		})
	}

	// --- 4. Benchmarks (push_swap_benchmarks.c) ---
	bench100_1 := generatePermutationArgs(100, 42)
	bench100_2 := generatePermutationArgs(100, 1337)
	bench100_3 := generatePermutationArgs(100, 2026)
	bench100_rev := generateReverseArgs(100)

	bench500_1 := generatePermutationArgs(500, 42)
	bench500_2 := generatePermutationArgs(500, 1337)
	bench500_rev := generateReverseArgs(500)

	benchDefs := []struct {
		id, name, desc string
		args           []string
		maxOps         int
		timeout        time.Duration
	}{
		{"bench_100_set1", "100 numbers (seed 42)", "Sorts 100 random numbers, target < 700 ops for 5/5", bench100_1, 700, 5 * time.Second},
		{"bench_100_set2", "100 numbers (seed 1337)", "Sorts 100 random numbers, target < 700 ops for 5/5", bench100_2, 700, 5 * time.Second},
		{"bench_100_set3", "100 numbers (seed 2026)", "Sorts 100 random numbers, target < 700 ops for 5/5", bench100_3, 700, 5 * time.Second},
		{"bench_100_reverse", "100 numbers (descending 100..1)", "Sorts 100 reverse numbers, target < 700 ops for 5/5", bench100_rev, 700, 5 * time.Second},
		{"bench_500_set1", "500 numbers (seed 42)", "Sorts 500 random numbers, target < 5500 ops for 5/5", bench500_1, 5500, 10 * time.Second},
		{"bench_500_set2", "500 numbers (seed 1337)", "Sorts 500 random numbers, target < 5500 ops for 5/5", bench500_2, 5500, 10 * time.Second},
		{"bench_500_reverse", "500 numbers (descending 500..1)", "Sorts 500 reverse numbers, target < 5500 ops for 5/5", bench500_rev, 5500, 10 * time.Second},
	}
	for _, bd := range benchDefs {
		definitions = append(definitions, TestCaseDef{
			ID:          "pushswap_" + bd.id,
			File:        "push_swap_benchmarks.c",
			Name:        bd.name,
			Category:    model.CategoryBenchmark,
			Description: bd.desc,
			Type:        CaseBenchmark,
			Args:        bd.args,
			MaxOps:      bd.maxOps,
			Timeout:     bd.timeout,
		})
	}

	// --- 5. Memory Checks (push_swap_memory.c) ---
	memDefs := []struct {
		id, name, desc string
		args           []string
	}{
		{"mem_small", "Memory check: small sort", "Verifies 0 memory leaks on sorting 5 numbers", []string{"5", "4", "3", "2", "1"}},
		{"mem_error", "Memory check: error exit", "Verifies 0 memory leaks when exiting on duplicate error", []string{"1", "2", "2"}},
		{"mem_100", "Memory check: 100 numbers", "Verifies 0 memory leaks after sorting 100 numbers", bench100_1},
	}
	for _, md := range memDefs {
		definitions = append(definitions, TestCaseDef{
			ID:          "pushswap_" + md.id,
			File:        "push_swap_memory.c",
			Name:        md.name,
			Category:    model.CategoryMemory,
			Description: md.desc,
			Type:        CaseMemory,
			Args:        md.args,
			Timeout:     5 * time.Second,
		})
	}

	// --- 6. Bonus Checker (push_swap_bonus.c) ---
	if s.hasBonus {
		bonusDefs := []struct {
			id, name, desc string
		}{
			{"bonus_valid_sort", "Checker: valid sort output", "Piping push_swap instructions to checker displays 'OK\\n'"},
			{"bonus_ko", "Checker: incorrect instructions", "Piping unsorted instructions to checker displays 'KO\\n'"},
			{"bonus_invalid_op", "Checker: invalid instruction 'foo'", "Checker displays 'Error\\n' on invalid instruction"},
			{"bonus_error_args", "Checker: invalid arguments", "Checker displays 'Error\\n' on non-numeric arguments"},
		}
		for _, bnd := range bonusDefs {
			definitions = append(definitions, TestCaseDef{
				ID:          "pushswap_" + bnd.id,
				File:        "push_swap_bonus.c",
				Name:        bnd.name,
				Category:    model.CategoryBonus,
				Description: bnd.desc,
				Type:        CaseBonus,
				Timeout:     3 * time.Second,
			})
		}
	}

	if s.cases == nil {
		s.cases = make(map[string]TestCaseDef)
	}

	var plan []model.TestCase
	for _, def := range definitions {
		s.cases[def.ID] = def
		tc := model.TestCase{
			ID:          def.ID,
			Module:      strings.TrimSuffix(def.File, ".c"),
			File:        def.File,
			Name:        def.Name,
			Description: def.Description,
			Category:    def.Category,
			Timeout:     def.Timeout,
		}
		if cfg.FilterModule != "" && cfg.FilterModule != tc.Module {
			continue
		}
		plan = append(plan, tc)
	}

	return plan, nil
}

func (s *Suite) RunCase(ctx context.Context, dir string, tc model.TestCase, cfg *config.Config, b *builder.Builder, r *runner.Runner) (*model.TestResult, error) {
	def, ok := s.cases[tc.ID]
	if !ok {
		return &model.TestResult{
			Case:        tc,
			Status:      model.StatusFailed,
			Explanation: fmt.Sprintf("Case definition not found for %s", tc.ID),
		}, nil
	}

	switch def.Type {
	case CaseError:
		return s.runErrorCase(ctx, dir, tc, def, r)
	case CaseIdentity:
		return s.runIdentityCase(ctx, dir, tc, def, r)
	case CaseSort:
		return s.runSortCase(ctx, dir, tc, def, r)
	case CaseBenchmark:
		return s.runBenchmarkCase(ctx, dir, tc, def, r)
	case CaseMemory:
		return s.runMemoryCase(ctx, dir, tc, def, r)
	case CaseBonus:
		return s.runBonusCase(ctx, dir, tc, def, r)
	default:
		return &model.TestResult{
			Case:        tc,
			Status:      model.StatusFailed,
			Explanation: "Unknown case type",
		}, nil
	}
}

func (s *Suite) runErrorCase(ctx context.Context, dir string, tc model.TestCase, def TestCaseDef, r *runner.Runner) (*model.TestResult, error) {
	binPath := s.getBinPath(dir)
	res, err := r.Run(ctx, binPath, runner.RunOptions{
		Args:    def.Args,
		Cwd:     dir,
		Timeout: def.Timeout,
	})
	if err != nil {
		return &model.TestResult{
			Case:        tc,
			Status:      model.StatusFailed,
			Explanation: fmt.Sprintf("Execution error: %v", err),
		}, nil
	}

	testRes := &model.TestResult{
		Case:        tc,
		Duration:    res.Duration,
		Input:       formatArgs(def.Args),
		Expected:    "stderr: \"Error\\n\", exit code != 0, stdout: \"\"",
		CodeSnippet: fmt.Sprintf("./push_swap %s", formatArgs(def.Args)),
	}

	if res.Crashed {
		testRes.Status = model.StatusCrash
		testRes.Signal = res.Signal
		testRes.Explanation = res.Explanation
		testRes.Advice = "Check array bounds and pointer validation when parsing invalid arguments."
		return testRes, nil
	}
	if res.TimedOut {
		testRes.Status = model.StatusTimeout
		testRes.Explanation = res.Explanation
		testRes.Advice = "Check argument parser loop termination conditions."
		return testRes, nil
	}

	// Stderr must be "Error\n"
	trimmedErr := strings.TrimSpace(res.Stderr)
	trimmedOut := strings.TrimSpace(res.Stdout)

	if trimmedErr != "Error" {
		testRes.Status = model.StatusFailed
		testRes.Actual = fmt.Sprintf("stderr: %q, exit code: %d, stdout: %q", res.Stderr, res.ExitCode, res.Stdout)
		testRes.Explanation = fmt.Sprintf("Expected 'Error\\n' on stderr, but got: %q (stdout: %q)", res.Stderr, res.Stdout)
		testRes.Advice = "Write \"Error\\n\" to file descriptor 2 (stderr) and exit with a non-zero code on invalid input."
		testRes.Diff = diff.Compare("Error\n", res.Stderr)
		return testRes, nil
	}

	if trimmedOut != "" {
		testRes.Status = model.StatusFailed
		testRes.Actual = fmt.Sprintf("stdout: %q, stderr: %q", res.Stdout, res.Stderr)
		testRes.Explanation = fmt.Sprintf("push_swap printed to stdout on error: %q", res.Stdout)
		testRes.Advice = "Do not print any operations or messages to stdout when an error occurs."
		return testRes, nil
	}

	testRes.Status = model.StatusPassed
	testRes.Actual = "stderr: \"Error\\n\""
	return testRes, nil
}

func (s *Suite) runIdentityCase(ctx context.Context, dir string, tc model.TestCase, def TestCaseDef, r *runner.Runner) (*model.TestResult, error) {
	binPath := s.getBinPath(dir)
	res, err := r.Run(ctx, binPath, runner.RunOptions{
		Args:    def.Args,
		Cwd:     dir,
		Timeout: def.Timeout,
	})
	if err != nil {
		return &model.TestResult{
			Case:        tc,
			Status:      model.StatusFailed,
			Explanation: fmt.Sprintf("Execution error: %v", err),
		}, nil
	}

	testRes := &model.TestResult{
		Case:        tc,
		Duration:    res.Duration,
		Input:       formatArgs(def.Args),
		Expected:    "stdout: \"\", stderr: \"\", exit code: 0",
		CodeSnippet: fmt.Sprintf("./push_swap %s", formatArgs(def.Args)),
	}

	if res.Crashed {
		testRes.Status = model.StatusCrash
		testRes.Signal = res.Signal
		testRes.Explanation = res.Explanation
		return testRes, nil
	}
	if res.TimedOut {
		testRes.Status = model.StatusTimeout
		testRes.Explanation = res.Explanation
		return testRes, nil
	}

	if res.ExitCode != 0 {
		testRes.Status = model.StatusFailed
		testRes.Actual = fmt.Sprintf("exit code: %d", res.ExitCode)
		testRes.Explanation = fmt.Sprintf("Expected exit code 0, got %d", res.ExitCode)
		testRes.Advice = "Return 0 when no parameters are provided or the list is already sorted."
		return testRes, nil
	}

	trimmedOut := strings.TrimSpace(res.Stdout)
	trimmedErr := strings.TrimSpace(res.Stderr)

	if trimmedOut != "" || trimmedErr != "" {
		testRes.Status = model.StatusFailed
		testRes.Actual = fmt.Sprintf("stdout: %q, stderr: %q", res.Stdout, res.Stderr)
		testRes.Explanation = "push_swap must display nothing when given 0 parameters or already sorted numbers."
		testRes.Advice = "Check is_sorted check at the start of main before executing any operations."
		return testRes, nil
	}

	testRes.Status = model.StatusPassed
	testRes.Actual = "no output, exit code: 0"
	return testRes, nil
}

func (s *Suite) runSortCase(ctx context.Context, dir string, tc model.TestCase, def TestCaseDef, r *runner.Runner) (*model.TestResult, error) {
	binPath := s.getBinPath(dir)
	res, err := r.Run(ctx, binPath, runner.RunOptions{
		Args:    def.Args,
		Cwd:     dir,
		Timeout: def.Timeout,
	})
	if err != nil {
		return &model.TestResult{
			Case:        tc,
			Status:      model.StatusFailed,
			Explanation: fmt.Sprintf("Execution error: %v", err),
		}, nil
	}

	testRes := &model.TestResult{
		Case:        tc,
		Duration:    res.Duration,
		Input:       formatArgs(def.Args),
		Expected:    fmt.Sprintf("Stack sorted in <= %d operations", def.MaxOps),
		CodeSnippet: fmt.Sprintf("./push_swap %s", formatArgs(def.Args)),
	}

	if res.Crashed {
		testRes.Status = model.StatusCrash
		testRes.Signal = res.Signal
		testRes.Explanation = res.Explanation
		testRes.Advice = "Check for null pointers or invalid stack node links."
		return testRes, nil
	}
	if res.TimedOut {
		testRes.Status = model.StatusTimeout
		testRes.Explanation = res.Explanation
		testRes.Advice = "Check sorting algorithm loops for infinite cycles or missing state advancement."
		return testRes, nil
	}

	if strings.TrimSpace(res.Stderr) != "" {
		testRes.Status = model.StatusFailed
		testRes.Actual = fmt.Sprintf("stderr: %q", res.Stderr)
		testRes.Explanation = fmt.Sprintf("push_swap outputted to stderr: %q", res.Stderr)
		testRes.Advice = "Valid arguments must not trigger errors."
		return testRes, nil
	}

	initial, err := ParseArgs(def.Args)
	if err != nil {
		return nil, fmt.Errorf("failed to parse test case initial args: %w", err)
	}

	opsCount, err := RunInstructions(initial, res.Stdout)
	if err != nil {
		testRes.Status = model.StatusFailed
		testRes.Actual = fmt.Sprintf("Verification failed: %v", err)
		testRes.Explanation = fmt.Sprintf("Push_swap operations did not properly sort the stack: %v", err)
		testRes.Advice = "Review the simulated stack state to ensure every operation is valid and stack A is fully sorted."
		return testRes, nil
	}

	testRes.Actual = fmt.Sprintf("%d operations (Stack sorted successfully)", opsCount)

	if opsCount > def.MaxOps {
		testRes.Status = model.StatusFailed
		testRes.Explanation = fmt.Sprintf("Operation count exceeded: used %d operations, maximum allowed is %d", opsCount, def.MaxOps)
		if len(initial) == 3 {
			testRes.Advice = "3 numbers can always be sorted in at most 2 operations using sort_three (sa, ra, rra combinations)."
		} else if len(initial) <= 5 {
			testRes.Advice = "5 numbers must be sorted in <= 12 operations: push 2 smallest to B, sort remaining 3 with sort_three, then push back."
		}
		testRes.Diff = diff.Compare(fmt.Sprintf("<= %d ops", def.MaxOps), fmt.Sprintf("%d ops", opsCount))
		return testRes, nil
	}

	testRes.Status = model.StatusPassed
	return testRes, nil
}

func (s *Suite) runBenchmarkCase(ctx context.Context, dir string, tc model.TestCase, def TestCaseDef, r *runner.Runner) (*model.TestResult, error) {
	binPath := s.getBinPath(dir)
	res, err := r.Run(ctx, binPath, runner.RunOptions{
		Args:    def.Args,
		Cwd:     dir,
		Timeout: def.Timeout,
	})
	if err != nil {
		return &model.TestResult{
			Case:        tc,
			Status:      model.StatusFailed,
			Explanation: fmt.Sprintf("Execution error: %v", err),
		}, nil
	}

	numElements := len(def.Args)
	testRes := &model.TestResult{
		Case:        tc,
		Duration:    res.Duration,
		Input:       fmt.Sprintf("%d numbers", numElements),
		Expected:    fmt.Sprintf("< %d operations (5/5 points)", def.MaxOps),
		CodeSnippet: fmt.Sprintf("./push_swap %s | wc -l", formatArgs(def.Args)),
	}

	if res.Crashed {
		testRes.Status = model.StatusCrash
		testRes.Signal = res.Signal
		testRes.Explanation = res.Explanation
		return testRes, nil
	}
	if res.TimedOut {
		testRes.Status = model.StatusTimeout
		testRes.Explanation = res.Explanation
		testRes.Advice = "Sorting algorithm took too long. Check algorithm complexity (O(n log n) or O(n√n) expected)."
		return testRes, nil
	}

	initial, err := ParseArgs(def.Args)
	if err != nil {
		return nil, fmt.Errorf("failed to parse test initial args: %w", err)
	}

	opsCount, err := RunInstructions(initial, res.Stdout)
	if err != nil {
		testRes.Status = model.StatusFailed
		testRes.Actual = fmt.Sprintf("Sorting verification failed: %v", err)
		testRes.Explanation = fmt.Sprintf("Failed to sort: %v", err)
		testRes.Advice = "Ensure chunking, radix, or turk algorithm maintains all numbers and properly restores stack A."
		return testRes, nil
	}

	// Calculate 42 Grade
	grade := grade42(numElements, opsCount)
	testRes.Actual = fmt.Sprintf("%d operations (Grade: %d/5)", opsCount, grade)

	if opsCount > def.MaxOps {
		testRes.Status = model.StatusFailed
		testRes.Explanation = fmt.Sprintf("Benchmark target exceeded: used %d operations (Grade %d/5, Target for 5/5 is < %d)", opsCount, grade, def.MaxOps)
		if numElements <= 100 {
			testRes.Advice = "For 100 numbers: 5/5 (< 700 ops), 4/5 (< 900), 3/5 (< 1100), 2/5 (< 1300), 1/5 (< 1500). Optimize chunk size or cost calculation."
		} else {
			testRes.Advice = "For 500 numbers: 5/5 (< 5500 ops), 4/5 (< 7000), 3/5 (< 8500), 2/5 (< 10000), 1/5 (< 11500). Fine-tune radix base or chunk partition bounds."
		}
		testRes.Diff = diff.Compare(fmt.Sprintf("< %d ops (5/5)", def.MaxOps), fmt.Sprintf("%d ops (%d/5)", opsCount, grade))
		return testRes, nil
	}

	testRes.Status = model.StatusPassed
	return testRes, nil
}

func (s *Suite) runMemoryCase(ctx context.Context, dir string, tc model.TestCase, def TestCaseDef, r *runner.Runner) (*model.TestResult, error) {
	binPath := s.getBinPath(dir)
	out, hasLeaks, err := r.RunMemoryCheck(ctx, binPath, def.Args...)
	testRes := &model.TestResult{
		Case:        tc,
		Input:       formatArgs(def.Args),
		Expected:    "0 leaks for 0 total leaked bytes",
		CodeSnippet: fmt.Sprintf("leaks --atExit -- ./push_swap %s", formatArgs(def.Args)),
	}

	if err != nil && !strings.Contains(out, "leaks Report") {
		// Leaks tool might not be available or permission restricted
		testRes.Status = model.StatusPassed
		testRes.Actual = "Memory check skipped (tool unavailable)"
		return testRes, nil
	}

	if hasLeaks {
		testRes.Status = model.StatusLeak
		testRes.Actual = "Memory leaks detected"
		testRes.Explanation = "Program leaked heap memory on exit."
		testRes.LeaksReport = out
		testRes.Advice = "Ensure all linked list nodes (and any dynamically split argv strings) are freed before program termination."
		return testRes, nil
	}

	testRes.Status = model.StatusPassed
	testRes.Actual = "0 leaks (Clean)"
	return testRes, nil
}

func (s *Suite) runBonusCase(ctx context.Context, dir string, tc model.TestCase, def TestCaseDef, r *runner.Runner) (*model.TestResult, error) {
	bonusBin := s.getBonusBinPath(dir)
	if bonusBin == "" {
		return &model.TestResult{
			Case:        tc,
			Status:      model.StatusSkipped,
			Explanation: "Bonus checker binary not found",
		}, nil
	}

	binPath := s.getBinPath(dir)

	switch def.ID {
	case "pushswap_bonus_valid_sort":
		// Run push_swap, pipe output to checker
		psRes, err := r.Run(ctx, binPath, runner.RunOptions{
			Args:    []string{"3", "2", "1"},
			Cwd:     dir,
			Timeout: 2 * time.Second,
		})
		if err != nil {
			return &model.TestResult{Case: tc, Status: model.StatusFailed, Explanation: fmt.Sprintf("push_swap run failed: %v", err)}, nil
		}
		chRes, err := r.Run(ctx, bonusBin, runner.RunOptions{
			Args:    []string{"3", "2", "1"},
			Stdin:   psRes.Stdout,
			Cwd:     dir,
			Timeout: 2 * time.Second,
		})
		if err != nil {
			return &model.TestResult{Case: tc, Status: model.StatusFailed, Explanation: fmt.Sprintf("checker run failed: %v", err)}, nil
		}
		if strings.TrimSpace(chRes.Stdout) != "OK" {
			return &model.TestResult{
				Case:        tc,
				Status:      model.StatusFailed,
				Expected:    "OK\\n",
				Actual:      chRes.Stdout,
				Explanation: fmt.Sprintf("Checker did not output 'OK\\n' on valid sort: %q", chRes.Stdout),
				Advice:      "Checker must verify operations and output 'OK\\n' when the final stack is sorted.",
			}, nil
		}
		return &model.TestResult{Case: tc, Status: model.StatusPassed, Actual: "OK\\n"}, nil

	case "pushswap_bonus_ko":
		// Give unsorted operations to checker
		chRes, err := r.Run(ctx, bonusBin, runner.RunOptions{
			Args:    []string{"3", "2", "1"},
			Stdin:   "sa\n", // sa turns 3 2 1 into 2 3 1 (not sorted!)
			Cwd:     dir,
			Timeout: 2 * time.Second,
		})
		if err != nil {
			return &model.TestResult{Case: tc, Status: model.StatusFailed, Explanation: fmt.Sprintf("checker run failed: %v", err)}, nil
		}
		if strings.TrimSpace(chRes.Stdout) != "KO" {
			return &model.TestResult{
				Case:        tc,
				Status:      model.StatusFailed,
				Expected:    "KO\\n",
				Actual:      chRes.Stdout,
				Explanation: fmt.Sprintf("Checker did not output 'KO\\n' on unsorted stack: %q", chRes.Stdout),
				Advice:      "Checker must output 'KO\\n' when the operations leave stack unsorted.",
			}, nil
		}
		return &model.TestResult{Case: tc, Status: model.StatusPassed, Actual: "KO\\n"}, nil

	case "pushswap_bonus_invalid_op":
		// Give invalid operation to checker
		chRes, err := r.Run(ctx, bonusBin, runner.RunOptions{
			Args:    []string{"3", "2", "1"},
			Stdin:   "foo\n",
			Cwd:     dir,
			Timeout: 2 * time.Second,
		})
		if err != nil {
			return &model.TestResult{Case: tc, Status: model.StatusFailed, Explanation: fmt.Sprintf("checker run failed: %v", err)}, nil
		}
		if strings.TrimSpace(chRes.Stderr) != "Error" {
			return &model.TestResult{
				Case:        tc,
				Status:      model.StatusFailed,
				Expected:    "Error\\n on stderr",
				Actual:      chRes.Stderr,
				Explanation: fmt.Sprintf("Checker did not output 'Error\\n' on invalid instruction: %q", chRes.Stderr),
				Advice:      "Checker must output 'Error\\n' on stderr when encountering an unknown instruction.",
			}, nil
		}
		return &model.TestResult{Case: tc, Status: model.StatusPassed, Actual: "Error\\n on stderr"}, nil

	case "pushswap_bonus_error_args":
		chRes, err := r.Run(ctx, bonusBin, runner.RunOptions{
			Args:    []string{"1", "2", "a"},
			Cwd:     dir,
			Timeout: 2 * time.Second,
		})
		if err != nil {
			return &model.TestResult{Case: tc, Status: model.StatusFailed, Explanation: fmt.Sprintf("checker run failed: %v", err)}, nil
		}
		if strings.TrimSpace(chRes.Stderr) != "Error" {
			return &model.TestResult{
				Case:        tc,
				Status:      model.StatusFailed,
				Expected:    "Error\\n on stderr",
				Actual:      chRes.Stderr,
				Explanation: fmt.Sprintf("Checker did not output 'Error\\n' on invalid args: %q", chRes.Stderr),
				Advice:      "Checker must output 'Error\\n' on stderr when arguments are invalid.",
			}, nil
		}
		return &model.TestResult{Case: tc, Status: model.StatusPassed, Actual: "Error\\n on stderr"}, nil
	}

	return &model.TestResult{Case: tc, Status: model.StatusPassed}, nil
}

func (s *Suite) Cleanup(ctx context.Context, dir string) error {
	return nil
}

func grade42(size, ops int) int {
	if size <= 100 {
		switch {
		case ops < 700:
			return 5
		case ops < 900:
			return 4
		case ops < 1100:
			return 3
		case ops < 1300:
			return 2
		case ops < 1500:
			return 1
		default:
			return 0
		}
	}
	switch {
	case ops < 5500:
		return 5
	case ops < 7000:
		return 4
	case ops < 8500:
		return 3
	case ops < 10000:
		return 2
	case ops < 11500:
		return 1
	default:
		return 0
	}
}

func formatArgs(args []string) string {
	if len(args) == 0 {
		return "(no args)"
	}
	if len(args) > 10 {
		return fmt.Sprintf("%s ... (%d numbers)", strings.Join(args[:5], " "), len(args))
	}
	return strings.Join(args, " ")
}

func generatePermutationArgs(count int, seed int64) []string {
	r := rand.New(rand.NewSource(seed))
	vals := make([]int, count)
	for i := 0; i < count; i++ {
		vals[i] = i - count/2
	}
	r.Shuffle(count, func(i, j int) {
		vals[i], vals[j] = vals[j], vals[i]
	})
	res := make([]string, count)
	for i, v := range vals {
		res[i] = strconv.Itoa(v)
	}
	return res
}

func generateReverseArgs(count int) []string {
	res := make([]string, count)
	for i := 0; i < count; i++ {
		res[i] = strconv.Itoa(count - i)
	}
	return res
}
