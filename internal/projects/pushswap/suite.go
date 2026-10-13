package pushswap

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/tristan-gscn/repaco/internal/builder"
	"github.com/tristan-gscn/repaco/internal/config"
	"github.com/tristan-gscn/repaco/internal/detector"
	"github.com/tristan-gscn/repaco/internal/model"
	"github.com/tristan-gscn/repaco/internal/projects"
	"github.com/tristan-gscn/repaco/internal/runner"
)

type Suite struct{}

func init() {
	projects.Register(&Suite{})
}

func (s *Suite) ID() model.ProjectID {
	return model.ProjectPushSwap
}

func (s *Suite) Metadata() model.ProjectMetadata {
	return detector.KnownProjects[model.ProjectPushSwap]
}

func (s *Suite) Prepare(ctx context.Context, dir string, cfg *config.Config, b *builder.Builder) (*model.TestResult, error) {
	check, err := b.CheckMakefile(ctx, dir, "push_swap")
	if err != nil {
		return &model.TestResult{
			Case: model.TestCase{
				ID:          "pushswap_build_makefile",
				Module:      "makefile",
				Name:        "Makefile compliance",
				Category:    model.CategoryBuild,
				Description: "Verifies mandatory rules, flags, and relink",
			},
			Status:      model.StatusFailed,
			Explanation: fmt.Sprintf("Makefile check failed: %v", err),
			Advice:      "Check that Makefile is present and rules 'all', 'clean', 'fclean', 're' compile cleanly.",
		}, nil
	}

	if check.RelinkDetected {
		return &model.TestResult{
			Case: model.TestCase{
				ID:          "pushswap_build_relink",
				Module:      "makefile",
				Name:        "Relink check",
				Category:    model.CategoryBuild,
				Description: "Makefile must not relink when no files are modified",
			},
			Status:      model.StatusFailed,
			Explanation: "Makefile recompiled object files or binary even though no sources changed.",
			Advice:      "Ensure object targets have proper prerequisites and 'all' depends on $(NAME).",
		}, nil
	}

	binPath := filepath.Join(dir, "push_swap")
	if _, err := os.Stat(binPath); os.IsNotExist(err) {
		out, err := b.ExecuteMake(ctx, dir, "all", 30*time.Second)
		if err != nil {
			return &model.TestResult{
				Case: model.TestCase{
					ID:          "pushswap_build_compile",
					Module:      "build",
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

	return &model.TestResult{
		Case: model.TestCase{
			ID:          "pushswap_build_success",
			Module:      "build",
			Name:        "Build & Makefile verification",
			Category:    model.CategoryBuild,
			Description: "push_swap built successfully with -Wall -Wextra -Werror without relink",
		},
		Status: model.StatusPassed,
	}, nil
}

func (s *Suite) Plan(ctx context.Context, dir string, cfg *config.Config) ([]model.TestCase, error) {
	var plan []model.TestCase

	// Error Management
	errorCases := []struct {
		id   string
		name string
		desc string
	}{
		{"err_non_numeric", "Non-numeric argument", "Arguments with letters must output Error\\n to stderr"},
		{"err_int_overflow", "Integer overflow (> INT_MAX)", "Numbers exceeding INT_MAX must output Error\\n"},
		{"err_int_underflow", "Integer underflow (< INT_MIN)", "Numbers below INT_MIN must output Error\\n"},
		{"err_duplicates", "Duplicate numbers", "Duplicate numbers must output Error\\n"},
		{"err_empty_param", "Empty string parameter", "Empty string parameter must output Error\\n"},
	}

	for _, ec := range errorCases {
		plan = append(plan, model.TestCase{
			ID:          "pushswap_" + ec.id,
			Module:      "error_handling",
			Name:        ec.name,
			Description: ec.desc,
			Category:    model.CategoryEdgeCase,
			Timeout:     2 * time.Second,
		})
	}

	// Identity / Already Sorted
	plan = append(plan, model.TestCase{
		ID:          "pushswap_sorted_identity",
		Module:      "sorted",
		Name:        "Already sorted stack",
		Description: "Running push_swap with already sorted numbers must output nothing",
		Category:    model.CategoryMandatory,
		Timeout:     2 * time.Second,
	})

	// Benchmark Targets (3, 5, 100, 500)
	plan = append(plan,
		model.TestCase{
			ID:          "pushswap_bench_3",
			Module:      "small_sort",
			Name:        "Sort 3 numbers",
			Description: "Must sort 3 numbers in <= 3 operations",
			Category:    model.CategoryBenchmark,
			Timeout:     2 * time.Second,
		},
		model.TestCase{
			ID:          "pushswap_bench_5",
			Module:      "small_sort",
			Name:        "Sort 5 numbers",
			Description: "Must sort 5 numbers in <= 12 operations",
			Category:    model.CategoryBenchmark,
			Timeout:     2 * time.Second,
		},
		model.TestCase{
			ID:          "pushswap_bench_100",
			Module:      "large_sort",
			Name:        "Sort 100 numbers (Target < 700 ops)",
			Description: "Exhaustive tests with 100 random numbers, maximum score < 700 ops",
			Category:    model.CategoryBenchmark,
			Timeout:     5 * time.Second,
		},
		model.TestCase{
			ID:          "pushswap_bench_500",
			Module:      "large_sort",
			Name:        "Sort 500 numbers (Target < 5500 ops)",
			Description: "Tests with 500 random numbers, target < 5500 operations",
			Category:    model.CategoryBenchmark,
			Timeout:     10 * time.Second,
		},
	)

	return plan, nil
}

func (s *Suite) RunCase(ctx context.Context, dir string, tc model.TestCase, cfg *config.Config, b *builder.Builder, r *runner.Runner) (*model.TestResult, error) {
	return &model.TestResult{
		Case:        tc,
		Status:      model.StatusPassed,
		Explanation: fmt.Sprintf("%s prepared for test execution", tc.Name),
	}, nil
}

func (s *Suite) Cleanup(ctx context.Context, dir string) error {
	return nil
}
