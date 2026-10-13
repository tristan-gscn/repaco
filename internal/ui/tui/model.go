package tui

import (
	"os"
	"path/filepath"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/tristan-gscn/repaco/internal/detector"
	"github.com/tristan-gscn/repaco/internal/model"
)

type EventMsg model.EngineEvent

type Model struct {
	detection    *model.DetectionResult
	results      []model.TestResult
	cursor       int
	running      bool
	done         bool
	statusMsg    string
	fatalErr     string
	width        int
	height       int
	eventsChan   <-chan model.EngineEvent
	stats        model.SummaryStats
	spinner      spinner.Model
	totalTests   int
	currentTest  int
	startTime    time.Time
	elapsedTime  time.Duration

	// Discovery mode when no project is detected
	isDiscovery     bool
	knownProjects   []model.ProjectMetadata
	discoveryCursor int
	targetDir       string
}

func NewModel(eventsChan <-chan model.EngineEvent) Model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(JBAccent)

	known := []model.ProjectMetadata{
		detector.KnownProjects[model.ProjectLibft],
		detector.KnownProjects[model.ProjectFtPrintf],
		detector.KnownProjects[model.ProjectGetNextLine],
		detector.KnownProjects[model.ProjectPushSwap],
	}

	cwd, _ := os.Getwd()

	return Model{
		eventsChan:    eventsChan,
		running:       true,
		statusMsg:     "Scanning directory for 42 project...",
		spinner:       s,
		startTime:     time.Now(),
		knownProjects: known,
		targetDir:     cwd,
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, waitForEvent(m.eventsChan))
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
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case spinner.TickMsg:
		if m.running {
			m.elapsedTime = time.Since(m.startTime)
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			cmds = append(cmds, cmd)
		}

	case EventMsg:
		ev := model.EngineEvent(msg)
		switch ev.Type {
		case model.EventDetectDone:
			m.detection = ev.Detection
			if ev.Detection != nil && ev.Detection.RootDir != "" {
				m.targetDir = ev.Detection.RootDir
			}
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
			m.currentTest = ev.Current
			m.totalTests = ev.Total
		case model.EventTestCompleted:
			if ev.TestResult != nil {
				m.results = append(m.results, *ev.TestResult)
				m.updateStats()
			}
		case model.EventSuiteDone:
			m.running = false
			m.done = true
			m.elapsedTime = time.Since(m.startTime)
			m.statusMsg = "Testing completed."
			return m, nil
		case model.EventFatalError:
			m.running = false
			m.done = true
			m.isDiscovery = true
			m.fatalErr = ev.Error.Error()
			return m, nil
		}
		cmds = append(cmds, waitForEvent(m.eventsChan))

	case tea.KeyMsg:
		switch msg.String() {
		case "esc", "ctrl+c", "q":
			return m, tea.Quit

		case "up":
			if m.isDiscovery {
				if m.discoveryCursor > 0 {
					m.discoveryCursor--
				}
				return m, nil
			}
			if m.cursor > 0 {
				m.cursor--
			}

		case "down":
			if m.isDiscovery {
				if m.discoveryCursor < len(m.knownProjects)-1 {
					m.discoveryCursor++
				}
				return m, nil
			}
			visible := m.results
			if m.cursor < len(visible)-1 {
				m.cursor++
			}
		}
	}

	return m, tea.Batch(cmds...)
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

// checkFileExists checks if a file exists relative to targetDir or anywhere in tree
func (m Model) checkFileExists(filename string) bool {
	p := filepath.Join(m.targetDir, filename)
	if _, err := os.Stat(p); err == nil {
		return true
	}
	var found bool
	_ = filepath.Walk(m.targetDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || found {
			return nil
		}
		if !info.IsDir() && info.Name() == filename {
			found = true
		}
		return nil
	})
	return found
}
