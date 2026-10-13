package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/tristan-gscn/repaco/internal/model"
)

// Run launches the interactive Bubbletea TUI application.
func Run(eventsChan <-chan model.EngineEvent) error {
	m := NewModel(eventsChan)
	p := tea.NewProgram(m, tea.WithAltScreen())
	_, err := p.Run()
	return err
}
