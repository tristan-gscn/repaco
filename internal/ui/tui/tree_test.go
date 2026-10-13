package tui

import (
	"testing"

	"github.com/tristan-gscn/repaco/internal/model"
)

func TestTreeRowsGroupingAndCollapsing(t *testing.T) {
	m := Model{
		results: []model.TestResult{
			{
				Case: model.TestCase{
					ID:       "case_1",
					Module:   "ft_split",
					Name:     "Standard split",
					Category: model.CategoryMandatory,
				},
				Status: model.StatusPassed,
			},
			{
				Case: model.TestCase{
					ID:       "case_2",
					Module:   "ft_split",
					Name:     "Consecutive delimiters",
					Category: model.CategoryMandatory,
				},
				Status: model.StatusFailed,
			},
			{
				Case: model.TestCase{
					ID:       "case_3",
					Module:   "ft_isalpha",
					Name:     "Printable uppercase ASCII",
					Category: model.CategoryMandatory,
				},
				Status: model.StatusPassed,
			},
		},
		expandedGroups: make(map[string]bool),
	}

	// 1. By default: passed tests are collapsed (ft_isalpha.c), failed tests are expanded (ft_split.c)
	rows := m.buildTreeRows()
	// Should have:
	// row 0: Group "ft_split.c" (1/2 passed, GroupStatus: FAIL, Expanded: true)
	// row 1: Child "Standard split"
	// row 2: Child "Consecutive delimiters"
	// row 3: Group "ft_isalpha.c" (1/1 passed, GroupStatus: PASS, Expanded: false)
	if len(rows) != 4 {
		t.Fatalf("expected 4 visible tree rows by default, got %d", len(rows))
	}

	if !rows[0].IsGroup || rows[0].File != "ft_split.c" || rows[0].PassedCount != 1 || rows[0].TotalCount != 2 || rows[0].GroupStatus != model.StatusFailed || !rows[0].Expanded {
		t.Errorf("unexpected row 0: %+v", rows[0])
	}
	if rows[1].IsGroup || rows[1].Result.Case.Name != "Standard split" {
		t.Errorf("unexpected row 1: %+v", rows[1])
	}
	if rows[2].IsGroup || rows[2].Result.Case.Name != "Consecutive delimiters" {
		t.Errorf("unexpected row 2: %+v", rows[2])
	}
	if !rows[3].IsGroup || rows[3].File != "ft_isalpha.c" || rows[3].PassedCount != 1 || rows[3].TotalCount != 1 || rows[3].GroupStatus != model.StatusPassed || rows[3].Expanded {
		t.Errorf("unexpected row 3 (should be collapsed): %+v", rows[3])
	}

	// 2. Expand "ft_isalpha.c" manually
	m.expandedGroups["ft_isalpha.c"] = true
	expandedRows := m.buildTreeRows()
	if len(expandedRows) != 5 {
		t.Fatalf("expected 5 visible tree rows after expanding ft_isalpha.c, got %d", len(expandedRows))
	}
	if !expandedRows[3].Expanded {
		t.Errorf("expected ft_isalpha.c to be expanded")
	}
	if expandedRows[4].IsGroup || expandedRows[4].Result.Case.Name != "Printable uppercase ASCII" {
		t.Errorf("unexpected row 4: %+v", expandedRows[4])
	}

	// 3. Collapse "ft_split.c" manually
	m.expandedGroups["ft_split.c"] = false
	collapsedRows := m.buildTreeRows()
	// Should have:
	// row 0: Group "ft_split.c" (Expanded: false)
	// row 1: Group "ft_isalpha.c" (Expanded: true)
	// row 2: Child "Printable uppercase ASCII"
	if len(collapsedRows) != 3 {
		t.Fatalf("expected 3 visible tree rows after collapsing ft_split.c, got %d", len(collapsedRows))
	}
	if collapsedRows[0].Expanded {
		t.Errorf("expected ft_split.c to be collapsed")
	}
	if !collapsedRows[1].Expanded {
		t.Errorf("expected ft_isalpha.c to remain expanded")
	}
}
