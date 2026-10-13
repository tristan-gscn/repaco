package ftprintf

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/tristan-gscn/repaco/internal/builder"
	"github.com/tristan-gscn/repaco/internal/config"
	"github.com/tristan-gscn/repaco/internal/model"
	"github.com/tristan-gscn/repaco/internal/runner"
)

func TestFtPrintfFullBatteryOnUserCode(t *testing.T) {
	ftprintfPath, err := filepath.Abs("../../../../ft-printf")
	if err != nil {
		t.Fatalf("Failed to resolve ft-printf path: %v", err)
	}
	if _, err := os.Stat(ftprintfPath); os.IsNotExist(err) {
		t.Skipf("Local ft-printf project not found at %s, skipping integration test", ftprintfPath)
	}

	suite := &Suite{}
	cfg := config.NewDefaultConfig()
	cfg.TargetDir = ftprintfPath
	cfg.DefaultTimeout = 5 * time.Second

	b := builder.New()
	r := runner.New(cfg.DefaultTimeout)
	ctx := context.Background()

	// 1. Prepare
	prep, err := suite.Prepare(ctx, ftprintfPath, cfg, b)
	if err != nil {
		t.Fatalf("Prepare error: %v", err)
	}
	if prep.Status != model.StatusPassed {
		t.Fatalf("Prepare failed: %s (%s)", prep.Explanation, prep.Advice)
	}
	t.Logf("Prepare status: %s", prep.Status)

	// 2. Plan
	cases, err := suite.Plan(ctx, ftprintfPath, cfg)
	if err != nil {
		t.Fatalf("Plan error: %v", err)
	}
	t.Logf("Planned %d tests", len(cases))

	// 3. Run all tests
	passed := 0
	failed := 0
	for _, tc := range cases {
		res, err := suite.RunCase(ctx, ftprintfPath, tc, cfg, b, r)
		if err != nil {
			t.Errorf("Error running test %s: %v", tc.ID, err)
			failed++
			continue
		}
		if res.Status == model.StatusPassed {
			passed++
		} else {
			failed++
			t.Logf("FAIL [%s] %s: %s (Advice: %s)\nInput: %s\nExpected: %s\nActual: %s",
				tc.ID, tc.Name, res.Explanation, res.Advice, res.Input, res.Expected, res.Actual)
		}
	}

	t.Logf("ft_printf Battery Results: %d Passed, %d Failed (Total %d)", passed, failed, len(cases))
	_ = suite.Cleanup(ctx, ftprintfPath)
}
