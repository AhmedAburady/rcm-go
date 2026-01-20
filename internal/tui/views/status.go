package views

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/ahmabora1/rcm/internal/config"
	"github.com/ahmabora1/rcm/internal/ssh"
	"github.com/ahmabora1/rcm/internal/tui/styles"
)

// StatusState represents the view state
type StatusState int

const (
	StatusStateLoading StatusState = iota
	StatusStateReady
	StatusStateError
)

// ServiceHealth holds health information for a service
type ServiceHealth struct {
	Name    string
	Running bool
	Status  string
}

// MachineStatus holds status for a machine
type MachineStatus struct {
	Host     string
	Online   bool
	Services []ServiceHealth
}

// StatusModel is the Bubbletea model for the status view
type StatusModel struct {
	state    StatusState
	config   *config.Config
	server   MachineStatus
	client   MachineStatus
	spinner  spinner.Model
	err      error
	width    int
	height   int
	showHelp bool
}

type statusLoadedMsg struct {
	server MachineStatus
	client MachineStatus
}

type statusErrMsg struct {
	err error
}

// NewStatusModel creates a new status view model
func NewStatusModel(cfg *config.Config) StatusModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(styles.Primary)

	return StatusModel{
		state:   StatusStateLoading,
		config:  cfg,
		spinner: s,
		width:   80,
		height:  24,
	}
}

// Init initializes the model
func (m StatusModel) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		m.loadStatusCmd(),
	)
}

// Update handles messages
func (m StatusModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
			m.state = StatusStateLoading
			return m, tea.Batch(m.spinner.Tick, m.loadStatusCmd())
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case statusLoadedMsg:
		m.state = StatusStateReady
		m.server = msg.server
		m.client = msg.client
		return m, nil

	case statusErrMsg:
		m.state = StatusStateError
		m.err = msg.err
		return m, nil

	case spinner.TickMsg:
		m.spinner, cmd = m.spinner.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

// View renders the UI
func (m StatusModel) View() string {
	var lines []string

	// Title
	lines = append(lines, styles.WindowTitle.Render("Service Status"))
	lines = append(lines, "")

	switch m.state {
	case StatusStateLoading:
		lines = append(lines, fmt.Sprintf("%s Checking services...", m.spinner.View()))

	case StatusStateError:
		lines = append(lines, styles.Error.Render(fmt.Sprintf("Error: %v", m.err)))

	case StatusStateReady:
		// Server status
		lines = append(lines, m.renderMachineStatus("Server", m.server))
		// Client status
		lines = append(lines, m.renderMachineStatus("Client", m.client))
	}

	// Help text
	lines = append(lines, "")
	if m.showHelp {
		lines = append(lines, styles.Dimmed.Render("ESC: back  r: refresh  ?: help"))
	} else {
		lines = append(lines, styles.Dimmed.Render("? for help  ESC to go back"))
	}

	content := strings.Join(lines, "\n")
	return styles.CenterWindow(content, m.width, m.height, 52)
}

func (m StatusModel) renderMachineStatus(name string, status MachineStatus) string {
	var s strings.Builder

	// Machine header
	hostStyle := styles.Info.Bold(true)
	onlineIcon := styles.CheckMark()
	if !status.Online {
		onlineIcon = styles.CrossMark()
	}

	s.WriteString(fmt.Sprintf("%s %s (%s)\n", onlineIcon, hostStyle.Render(name), status.Host))

	if !status.Online {
		s.WriteString(styles.Dimmed.Render("  Unable to connect\n"))
		return s.String()
	}

	// Services
	for _, svc := range status.Services {
		icon := styles.CheckMark()
		statusText := styles.Success.Render(svc.Status)
		if !svc.Running {
			icon = styles.CrossMark()
			statusText = styles.Error.Render(svc.Status)
		}
		s.WriteString(fmt.Sprintf("  %s %-18s %s\n", icon, svc.Name, statusText))
	}

	return s.String()
}

// loadStatusCmd creates a command to load status
func (m StatusModel) loadStatusCmd() tea.Cmd {
	return func() tea.Msg {
		serverStatus := m.checkMachine(
			m.config.Server.Host,
			m.config.Server.User,
			m.config.Server.SSHKey,
			[]string{"rathole-server"},
			m.config.Server.CaddyComposeDir,
		)

		clientStatus := m.checkMachine(
			m.config.Client.Host,
			m.config.Client.User,
			m.config.Client.SSHKey,
			[]string{"rathole-client"},
			"",
		)

		return statusLoadedMsg{
			server: serverStatus,
			client: clientStatus,
		}
	}
}

func (m StatusModel) checkMachine(host, user, keyPath string, services []string, composeDir string) MachineStatus {
	status := MachineStatus{
		Host:     host,
		Online:   false,
		Services: []ServiceHealth{},
	}

	client, err := ssh.NewClient(host, user, keyPath)
	if err != nil {
		return status
	}
	defer client.Close()

	status.Online = true

	// Check systemd services
	for _, svc := range services {
		running, statusText, _ := client.GetServiceStatus(svc)
		if statusText == "" {
			statusText = "unknown"
		}
		status.Services = append(status.Services, ServiceHealth{
			Name:    svc,
			Running: running,
			Status:  statusText,
		})
	}

	// Check docker compose if configured
	if composeDir != "" {
		running, _, _ := client.GetDockerComposeStatus(composeDir)
		statusText := "stopped"
		if running {
			statusText = "running"
		}
		status.Services = append(status.Services, ServiceHealth{
			Name:    "caddy (docker)",
			Running: running,
			Status:  statusText,
		})
	}

	return status
}
