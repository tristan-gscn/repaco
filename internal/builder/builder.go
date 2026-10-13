package builder

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// MakefileCheckResult holds the verification of 42 Makefile norms.
type MakefileCheckResult struct {
	Present        bool     `json:"present"`
	HasRules       []string `json:"has_rules"`
	MissingRules   []string `json:"missing_rules"`
	HasWall        bool     `json:"has_wall"`
	HasWextra      bool     `json:"has_wextra"`
	HasWerror      bool     `json:"has_werror"`
	RelinkDetected bool     `json:"relink_detected"`
	Error          string   `json:"error,omitempty"`
}

// Builder orchestrates C builds, Makefile checks, and harness linking.
type Builder struct {
	Compiler string
}

// New creates a new Builder with standard C compiler.
func New() *Builder {
	compiler := "cc"
	if _, err := exec.LookPath("clang"); err == nil {
		compiler = "clang"
	}
	return &Builder{
		Compiler: compiler,
	}
}

// CheckMakefile verifies rule presence, flags, and relink behavior.
func (b *Builder) CheckMakefile(ctx context.Context, dir string, targetName string) (*MakefileCheckResult, error) {
	makefilePath := filepath.Join(dir, "Makefile")
	if _, err := os.Stat(makefilePath); err != nil {
		return &MakefileCheckResult{Present: false}, nil
	}

	content, err := os.ReadFile(makefilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read Makefile: %w", err)
	}

	sContent := string(content)
	res := &MakefileCheckResult{
		Present:   true,
		HasWall:   strings.Contains(sContent, "-Wall"),
		HasWextra: strings.Contains(sContent, "-Wextra"),
		HasWerror: strings.Contains(sContent, "-Werror"),
	}

	mandatoryRules := []string{"all", "clean", "fclean", "re"}
	for _, rule := range mandatoryRules {
		cmd := exec.CommandContext(ctx, "make", "-n", "-C", dir, rule)
		if err := cmd.Run(); err == nil {
			res.HasRules = append(res.HasRules, rule)
		} else {
			res.MissingRules = append(res.MissingRules, rule)
		}
	}

	// Relink check: run make, then run make again and check output
	cmd1 := exec.CommandContext(ctx, "make", "-C", dir)
	_ = cmd1.Run()

	var stdout2 bytes.Buffer
	cmd2 := exec.CommandContext(ctx, "make", "-C", dir)
	cmd2.Stdout = &stdout2
	cmd2.Stderr = &stdout2
	_ = cmd2.Run()

	outStr := strings.ToLower(stdout2.String())
	// If output contains compilation commands (like "cc " or "clang " or "gcc "), relink is happening
	if strings.Contains(outStr, "cc ") || strings.Contains(outStr, "clang ") || strings.Contains(outStr, "gcc ") {
		res.RelinkDetected = true
	}

	return res, nil
}

// CompileHarness compiles a given C harness source into an executable.
func (b *Builder) CompileHarness(ctx context.Context, srcFile, outBinary string, includeDirs []string, extraFlags []string, linkLibs []string) (string, error) {
	args := []string{"-Wall", "-Wextra", "-Werror"}
	for _, inc := range includeDirs {
		args = append(args, "-I"+inc)
	}
	args = append(args, extraFlags...)
	args = append(args, srcFile, "-o", outBinary)
	args = append(args, linkLibs...)

	cmd := exec.CommandContext(ctx, b.Compiler, args...)
	var combined bytes.Buffer
	cmd.Stdout = &combined
	cmd.Stderr = &combined

	err := cmd.Run()
	if err != nil {
		return combined.String(), fmt.Errorf("compilation failed: %w\n%s", err, combined.String())
	}
	return combined.String(), nil
}

// ExecuteMake runs a specific make target in dir.
func (b *Builder) ExecuteMake(ctx context.Context, dir, target string, timeout time.Duration) (string, error) {
	c, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	var args []string
	args = append(args, "-C", dir)
	if target != "" {
		args = append(args, target)
	}

	cmd := exec.CommandContext(c, "make", args...)
	var output bytes.Buffer
	cmd.Stdout = &output
	cmd.Stderr = &output

	err := cmd.Run()
	if err != nil {
		return output.String(), fmt.Errorf("make %s failed: %w\n%s", target, err, output.String())
	}
	return output.String(), nil
}
