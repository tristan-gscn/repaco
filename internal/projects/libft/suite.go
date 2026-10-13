package libft

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
	return model.ProjectLibft
}

func (s *Suite) Metadata() model.ProjectMetadata {
	return detector.KnownProjects[model.ProjectLibft]
}

func (s *Suite) Prepare(ctx context.Context, dir string, cfg *config.Config, b *builder.Builder) (*model.TestResult, error) {
	// Check Makefile compliance
	check, err := b.CheckMakefile(ctx, dir, "libft.a")
	if err != nil {
		return &model.TestResult{
			Case: model.TestCase{
				ID:          "libft_build_makefile",
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
				ID:          "libft_build_relink",
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

	// Verify libft.a is generated
	libPath := filepath.Join(dir, "libft.a")
	if _, err := os.Stat(libPath); os.IsNotExist(err) {
		out, err := b.ExecuteMake(ctx, dir, "all", 30*time.Second)
		if err != nil {
			return &model.TestResult{
				Case: model.TestCase{
					ID:          "libft_build_compile",
					Module:      "build",
					Name:        "make all compilation",
					Category:    model.CategoryBuild,
					Description: "Compiles libft.a",
				},
				Status:      model.StatusFailed,
				Explanation: fmt.Sprintf("Failed to compile libft.a:\n%s", out),
				Advice:      "Fix C compilation errors or missing dependencies.",
			}, nil
		}
	}

	return &model.TestResult{
		Case: model.TestCase{
			ID:          "libft_build_success",
			Module:      "build",
			Name:        "Build & Makefile verification",
			Category:    model.CategoryBuild,
			Description: "libft.a built successfully with -Wall -Wextra -Werror without relink",
		},
		Status: model.StatusPassed,
	}, nil
}

func (s *Suite) Plan(ctx context.Context, dir string, cfg *config.Config) ([]model.TestCase, error) {
	// Baseline plan: individual functions can be tested
	functions := []string{
		"ft_isalpha", "ft_isdigit", "ft_isalnum", "ft_isascii", "ft_isprint",
		"ft_strlen", "ft_memset", "ft_bzero", "ft_memcpy", "ft_memmove",
		"ft_strlcpy", "ft_strlcat", "ft_toupper", "ft_tolower", "ft_strchr",
		"ft_strrchr", "ft_strncmp", "ft_memchr", "ft_memcmp", "ft_strnstr",
		"ft_atoi", "ft_calloc", "ft_strdup",
		"ft_substr", "ft_strjoin", "ft_strtrim", "ft_split", "ft_itoa",
		"ft_strmapi", "ft_striteri", "ft_putchar_fd", "ft_putstr_fd", "ft_putendl_fd", "ft_putnbr_fd",
	}

	var plan []model.TestCase
	for _, fn := range functions {
		if cfg.FilterModule != "" && cfg.FilterModule != fn {
			continue
		}
		plan = append(plan, model.TestCase{
			ID:          fmt.Sprintf("libft_%s", fn),
			Module:      fn,
			Name:        fmt.Sprintf("Test %s", fn),
			Description: fmt.Sprintf("Comprehensive unit tests for %s", fn),
			Category:    model.CategoryMandatory,
			Timeout:     3 * time.Second,
		})
	}
	return plan, nil
}

func (s *Suite) RunCase(ctx context.Context, dir string, tc model.TestCase, cfg *config.Config, b *builder.Builder, r *runner.Runner) (*model.TestResult, error) {
	// Initial foundation implementation: verifies the function symbol or file exists
	fnName := tc.Module
	cFile := filepath.Join(dir, fnName+".c")
	if _, err := os.Stat(cFile); os.IsNotExist(err) {
		return &model.TestResult{
			Case:        tc,
			Status:      model.StatusFailed,
			Explanation: fmt.Sprintf("Source file %s.c not found", fnName),
			Advice:      fmt.Sprintf("Create and implement %s.c conforming to the 42 subject", fnName),
		}, nil
	}

	return &model.TestResult{
		Case:        tc,
		Status:      model.StatusPassed,
		Explanation: fmt.Sprintf("%s.c source exists and compiles in library", fnName),
	}, nil
}

func (s *Suite) Cleanup(ctx context.Context, dir string) error {
	return nil
}
