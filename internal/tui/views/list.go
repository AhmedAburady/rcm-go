package views

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/ahmabora1/rcm/internal/config"
	"github.com/ahmabora1/rcm/internal/parser"
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
	state       ListState
	config      *config.Config
	services    []parser.Service
	spinner     spinner.Model
	err         error
	width       int
	height      int
	showHelp    bool
	selectedIdx int
	scrollTop   int
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
		width:   80,
		height:  24,
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
		case "up", "k":
			if m.selectedIdx > 0 {
				m.selectedIdx--
			}
		case "down", "j":
			if m.selectedIdx < len(m.services)-1 {
				m.selectedIdx++
			}
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case servicesLoadedMsg:
		m.state = ListStateReady
		m.services = msg.services
		return m, nil

	case listErrMsg:
		m.state = ListStateError
		m.err = msg.err
		return m, nil

	case spinner.TickMsg:
		m.spinner, cmd = m.spinner.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

// View renders the UI
func (m ListModel) View() string {
	var lines []string

	// Title
	lines = append(lines, styles.WindowTitle.Render("Services"))
	lines = append(lines, "")

	switch m.state {
	case ListStateLoading:
		lines = append(lines, fmt.Sprintf("%s Loading services...", m.spinner.View()))

	case ListStateError:
		lines = append(lines, styles.Error.Render(fmt.Sprintf("Error: %v", m.err)))

	case ListStateReady:
		// Simple list of services
		for i, svc := range m.services {
			var line string
			name := fmt.Sprintf("%-12s", svc.Name)
			addr := fmt.Sprintf("%-20s", svc.LocalAddr)
			port := fmt.Sprintf(":%d", svc.VPSPort)

			if i == m.selectedIdx {
				// Selected row
				selectedStyle := lipgloss.NewStyle().
					Background(styles.Primary).
					Foreground(styles.White).
					Bold(true)
				line = selectedStyle.Render(fmt.Sprintf(" %s %s %s ", name, addr, port))
			} else {
				line = fmt.Sprintf(" %s %s %s ", name, addr, port)
			}
			lines = append(lines, line)
		}

		if len(m.services) == 0 {
			lines = append(lines, styles.Dimmed.Render("No services found"))
		}
	}

	// Help text
	lines = append(lines, "")
	if m.showHelp {
		lines = append(lines, styles.Dimmed.Render("↑/k up  ↓/j down  r refresh  ESC back"))
	} else {
		lines = append(lines, styles.Dimmed.Render("? help  ESC back"))
	}

	content := strings.Join(lines, "\n")
	return styles.CenterWindow(content, m.width, m.height, 52)
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
