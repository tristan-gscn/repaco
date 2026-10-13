package tui

import (
	"strings"
	"testing"

	"github.com/tristan-gscn/repaco/internal/model"
)

func TestRenderBottomBarAllPassed(t *testing.T) {
	m := Model{
		results: []model.TestResult{
			{Status: model.StatusPassed},
			{Status: model.StatusPassed},
		},
		stats: model.SummaryStats{
			Total:  2,
			Passed: 2,
		},
		totalTests: 2,
		running:    false,
		done:       true,
	}

	bar := m.renderBottomBar(80)
	// Must display 100% and (2/2)
	if !strings.Contains(bar, "100%") {
		t.Errorf("expected 100%% in bottom bar, got: %s", bar)
	}
	if !strings.Contains(bar, "(2/2)") {
		t.Errorf("expected (2/2) in bottom bar, got: %s", bar)
	}
}

func TestRenderBottomBarAllFailed(t *testing.T) {
	m := Model{
		results: []model.TestResult{
			{Status: model.StatusFailed},
			{Status: model.StatusFailed},
		},
		stats: model.SummaryStats{
			Total:  2,
			Failed: 2,
		},
		totalTests: 2,
		running:    false,
		done:       true,
	}

	bar := m.renderBottomBar(80)
	// Must display 0% and (0/2)
	if !strings.Contains(bar, "0%") {
		t.Errorf("expected 0%% in bottom bar, got: %s", bar)
	}
	if !strings.Contains(bar, "(0/2)") {
		t.Errorf("expected (0/2) in bottom bar, got: %s", bar)
	}
}

func TestRenderBottomBarMixedHalfPassed(t *testing.T) {
	m := Model{
		results: []model.TestResult{
			{Status: model.StatusPassed},
			{Status: model.StatusFailed},
		},
		stats: model.SummaryStats{
			Total:  2,
			Passed: 1,
			Failed: 1,
		},
		totalTests: 2,
		running:    false,
		done:       true,
	}

	bar := m.renderBottomBar(80)
	// Must display 50% and (1/2)
	if !strings.Contains(bar, "50%") {
		t.Errorf("expected 50%% in bottom bar, got: %s", bar)
	}
	if !strings.Contains(bar, "(1/2)") {
		t.Errorf("expected (1/2) in bottom bar, got: %s", bar)
	}
}
