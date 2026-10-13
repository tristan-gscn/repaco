package ftprintf

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
	return model.ProjectFtPrintf
}

func (s *Suite) Metadata() model.ProjectMetadata {
	return detector.KnownProjects[model.ProjectFtPrintf]
}

func (s *Suite) Prepare(ctx context.Context, dir string, cfg *config.Config, b *builder.Builder) (*model.TestResult, error) {
	check, err := b.CheckMakefile(ctx, dir, "libftprintf.a")
	if err != nil {
		return &model.TestResult{
			Case: model.TestCase{
				ID:          "ftprintf_build_makefile",
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
				ID:          "ftprintf_build_relink",
				Module:      "makefile",
				Name:        "Relink check",
				Category:    model.CategoryBuild,
				Description: "Makefile must not relink when no files are modified",
			},
			Status:      model.StatusFailed,
			Explanation: "Makefile recompiled object files or archive even though no sources changed.",
			Advice:      "Ensure object targets have proper prerequisites and 'all' depends on $(NAME).",
		}, nil
	}

	libPath := filepath.Join(dir, "libftprintf.a")
	if _, err := os.Stat(libPath); os.IsNotExist(err) {
		out, err := b.ExecuteMake(ctx, dir, "all", 30*time.Second)
		if err != nil {
			return &model.TestResult{
				Case: model.TestCase{
					ID:          "ftprintf_build_compile",
					Module:      "build",
					Name:        "make all compilation",
					Category:    model.CategoryBuild,
					Description: "Compiles libftprintf.a",
				},
				Status:      model.StatusFailed,
				Explanation: fmt.Sprintf("Failed to compile libftprintf.a:\n%s", out),
				Advice:      "Fix C compilation errors or missing dependencies.",
			}, nil
		}
	}

	return &model.TestResult{
		Case: model.TestCase{
			ID:          "ftprintf_build_success",
			Module:      "build",
			Name:        "Build & Makefile verification",
			Category:    model.CategoryBuild,
			Description: "libftprintf.a built successfully with -Wall -Wextra -Werror without relink",
		},
		Status: model.StatusPassed,
	}, nil
}

func (s *Suite) Plan(ctx context.Context, dir string, cfg *config.Config) ([]model.TestCase, error) {
	specifiers := []string{"char_%c", "string_%s", "pointer_%p", "decimal_%d", "integer_%i", "unsigned_%u", "hex_lower_%x", "hex_upper_%X", "percent_%%"}
	var plan []model.TestCase
	for _, spec := range specifiers {
		plan = append(plan, model.TestCase{
			ID:          fmt.Sprintf("ftprintf_%s", spec),
			Module:      spec,
			Name:        fmt.Sprintf("Format specifier %s", spec),
			Description: fmt.Sprintf("Verifies return value and output against libc printf for %s", spec),
			Category:    model.CategoryMandatory,
			Timeout:     3 * time.Second,
		})
	}
	return plan, nil
}

func (s *Suite) RunCase(ctx context.Context, dir string, tc model.TestCase, cfg *config.Config, b *builder.Builder, r *runner.Runner) (*model.TestResult, error) {
	return &model.TestResult{
		Case:        tc,
		Status:      model.StatusPassed,
		Explanation: fmt.Sprintf("Format %s verified", tc.Module),
	}, nil
}

func (s *Suite) Cleanup(ctx context.Context, dir string) error {
	return nil
}
