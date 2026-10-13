package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/mattn/go-isatty"
	"github.com/tristan-gscn/repaco/internal/config"
	"github.com/tristan-gscn/repaco/internal/engine"
	"github.com/tristan-gscn/repaco/internal/model"
	"github.com/tristan-gscn/repaco/internal/ui/cli"
	"github.com/tristan-gscn/repaco/internal/ui/tui"

	// Register all project suites
	_ "github.com/tristan-gscn/repaco/internal/projects/ftprintf"
	_ "github.com/tristan-gscn/repaco/internal/projects/getnextline"
	_ "github.com/tristan-gscn/repaco/internal/projects/libft"
	_ "github.com/tristan-gscn/repaco/internal/projects/pushswap"
)

var (
	version = "0.1.0-alpha"
)

func main() {
	cfg := config.NewDefaultConfig()

	var (
		dirFlag        string
		projectFlag    string
		noTuiFlag      bool
		failFastFlag   bool
		filterFlag     string
		timeoutFlag    time.Duration
		verboseFlag    bool
		showVersion    bool
	)

	flag.StringVar(&dirFlag, "dir", ".", "Directory of the 42 project to test (default: current directory)")
	flag.StringVar(&projectFlag, "project", "", "Override project auto-detection (e.g. libft, ft_printf, get_next_line, push_swap)")
	flag.BoolVar(&noTuiFlag, "no-tui", false, "Disable interactive TUI and run with standard console output")
	flag.BoolVar(&failFastFlag, "fail-fast", false, "Stop execution on first test failure")
	flag.StringVar(&filterFlag, "filter", "", "Filter tests by module or function name")
	flag.DurationVar(&timeoutFlag, "timeout", 5*time.Second, "Default timeout per test")
	flag.BoolVar(&verboseFlag, "verbose", false, "Verbose output")
	flag.BoolVar(&showVersion, "version", false, "Show repaco version")

	flag.Usage = func() {
		fmt.Printf("repaco - Universal 42 Project Tester (v%s)\n\n", version)
		fmt.Println("Usage:")
		fmt.Println("  repaco [options] [path_to_project]")
		fmt.Println("\nOptions:")
		flag.PrintDefaults()
	}

	flag.Parse()

	if showVersion {
		fmt.Printf("repaco version %s\n", version)
		os.Exit(0)
	}

	// Positional argument takes precedence for target directory
	if args := flag.Args(); len(args) > 0 {
		cfg.TargetDir = args[0]
	} else {
		cfg.TargetDir = dirFlag
	}

	cfg.ProjectOverride = projectFlag
	cfg.NoTUI = noTuiFlag
	cfg.FailFast = failFastFlag
	cfg.FilterModule = filterFlag
	cfg.DefaultTimeout = timeoutFlag
	cfg.Verbose = verboseFlag

	// Auto-disable TUI if stdout is not a terminal
	isTTY := isatty.IsTerminal(os.Stdout.Fd()) || isatty.IsCygwinTerminal(os.Stdout.Fd())
	if !isTTY {
		cfg.NoTUI = true
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	eng := engine.New(cfg)
	eventsChan := make(chan model.EngineEvent, 100)

	if cfg.NoTUI {
		printer := cli.NewPrinter()
		var suiteResult *model.SuiteResult
		var runErr error

		go func() {
			suiteResult, runErr = eng.Run(ctx, eventsChan)
		}()

		for ev := range eventsChan {
			switch ev.Type {
			case model.EventDetectDone:
				if ev.Detection != nil {
					printer.PrintDetection(ev.Detection)
				}
			case model.EventTestCompleted:
				if ev.TestResult != nil {
					printer.PrintTestResult(ev.TestResult)
				}
			case model.EventBuildDone:
				if ev.TestResult != nil && ev.TestResult.Status != model.StatusPassed {
					printer.PrintTestResult(ev.TestResult)
				}
			case model.EventFatalError:
				fmt.Fprintf(os.Stderr, "Error: %v\n", ev.Error)
			}
		}

		if runErr != nil {
			fmt.Fprintf(os.Stderr, "Execution aborted: %v\n", runErr)
			os.Exit(1)
		}

		if suiteResult != nil {
			printer.PrintSummary(suiteResult)
			if !suiteResult.Stats.IsPerfect() {
				os.Exit(1)
			}
		}
	} else {
		go func() {
			_, _ = eng.Run(ctx, eventsChan)
		}()

		if err := tui.Run(eventsChan); err != nil {
			fmt.Fprintf(os.Stderr, "TUI error: %v\n", err)
			os.Exit(1)
		}
	}
}
