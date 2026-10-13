package projects

import (
	"context"
	"fmt"

	"github.com/tristan-gscn/repaco/internal/builder"
	"github.com/tristan-gscn/repaco/internal/config"
	"github.com/tristan-gscn/repaco/internal/model"
	"github.com/tristan-gscn/repaco/internal/runner"
)

// ProjectSuite defines the contract that each 42 project test suite must satisfy.
type ProjectSuite interface {
	ID() model.ProjectID
	Metadata() model.ProjectMetadata
	// Prepare runs preparatory builds or checks before testing starts.
	Prepare(ctx context.Context, dir string, cfg *config.Config, b *builder.Builder) (*model.TestResult, error)
	// Plan returns all test cases that will be executed for this project.
	Plan(ctx context.Context, dir string, cfg *config.Config) ([]model.TestCase, error)
	// RunCase executes a single test case and returns detailed diagnostics.
	RunCase(ctx context.Context, dir string, tc model.TestCase, cfg *config.Config, b *builder.Builder, r *runner.Runner) (*model.TestResult, error)
	// Cleanup removes temporary binaries or scratch files generated during testing.
	Cleanup(ctx context.Context, dir string) error
}

// Registry stores all registered project suites.
type Registry struct {
	suites map[model.ProjectID]ProjectSuite
}

// Global registry instance.
var globalRegistry = &Registry{
	suites: make(map[model.ProjectID]ProjectSuite),
}

// Register adds a suite to the global registry.
func Register(suite ProjectSuite) {
	globalRegistry.suites[suite.ID()] = suite
}

// Get retrieves a registered suite by ProjectID.
func Get(id model.ProjectID) (ProjectSuite, error) {
	s, ok := globalRegistry.suites[id]
	if !ok {
		return nil, fmt.Errorf("no test suite registered for project ID: %s", id)
	}
	return s, nil
}

// All returns all registered suites.
func All() []ProjectSuite {
	var list []ProjectSuite
	for _, s := range globalRegistry.suites {
		list = append(list, s)
	}
	return list
}
