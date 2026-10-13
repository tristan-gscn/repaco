package model

import "time"

// SummaryStats contains aggregated metrics of a test session.
type SummaryStats struct {
	Total    int `json:"total"`
	Passed   int `json:"passed"`
	Failed   int `json:"failed"`
	Crashed  int `json:"crashed"`
	Timeout  int `json:"timeout"`
	Leaked   int `json:"leaked"`
	Skipped  int `json:"skipped"`
}

// SuccessRate returns percentage of passed tests over total applicable tests.
func (s SummaryStats) SuccessRate() float64 {
	applicable := s.Total - s.Skipped
	if applicable <= 0 {
		return 0.0
	}
	return (float64(s.Passed) / float64(applicable)) * 100.0
}

// IsPerfect returns true if all applicable tests passed with 0 failures, crashes, timeouts, or leaks.
func (s SummaryStats) IsPerfect() bool {
	return s.Passed > 0 && s.Failed == 0 && s.Crashed == 0 && s.Timeout == 0 && s.Leaked == 0
}

// SuiteResult contains all test results for a given module or project.
type SuiteResult struct {
	ProjectID ProjectID     `json:"project_id"`
	Stats     SummaryStats  `json:"stats"`
	Duration  time.Duration `json:"duration"`
	Results   []TestResult  `json:"results"`
}

// UpdateStats recalculates the summary metrics from Results list.
func (sr *SuiteResult) UpdateStats() {
	var s SummaryStats
	s.Total = len(sr.Results)
	for _, r := range sr.Results {
		switch r.Status {
		case StatusPassed:
			s.Passed++
		case StatusFailed:
			s.Failed++
		case StatusCrash:
			s.Crashed++
		case StatusTimeout:
			s.Timeout++
		case StatusLeak:
			s.Leaked++
		case StatusSkipped:
			s.Skipped++
		}
	}
	sr.Stats = s
}
