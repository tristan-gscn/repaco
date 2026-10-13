package runner

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/tristan-gscn/repaco/internal/model"
)

// ExecutionResult contains raw metrics from running a process.
type ExecutionResult struct {
	Stdout      string
	Stderr      string
	Combined    string
	ExitCode    int
	Duration    time.Duration
	TimedOut    bool
	Crashed     bool
	Signal      string
	LeakReport  string
	Status      model.Status
	Explanation string
}

// Runner handles safe subprocess execution.
type Runner struct {
	DefaultTimeout time.Duration
}

// New creates a new Runner instance.
func New(defaultTimeout time.Duration) *Runner {
	if defaultTimeout <= 0 {
		defaultTimeout = 5 * time.Second
	}
	return &Runner{
		DefaultTimeout: defaultTimeout,
	}
}

// RunOptions configures the execution parameters.
type RunOptions struct {
	Args    []string
	Stdin   string
	Cwd     string
	Env     []string
	Timeout time.Duration
}

// Run executes a binary safely and captures all diagnostics.
func (r *Runner) Run(ctx context.Context, binaryPath string, opts RunOptions) (*ExecutionResult, error) {
	timeout := opts.Timeout
	if timeout <= 0 {
		timeout = r.DefaultTimeout
	}

	execCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := exec.CommandContext(execCtx, binaryPath, opts.Args...)
	if opts.Cwd != "" {
		cmd.Dir = opts.Cwd
	}

	// Setup environment: inherit current env + pass ASAN settings
	asanOpts := "abort_on_error=1:color=never"
	if runtime.GOOS == "linux" {
		asanOpts = "detect_leaks=1:abort_on_error=1:color=never"
	}
	cmd.Env = append(os.Environ(),
		"ASAN_OPTIONS="+asanOpts,
	)
	cmd.Env = append(cmd.Env, opts.Env...)

	var stdoutBuf, stderrBuf bytes.Buffer
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf

	if opts.Stdin != "" {
		cmd.Stdin = strings.NewReader(opts.Stdin)
	}

	start := time.Now()
	err := cmd.Run()
	duration := time.Since(start)

	res := &ExecutionResult{
		Stdout:   stdoutBuf.String(),
		Stderr:   stderrBuf.String(),
		Duration: duration,
	}
	res.Combined = res.Stdout + res.Stderr

	// Check timeout
	if errors.Is(execCtx.Err(), context.DeadlineExceeded) {
		res.TimedOut = true
		res.Status = model.StatusTimeout
		res.Explanation = fmt.Sprintf("Process timed out after %v (possible infinite loop or blocking read)", timeout)
		return res, nil
	}

	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			res.ExitCode = exitErr.ExitCode()
			if ws, ok := exitErr.Sys().(syscall.WaitStatus); ok {
				if ws.Signaled() {
					res.Crashed = true
					res.Signal = ws.Signal().String()
					res.Status = model.StatusCrash
					res.Explanation = r.formatCrashExplanation(ws.Signal())
				}
			}
		} else {
			return nil, fmt.Errorf("failed to run binary: %w", err)
		}
	}

	// Inspect ASan output in stderr
	if strings.Contains(res.Stderr, "AddressSanitizer") || strings.Contains(res.Stderr, "LeakSanitizer") {
		res.LeakReport = res.Stderr
		if strings.Contains(res.Stderr, "leak") {
			res.Status = model.StatusLeak
			res.Explanation = "Memory leak detected by sanitizer"
		} else {
			res.Status = model.StatusCrash
			res.Explanation = "Memory corruption detected by AddressSanitizer (heap/stack overflow, use-after-free, or invalid read/write)"
		}
	}

	if res.Status == "" {
		res.Status = model.StatusPassed
	}

	return res, nil
}

func (r *Runner) formatCrashExplanation(sig syscall.Signal) string {
	switch sig {
	case syscall.SIGSEGV:
		return "Segmentation fault (SIGSEGV): tried to access unallocated or protected memory (null pointer dereference or out-of-bounds access)"
	case syscall.SIGBUS:
		return "Bus error (SIGBUS): misaligned memory access or invalid memory address"
	case syscall.SIGABRT:
		return "Abort (SIGABRT): program aborted abnormally (double free, corrupted heap, or assert failed)"
	case syscall.SIGFPE:
		return "Floating point exception (SIGFPE): division by zero or arithmetic overflow"
	default:
		return fmt.Sprintf("Killed by signal %s", sig.String())
	}
}

// RunMemoryCheck runs macOS leaks tool against a process command.
func (r *Runner) RunMemoryCheck(ctx context.Context, binaryPath string, args ...string) (string, bool, error) {
	cmdArgs := append([]string{"--atExit", "--", binaryPath}, args...)
	cmd := exec.CommandContext(ctx, "leaks", cmdArgs...)

	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = io.Discard

	err := cmd.Run()
	output := out.String()
	hasLeaks := strings.Contains(output, "leaks for") && !strings.Contains(output, "0 leaks for 0 total leaked bytes")

	return output, hasLeaks, err
}
