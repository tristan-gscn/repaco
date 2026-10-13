package engine

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"github.com/tristan-gscn/repaco/internal/builder"
	"github.com/tristan-gscn/repaco/internal/config"
	"github.com/tristan-gscn/repaco/internal/detector"
	"github.com/tristan-gscn/repaco/internal/model"
	"github.com/tristan-gscn/repaco/internal/projects"
	"github.com/tristan-gscn/repaco/internal/runner"
)

// Engine coordinates project detection, building, and test execution.
type Engine struct {
	cfg      *config.Config
	detector *detector.Detector
	builder  *builder.Builder
	runner   *runner.Runner
}

// New creates an Engine instance.
func New(cfg *config.Config) *Engine {
	return &Engine{
		cfg:      cfg,
		detector: detector.New(),
		builder:  builder.New(),
		runner:   runner.New(cfg.DefaultTimeout),
	}
}

// Run executes the complete testing pipeline and streams events to eventsChan.
func (e *Engine) Run(ctx context.Context, eventsChan chan<- model.EngineEvent) (*model.SuiteResult, error) {
	defer close(eventsChan)

	if absDir, err := filepath.Abs(e.cfg.TargetDir); err == nil {
		e.cfg.TargetDir = absDir
	}

	startTime := time.Now()

	// 1. Detection
	eventsChan <- model.EngineEvent{
		Type:    model.EventDetectStarted,
		Message: fmt.Sprintf("Detecting 42 project in %s...", e.cfg.TargetDir),
	}

	det, err := e.detector.Detect(e.cfg.TargetDir)
	if err != nil {
		eventsChan <- model.EngineEvent{
			Type:  model.EventFatalError,
			Error: err,
		}
		return nil, err
	}

	if e.cfg.ProjectOverride != "" {
		det.Project.ID = model.ProjectID(e.cfg.ProjectOverride)
		det.Project.Name = e.cfg.ProjectOverride
	}

	eventsChan <- model.EngineEvent{
		Type:      model.EventDetectDone,
		Detection: det,
		Message:   fmt.Sprintf("Detected %s (confidence: %.0f%%)", det.Project.Name, det.Confidence*100),
	}

	if det.Project.ID == model.ProjectUnknown {
		// Look in nearby directories first
		nearby, errAround := e.detector.DetectAround(e.cfg.TargetDir)
		if errAround == nil && len(nearby) > 0 {
			eventsChan <- model.EngineEvent{
				Type:           model.EventProjectsNearbyFound,
				ProjectsNearby: nearby,
				Message:        fmt.Sprintf("Found %d 42 projects nearby", len(nearby)),
			}
			return nil, nil
		}

		err := fmt.Errorf("no supported 42 project recognized in %s", e.cfg.TargetDir)
		eventsChan <- model.EngineEvent{
			Type:  model.EventFatalError,
			Error: err,
		}
		return nil, err
	}

	// 2. Fetch Suite
	suite, err := projects.Get(det.Project.ID)
	if err != nil {
		eventsChan <- model.EngineEvent{
			Type:  model.EventFatalError,
			Error: err,
		}
		return nil, err
	}

	// 3. Build & Prepare
	eventsChan <- model.EngineEvent{
		Type:    model.EventBuildStarted,
		Message: "Validating Makefile and building prerequisites...",
	}

	prepResult, err := suite.Prepare(ctx, e.cfg.TargetDir, e.cfg, e.builder)
	if err != nil {
		eventsChan <- model.EngineEvent{
			Type:  model.EventFatalError,
			Error: err,
		}
		return nil, err
	}

	suiteResult := &model.SuiteResult{
		ProjectID: det.Project.ID,
	}

	if prepResult != nil {
		suiteResult.Results = append(suiteResult.Results, *prepResult)
		eventsChan <- model.EngineEvent{
			Type:       model.EventBuildDone,
			TestResult: prepResult,
			Message:    "Build check finished",
		}
		if prepResult.Status == model.StatusFailed && e.cfg.FailFast {
			suiteResult.Duration = time.Since(startTime)
			suiteResult.UpdateStats()
			return suiteResult, nil
		}
	}

	// 4. Test Planning
	cases, err := suite.Plan(ctx, e.cfg.TargetDir, e.cfg)
	if err != nil {
		eventsChan <- model.EngineEvent{
			Type:  model.EventFatalError,
			Error: err,
		}
		return nil, err
	}

	totalCases := len(cases)
	for i, tc := range cases {
		select {
		case <-ctx.Done():
			return suiteResult, ctx.Err()
		default:
		}

		eventsChan <- model.EngineEvent{
			Type:    model.EventTestStarted,
			Current: i + 1,
			Total:   totalCases,
			Message: fmt.Sprintf("[%d/%d] Running %s (%s)", i+1, totalCases, tc.Name, tc.Category),
		}

		caseStart := time.Now()
		res, err := suite.RunCase(ctx, e.cfg.TargetDir, tc, e.cfg, e.builder, e.runner)
		if err != nil {
			res = &model.TestResult{
				Case:        tc,
				Status:      model.StatusFailed,
				Explanation: fmt.Sprintf("Internal test error: %v", err),
			}
		}
		if res.Duration == 0 {
			res.Duration = time.Since(caseStart)
		}

		suiteResult.Results = append(suiteResult.Results, *res)

		eventsChan <- model.EngineEvent{
			Type:       model.EventTestCompleted,
			TestResult: res,
			Current:    i + 1,
			Total:      totalCases,
		}

		if (res.Status == model.StatusFailed || res.Status == model.StatusCrash) && e.cfg.FailFast {
			break
		}
	}

	_ = suite.Cleanup(ctx, e.cfg.TargetDir)

	suiteResult.Duration = time.Since(startTime)
	suiteResult.UpdateStats()

	eventsChan <- model.EngineEvent{
		Type:    model.EventSuiteDone,
		Message: fmt.Sprintf("Finished %d tests in %v", len(suiteResult.Results), suiteResult.Duration),
	}

	return suiteResult, nil
}

// RunWithDir executes testing for a specific project directory.
func (e *Engine) RunWithDir(ctx context.Context, dir string, eventsChan chan<- model.EngineEvent) (*model.SuiteResult, error) {
	e.cfg.TargetDir = dir
	return e.Run(ctx, eventsChan)
}
