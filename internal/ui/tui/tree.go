package tui

import (
	"github.com/tristan-gscn/repaco/internal/model"
)

// TreeRow represents a flattened navigable line in the hierarchical test tree.
type TreeRow struct {
	IsGroup     bool
	File        string
	Result      model.TestResult   // valid for leaf sub-test
	SubResults  []model.TestResult // all tests belonging to this file group
	Expanded    bool
	PassedCount int
	TotalCount  int
	GroupStatus model.Status
}

// buildTreeRows constructs the flattened list of visible tree rows (parent files and their sub-tests).
func (m Model) buildTreeRows() []TreeRow {
	if len(m.results) == 0 {
		return nil
	}

	var fileOrder []string
	grouped := make(map[string][]model.TestResult)

	for _, r := range m.results {
		fn := r.Case.FileName()
		if _, exists := grouped[fn]; !exists {
			fileOrder = append(fileOrder, fn)
		}
		grouped[fn] = append(grouped[fn], r)
	}

	var rows []TreeRow
	for _, fn := range fileOrder {
		tests := grouped[fn]
		passed := 0
		groupStatus := model.StatusPassed

		for _, t := range tests {
			if t.Status == model.StatusPassed {
				passed++
			} else if t.Status == model.StatusCrash {
				groupStatus = model.StatusCrash
			} else if t.Status == model.StatusLeak && groupStatus != model.StatusCrash {
				groupStatus = model.StatusLeak
			} else if t.Status == model.StatusTimeout && groupStatus != model.StatusCrash && groupStatus != model.StatusLeak {
				groupStatus = model.StatusTimeout
			} else if t.Status == model.StatusFailed && groupStatus == model.StatusPassed {
				groupStatus = model.StatusFailed
			}
		}

		defaultExpanded := (groupStatus != model.StatusPassed)
		isExpanded := defaultExpanded
		if m.expandedGroups != nil {
			if exp, ok := m.expandedGroups[fn]; ok {
				isExpanded = exp
			}
		}

		// Parent File group
		rows = append(rows, TreeRow{
			IsGroup:     true,
			File:        fn,
			SubResults:  tests,
			Expanded:    isExpanded,
			PassedCount: passed,
			TotalCount:  len(tests),
			GroupStatus: groupStatus,
		})

		// Child sub-tests if group is expanded
		if isExpanded {
			for _, t := range tests {
				rows = append(rows, TreeRow{
					IsGroup:    false,
					File:       fn,
					Result:     t,
					SubResults: tests,
				})
			}
		}
	}

	return rows
}
