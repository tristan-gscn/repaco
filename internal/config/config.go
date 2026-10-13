package config

import (
	"time"
)

// Config holds all user-configurable parameters for repaco.
type Config struct {
	// TargetDir is the directory containing the 42 project to test.
	TargetDir string

	// ProjectOverride forces a specific project tester instead of auto-detecting.
	ProjectOverride string

	// FailFast stops test execution at the first failure.
	FailFast bool

	// FilterModule restricts tests to matching module names (e.g. "ft_split", "100_numbers").
	FilterModule string

	// FilterCategory restricts tests to matching category (MANDATORY, EDGE_CASE, etc.).
	FilterCategory string

	// EnableASan enables AddressSanitizer and UndefinedBehaviorSanitizer during harness compilation.
	EnableASan bool

	// EnableLeaksCheck enables macOS leaks tool checking when available.
	EnableLeaksCheck bool

	// DefaultTimeout specifies maximum time allowed per test case.
	DefaultTimeout time.Duration

	// ExportJSON specifies path to dump full results as JSON.
	ExportJSON string

	// Verbose outputs additional debugging information.
	Verbose bool
}

// NewDefaultConfig returns a standard configuration suitable for everyday usage.
func NewDefaultConfig() *Config {
	return &Config{
		TargetDir:        ".",
		ProjectOverride:  "",
		FailFast:         false,
		FilterModule:     "",
		FilterCategory:   "",
		EnableASan:       true,
		EnableLeaksCheck: true,
		DefaultTimeout:   5 * time.Second,
		ExportJSON:       "",
		Verbose:          false,
	}
}
