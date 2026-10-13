package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/tristan-gscn/repaco/internal/config"
	"github.com/tristan-gscn/repaco/internal/engine"
	"github.com/tristan-gscn/repaco/internal/ui/tui"

	// Register all project suites
	_ "github.com/tristan-gscn/repaco/internal/projects/ftprintf"
	_ "github.com/tristan-gscn/repaco/internal/projects/getnextline"
	_ "github.com/tristan-gscn/repaco/internal/projects/libft"
	_ "github.com/tristan-gscn/repaco/internal/projects/pushswap"
)

var (
	version = "0.1.0"
)

func main() {
	cfg := config.NewDefaultConfig()

	showVersion := flag.Bool("v", false, "Show version")
	flag.BoolVar(showVersion, "version", false, "Show version")

	flag.Usage = func() {
		fmt.Printf("repaco - Universal 42 Project Tester (v%s)\n\n", version)
		fmt.Println("Usage:")
		fmt.Println("  repaco              Launch TUI tester on current directory")
		fmt.Println("  repaco [path]       Launch TUI tester on specified project directory")
		fmt.Println("\nOptions:")
		flag.PrintDefaults()
	}

	flag.Parse()

	if *showVersion {
		fmt.Printf("repaco v%s\n", version)
		os.Exit(0)
	}

	// Positional argument for target project directory
	if args := flag.Args(); len(args) > 0 {
		cfg.TargetDir = args[0]
	} else {
		cfg.TargetDir = "."
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	eng := engine.New(cfg)

	// Launch full TUI interface immediately
	if err := tui.Run(ctx, eng); err != nil {
		fmt.Fprintf(os.Stderr, "TUI error: %v\n", err)
		os.Exit(1)
	}
}
