package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/tristan-gscn/repaco/internal/model"
)

type EventMsg model.EngineEvent

type Model struct {
	detection        *model.DetectionResult
	results          []model.TestResult
	cursor           int
	activePane       int // 0 = list, 1 = details
	detailScroll     int
	filterOnlyFailed bool
	running          bool
	done             bool
	statusMsg        string
	width            int
	height           int
	eventsChan       <-chan model.EngineEvent
	stats            model.SummaryStats
}

func NewModel(eventsChan <-chan model.EngineEvent) Model {
	return Model{
		eventsChan: eventsChan,
		running:    true,
		statusMsg:  "Initializing repaco...",
	}
}

func (m Model) Init() tea.Cmd {
	return waitForEvent(m.eventsChan)
}

func waitForEvent(ch <-chan model.EngineEvent) tea.Cmd {
	return func() tea.Msg {
		ev, ok := <-ch
		if !ok {
			return EventMsg{Type: model.EventSuiteDone}
		}
		return EventMsg(ev)
	}
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case EventMsg:
		ev := model.EngineEvent(msg)
		switch ev.Type {
		case model.EventDetectDone:
			m.detection = ev.Detection
			m.statusMsg = ev.Message
		case model.EventBuildStarted:
			m.statusMsg = ev.Message
		case model.EventBuildDone:
			if ev.TestResult != nil {
				m.results = append(m.results, *ev.TestResult)
				m.updateStats()
			}
		case model.EventTestStarted:
			m.statusMsg = ev.Message
		case model.EventTestCompleted:
			if ev.TestResult != nil {
				m.results = append(m.results, *ev.TestResult)
				m.updateStats()
			}
		case model.EventSuiteDone:
			m.running = false
			m.done = true
			m.statusMsg = "Test suite finished. Use keys to inspect details."
			return m, nil
		case model.EventFatalError:
			m.running = false
			m.done = true
			m.statusMsg = "Error: " + ev.Error.Error()
			return m, nil
		}
		return m, waitForEvent(m.eventsChan)

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "up", "k":
			if m.activePane == 0 {
				if m.cursor > 0 {
					m.cursor--
					m.detailScroll = 0
				}
			} else {
				if m.detailScroll > 0 {
					m.detailScroll--
				}
			}
		case "down", "j":
			visible := m.getVisibleResults()
			if m.activePane == 0 {
				if m.cursor < len(visible)-1 {
					m.cursor++
					m.detailScroll = 0
				}
			} else {
				m.detailScroll++
			}
		case "tab":
			m.activePane = 1 - m.activePane
		case "f":
			m.filterOnlyFailed = !m.filterOnlyFailed
			m.cursor = 0
			m.detailScroll = 0
		case "home", "g":
			if m.activePane == 0 {
				m.cursor = 0
			} else {
				m.detailScroll = 0
			}
		case "end", "G":
			visible := m.getVisibleResults()
			if m.activePane == 0 && len(visible) > 0 {
				m.cursor = len(visible) - 1
			}
		}
	}

	return m, nil
}

func (m *Model) updateStats() {
	var s model.SummaryStats
	s.Total = len(m.results)
	for _, r := range m.results {
		switch r.Status {
		case model.StatusPassed:
			s.Passed++
		case model.StatusFailed:
			s.Failed++
		case model.StatusCrash:
			s.Crashed++
		case model.StatusTimeout:
			s.Timeout++
		case model.StatusLeak:
			s.Leaked++
		case model.StatusSkipped:
			s.Skipped++
		}
	}
	m.stats = s
}

func (m Model) getVisibleResults() []model.TestResult {
	if !m.filterOnlyFailed {
		return m.results
	}
	var filtered []model.TestResult
	for _, r := range m.results {
		if r.Status != model.StatusPassed && r.Status != model.StatusSkipped {
			filtered = append(filtered, r)
		}
	}
	return filtered
}
