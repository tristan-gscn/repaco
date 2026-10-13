package pushswap

import (
	"context"
	"testing"

	"github.com/tristan-gscn/repaco/internal/config"
	"github.com/tristan-gscn/repaco/internal/model"
)

func TestSuitePlanning(t *testing.T) {
	s := &Suite{}
	if s.ID() != model.ProjectPushSwap {
		t.Fatalf("unexpected ID: %s", s.ID())
	}

	cfg := &config.Config{}
	plan, err := s.Plan(context.Background(), ".", cfg)
	if err != nil {
		t.Fatalf("Plan failed: %v", err)
	}

	if len(plan) == 0 {
		t.Fatalf("expected non-empty test plan")
	}

	// Verify categories present
	hasErrors := false
	hasIdentity := false
	hasSmall := false
	hasBench := false
	hasMem := false

	for _, tc := range plan {
		switch tc.File {
		case "push_swap_errors.c":
			hasErrors = true
		case "push_swap_identity.c":
			hasIdentity = true
		case "push_swap_small.c":
			hasSmall = true
		case "push_swap_benchmarks.c":
			hasBench = true
		case "push_swap_memory.c":
			hasMem = true
		}
	}

	if !hasErrors || !hasIdentity || !hasSmall || !hasBench || !hasMem {
		t.Fatalf("missing expected categories in plan: errors=%v identity=%v small=%v bench=%v mem=%v",
			hasErrors, hasIdentity, hasSmall, hasBench, hasMem)
	}
}

func TestGrade42(t *testing.T) {
	if grade := grade42(100, 690); grade != 5 {
		t.Fatalf("expected 5 for 690 ops on 100 nums, got %d", grade)
	}
	if grade := grade42(100, 750); grade != 4 {
		t.Fatalf("expected 4 for 750 ops on 100 nums, got %d", grade)
	}
	if grade := grade42(500, 5200); grade != 5 {
		t.Fatalf("expected 5 for 5200 ops on 500 nums, got %d", grade)
	}
	if grade := grade42(500, 11000); grade != 1 {
		t.Fatalf("expected 1 for 11000 ops on 500 nums, got %d", grade)
	}
}
