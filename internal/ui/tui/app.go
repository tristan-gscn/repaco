package tui

import (
	"context"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/tristan-gscn/repaco/internal/engine"
)

// Run launches the interactive Bubbletea TUI application.
func Run(ctx context.Context, eng *engine.Engine) error {
	m := NewModel(ctx, eng)
	p := tea.NewProgram(m, tea.WithAltScreen())
	_, err := p.Run()
	return err
}
