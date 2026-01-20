package views

import (
	"fmt"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/ahmabora1/rcm/internal/config"
	"github.com/ahmabora1/rcm/internal/parser"
	"github.com/ahmabora1/rcm/internal/tui/components"
	"github.com/ahmabora1/rcm/internal/tui/styles"
)

// ListState represents the view state
type ListState int

const (
	ListStateLoading ListState = iota
	ListStateReady
	ListStateError
)

// ListModel is the Bubbletea model for the list view
type ListModel struct {
	state    ListState
	config   *config.Config
	services []parser.Service
	table    table.Model
	spinner  spinner.Model
	err      error
	width    int
	height   int
	showHelp bool
}

// Messages
type servicesLoadedMsg struct {
	services []parser.Service
}

type listErrMsg struct {
	err error
}

// NewListModel creates a new list view model
func NewListModel(cfg *config.Config) ListModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(styles.Primary)

	return ListModel{
		state:   ListStateLoading,
		config:  cfg,
		spinner: s,
	}
}

// Init initializes the model
func (m ListModel) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		m.loadServicesCmd(),
	)
}

// Update handles messages
func (m ListModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "?":
			m.showHelp = !m.showHelp
		case "r":
			m.state = ListStateLoading
			return m, tea.Batch(m.spinner.Tick, m.loadServicesCmd())
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		if m.state == ListStateReady {
			m.table.SetHeight(m.height - 10)
		}

	case servicesLoadedMsg:
		m.state = ListStateReady
		m.services = msg.services
		height := m.height - 10
		if height < 5 {
			height = 10
		}
		m.table = components.NewSimpleServiceTable(msg.services, height)
		return m, nil

	case listErrMsg:
		m.state = ListStateError
		m.err = msg.err
		return m, nil

	case spinner.TickMsg:
		m.spinner, cmd = m.spinner.Update(msg)
		cmds = append(cmds, cmd)
	}

	if m.state == ListStateReady {
		m.table, cmd = m.table.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

// View renders the UI
func (m ListModel) View() string {
	switch m.state {
	case ListStateLoading:
		return fmt.Sprintf("\n  %s Loading services...\n", m.spinner.View())

	case ListStateError:
		return styles.Error.Render(fmt.Sprintf("\n  Error: %v\n\n  Press q to quit.\n", m.err))

	case ListStateReady:
		var s string
		s += "\n"
		s += styles.Title.Render("  RCM Services") + "\n\n"
		s += m.table.View() + "\n"

		if m.showHelp {
			s += "\n" + styles.HelpBar.Render("  ↑/k: up • ↓/j: down • q: quit • r: refresh • ?: toggle help")
		} else {
			s += "\n" + styles.HelpBar.Render("  Press ? for help")
		}

		return s
	}

	return ""
}

// loadServicesCmd creates a command to load services
func (m ListModel) loadServicesCmd() tea.Cmd {
	return func() tea.Msg {
		if m.config.Paths.Caddyfile == "" {
			return listErrMsg{err: fmt.Errorf("caddyfile path not configured")}
		}

		services, err := parser.ParseFile(m.config.Paths.Caddyfile)
		if err != nil {
			return listErrMsg{err: err}
		}
		return servicesLoadedMsg{services: services}
	}
}
