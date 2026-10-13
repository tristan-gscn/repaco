package tui

import (
	"context"
	"os"
	"path/filepath"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/tristan-gscn/repaco/internal/detector"
	"github.com/tristan-gscn/repaco/internal/engine"
	"github.com/tristan-gscn/repaco/internal/model"
)

type EventMsg model.EngineEvent

type Model struct {
	ctx          context.Context
	eng          *engine.Engine
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

	// Nearby projects mode when detected around
	isNearbyMode   bool
	nearbyProjects []model.DiscoveredProject
	nearbyCursor   int

	// Discovery mode when no project is detected anywhere
	isDiscovery     bool
	knownProjects   []model.ProjectMetadata
	discoveryCursor int
	targetDir       string

	// Detail pane focus and scrolling
	focusRight      bool
	detailScroll    int
	collapsedGroups map[string]bool
}

func NewModel(ctx context.Context, eng *engine.Engine) Model {
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
	ch := make(chan model.EngineEvent, 100)

	// Launch engine in background
	go func() {
		_, _ = eng.Run(ctx, ch)
	}()

	return Model{
		ctx:             ctx,
		eng:             eng,
		eventsChan:      ch,
		running:         true,
		statusMsg:       "Scanning directory for 42 project...",
		spinner:         s,
		startTime:       time.Now(),
		knownProjects:   known,
		targetDir:       cwd,
		collapsedGroups: make(map[string]bool),
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

		case model.EventProjectsNearbyFound:
			m.running = false
			m.isNearbyMode = true
			m.nearbyProjects = ev.ProjectsNearby
			m.nearbyCursor = 0
			m.statusMsg = ev.Message
			return m, nil

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
		case "ctrl+c", "q":
			return m, tea.Quit

		case "esc":
			if m.focusRight {
				m.focusRight = false
				return m, nil
			}
			return m, tea.Quit

		case "right":
			if !m.isNearbyMode && !m.isDiscovery && len(m.results) > 0 {
				treeRows := m.buildTreeRows()
				if m.cursor >= 0 && m.cursor < len(treeRows) {
					row := treeRows[m.cursor]
					if row.IsGroup && !row.Expanded {
						if m.collapsedGroups == nil {
							m.collapsedGroups = make(map[string]bool)
						}
						m.collapsedGroups[row.File] = false
						return m, nil
					}
				}
				m.focusRight = true
				return m, nil
			}

		case "left":
			if m.focusRight {
				m.focusRight = false
				return m, nil
			}
			if !m.isNearbyMode && !m.isDiscovery && len(m.results) > 0 {
				treeRows := m.buildTreeRows()
				if m.cursor >= 0 && m.cursor < len(treeRows) {
					row := treeRows[m.cursor]
					if row.IsGroup && row.Expanded {
						if m.collapsedGroups == nil {
							m.collapsedGroups = make(map[string]bool)
						}
						m.collapsedGroups[row.File] = true
						return m, nil
					} else if !row.IsGroup {
						for i := m.cursor - 1; i >= 0; i-- {
							if treeRows[i].IsGroup && treeRows[i].File == row.File {
								m.cursor = i
								m.detailScroll = 0
								break
							}
						}
						return m, nil
					}
				}
			}

		case "tab":
			if !m.isNearbyMode && !m.isDiscovery && len(m.results) > 0 {
				m.focusRight = !m.focusRight
				return m, nil
			}

		case "enter", " ":
			if m.isNearbyMode && len(m.nearbyProjects) > 0 {
				selected := m.nearbyProjects[m.nearbyCursor]
				m.isNearbyMode = false
				m.running = true
				m.results = nil
				m.cursor = 0
				m.detailScroll = 0
				m.focusRight = false
				m.stats = model.SummaryStats{}
				m.startTime = time.Now()
				m.statusMsg = "Testing " + selected.Detection.Project.Name + "..."

				newCh := make(chan model.EngineEvent, 100)
				m.eventsChan = newCh

				go func() {
					_, _ = m.eng.RunWithDir(m.ctx, selected.AbsPath, newCh)
				}()

				return m, waitForEvent(newCh)
			}
			if !m.isNearbyMode && !m.isDiscovery && len(m.results) > 0 {
				treeRows := m.buildTreeRows()
				if m.cursor >= 0 && m.cursor < len(treeRows) {
					row := treeRows[m.cursor]
					if row.IsGroup {
						if m.collapsedGroups == nil {
							m.collapsedGroups = make(map[string]bool)
						}
						m.collapsedGroups[row.File] = !m.collapsedGroups[row.File]
						newRows := m.buildTreeRows()
						if m.cursor >= len(newRows) {
							m.cursor = len(newRows) - 1
							if m.cursor < 0 {
								m.cursor = 0
							}
						}
						return m, nil
					} else {
						m.focusRight = true
						return m, nil
					}
				}
			}

		case "up":
			if m.isNearbyMode {
				if m.nearbyCursor > 0 {
					m.nearbyCursor--
				}
				return m, nil
			}
			if m.isDiscovery {
				if m.discoveryCursor > 0 {
					m.discoveryCursor--
				}
				return m, nil
			}
			if m.focusRight {
				if m.detailScroll > 0 {
					m.detailScroll--
				}
				return m, nil
			}
			if m.cursor > 0 {
				m.cursor--
				m.detailScroll = 0
			}

		case "down":
			if m.isNearbyMode {
				if m.nearbyCursor < len(m.nearbyProjects)-1 {
					m.nearbyCursor++
				}
				return m, nil
			}
			if m.isDiscovery {
				if m.discoveryCursor < len(m.knownProjects)-1 {
					m.discoveryCursor++
				}
				return m, nil
			}
			if m.focusRight {
				m.detailScroll++
				return m, nil
			}
			treeRows := m.buildTreeRows()
			if m.cursor < len(treeRows)-1 {
				m.cursor++
				m.detailScroll = 0
			}

		case "pgup", "pageup":
			if m.focusRight {
				m.detailScroll -= 6
				if m.detailScroll < 0 {
					m.detailScroll = 0
				}
			} else {
				m.cursor -= 10
				if m.cursor < 0 {
					m.cursor = 0
				}
				m.detailScroll = 0
			}

		case "pgdown", "pagedown":
			if m.focusRight {
				m.detailScroll += 6
			} else {
				treeRows := m.buildTreeRows()
				m.cursor += 10
				if m.cursor >= len(treeRows) {
					m.cursor = len(treeRows) - 1
					if m.cursor < 0 {
						m.cursor = 0
					}
				}
				m.detailScroll = 0
			}

		case "home":
			if m.focusRight {
				m.detailScroll = 0
			} else {
				m.cursor = 0
				m.detailScroll = 0
			}

		case "end":
			if m.focusRight {
				m.detailScroll = 9999
			} else {
				treeRows := m.buildTreeRows()
				if len(treeRows) > 0 {
					m.cursor = len(treeRows) - 1
				}
				m.detailScroll = 0
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
