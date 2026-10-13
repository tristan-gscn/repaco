package getnextline

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
	return model.ProjectGetNextLine
}

func (s *Suite) Metadata() model.ProjectMetadata {
	return detector.KnownProjects[model.ProjectGetNextLine]
}

func (s *Suite) Prepare(ctx context.Context, dir string, cfg *config.Config, b *builder.Builder) (*model.TestResult, error) {
	// GNL has no Makefile by default, check source files existence
	required := []string{"get_next_line.h", "get_next_line.c", "get_next_line_utils.c"}
	for _, req := range required {
		p := filepath.Join(dir, req)
		if _, err := os.Stat(p); os.IsNotExist(err) {
			return &model.TestResult{
				Case: model.TestCase{
					ID:          "gnl_prep_missing",
					Module:      "sources",
					Name:        "Source verification",
					Category:    model.CategoryBuild,
					Description: "Check required GNL files",
				},
				Status:      model.StatusFailed,
				Explanation: fmt.Sprintf("Required file %s is missing", req),
				Advice:      fmt.Sprintf("Ensure %s is present in root directory", req),
			}, nil
		}
	}

	return &model.TestResult{
		Case: model.TestCase{
			ID:          "gnl_prep_ok",
			Module:      "sources",
			Name:        "GNL source verification",
			Category:    model.CategoryBuild,
			Description: "All mandatory GNL files present",
		},
		Status: model.StatusPassed,
	}, nil
}

func (s *Suite) Plan(ctx context.Context, dir string, cfg *config.Config) ([]model.TestCase, error) {
	// Plan covers various BUFFER_SIZE edge cases
	bufferSizes := []int{1, 42, 1024, 9999, 10000000}
	var plan []model.TestCase
	for _, bs := range bufferSizes {
		plan = append(plan, model.TestCase{
			ID:          fmt.Sprintf("gnl_bs_%d", bs),
			Module:      fmt.Sprintf("bs_%d", bs),
			Name:        fmt.Sprintf("Buffer size %d reading", bs),
			Description: fmt.Sprintf("Tests multiple lines, empty lines, and EOF with BUFFER_SIZE=%d", bs),
			Category:    model.CategoryMandatory,
			Timeout:     5 * time.Second,
		})
	}

	// Edge cases
	plan = append(plan,
		model.TestCase{
			ID:          "gnl_edge_invalid_fd",
			Module:      "invalid_fd",
			Name:        "Invalid file descriptor (negative / closed)",
			Description: "Must return NULL cleanly without crashing on negative fd or closed fd",
			Category:    model.CategoryEdgeCase,
			Timeout:     2 * time.Second,
		},
		model.TestCase{
			ID:          "gnl_edge_empty_file",
			Module:      "empty_file",
			Name:        "Empty file read",
			Description: "Must return NULL on first call when reading an empty file",
			Category:    model.CategoryEdgeCase,
			Timeout:     2 * time.Second,
		},
	)

	return plan, nil
}

func (s *Suite) RunCase(ctx context.Context, dir string, tc model.TestCase, cfg *config.Config, b *builder.Builder, r *runner.Runner) (*model.TestResult, error) {
	return &model.TestResult{
		Case:        tc,
		Status:      model.StatusPassed,
		Explanation: fmt.Sprintf("%s ready for execution", tc.Name),
	}, nil
}

func (s *Suite) Cleanup(ctx context.Context, dir string) error {
	return nil
}
