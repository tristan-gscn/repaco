package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/mattn/go-isatty"
	"github.com/tristan-gscn/repaco/internal/config"
	"github.com/tristan-gscn/repaco/internal/engine"
	"github.com/tristan-gscn/repaco/internal/model"
	"github.com/tristan-gscn/repaco/internal/ui/agent"
	"github.com/tristan-gscn/repaco/internal/ui/tui"
	"github.com/tristan-gscn/repaco/internal/updater"

	// Register all project suites
	_ "github.com/tristan-gscn/repaco/internal/projects/codexion"
	_ "github.com/tristan-gscn/repaco/internal/projects/ftprintf"
	_ "github.com/tristan-gscn/repaco/internal/projects/getnextline"
	_ "github.com/tristan-gscn/repaco/internal/projects/libft"
	_ "github.com/tristan-gscn/repaco/internal/projects/pushswap"
)

func main() {
	cfg := config.NewDefaultConfig()

	showVersion := flag.Bool("v", false, "Show version")
	flag.BoolVar(showVersion, "version", false, "Show version")

	agentMode := flag.Bool("agent", false, "Agent mode: output AI-optimized markdown diagnostics with code snippets and diffs")
	jsonMode := flag.Bool("json", false, "JSON mode: output pure machine-readable JSON")
	noUpdate := flag.Bool("no-update", false, "Skip checking for online updates")

	flag.Usage = func() {
		fmt.Printf("repaco - Universal 42 Project Tester (v%s)\n\n", updater.CurrentVersion)
		fmt.Println("Usage:")
		fmt.Println("  repaco [options] [path]    Run tests on target project directory")
		fmt.Println("\nModes:")
		fmt.Println("  Interactive (default):     JetBrains-style TUI test tree and inspector")
		fmt.Println("  --agent:                   Headless AI agent mode with Markdown diagnostics")
		fmt.Println("  --json:                    Raw JSON output for CI / automation pipelines")
		fmt.Println("\nOptions:")
		flag.PrintDefaults()
	}

	flag.Parse()

	if *showVersion {
		if updater.GitCommit != "" {
			fmt.Printf("repaco v%s (%s)\n", updater.CurrentVersion, updater.GitCommit)
		} else {
			fmt.Printf("repaco v%s\n", updater.CurrentVersion)
		}
		os.Exit(0)
	}

	// Positional argument for target project directory
	if args := flag.Args(); len(args) > 0 {
		cfg.TargetDir = args[0]
	} else {
		cfg.TargetDir = "."
	}
	if absDir, err := filepath.Abs(cfg.TargetDir); err == nil {
		cfg.TargetDir = absDir
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	// If --agent, --json, or non-interactive terminal (e.g. piped or run by an AI agent subprocess)
	isTerm := isatty.IsTerminal(os.Stdout.Fd()) || isatty.IsCygwinTerminal(os.Stdout.Fd())
	useAgent := *agentMode || (!isTerm && !*jsonMode)
	useJSON := *jsonMode

	// Check for updates online if interactive and not disabled
	if !useAgent && !useJSON && !*noUpdate && isTerm {
		updater.CheckAndPromptUpdate(ctx)
	}

	eng := engine.New(cfg)

	if useAgent || useJSON {
		eventsChan := make(chan model.EngineEvent, 100)
		go func() {
			for range eventsChan {
			}
		}()
		res, err := eng.Run(ctx, eventsChan)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Engine execution error: %v\n", err)
			os.Exit(1)
		}
		if res == nil {
			fmt.Fprintf(os.Stderr, "No 42 project directly detected in %s. Launch repaco interactively to browse projects.\n", cfg.TargetDir)
			os.Exit(1)
		}

		if useJSON {
			if err := agent.PrintJSON(os.Stdout, res); err != nil {
				fmt.Fprintf(os.Stderr, "JSON encoding error: %v\n", err)
				os.Exit(1)
			}
		} else {
			agent.PrintAgentReport(os.Stdout, res)
		}

		if res == nil || !res.Stats.IsPerfect() {
			os.Exit(1)
		}
		os.Exit(0)
	}

	// Launch full interactive TUI interface
	if err := tui.Run(ctx, eng); err != nil {
		fmt.Fprintf(os.Stderr, "TUI error: %v\n", err)
		os.Exit(1)
	}
}
