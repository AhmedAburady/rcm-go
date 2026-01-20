package views

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"

	"github.com/AhmedAburady/rcm-go/internal/config"
	"github.com/AhmedAburady/rcm-go/internal/ssh"
	"github.com/AhmedAburady/rcm-go/internal/tui/styles"
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
		case "ctrl+c":
			return m, tea.Quit
		case "q", "esc":
			return m, func() tea.Msg { return GoBackMsg{} }
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
		lines = append(lines, fmt.Sprintf("  %s Checking services...", m.spinner.View()))

	case StatusStateError:
		lines = append(lines, styles.Error.Render(fmt.Sprintf("  Error: %v", m.err)))

	case StatusStateReady:
		// Server table
		lines = append(lines, m.renderMachineTable("Server", m.server))
		lines = append(lines, "")
		// Client table
		lines = append(lines, m.renderMachineTable("Client", m.client))
	}

	// Help text
	lines = append(lines, "")
	lines = append(lines, styles.Dimmed.Render("r refresh  ESC go back"))

	content := strings.Join(lines, "\n")

	// Wrap in fixed-size box
	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(styles.Border).
		Padding(1, 3).
		Width(100).
		Height(18)

	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, box.Render(content))
}

func (m StatusModel) renderMachineTable(name string, status MachineStatus) string {
	// Online/offline indicator for header
	onlineIcon := styles.CheckMark()
	onlineText := "Online"
	if !status.Online {
		onlineIcon = styles.CrossMark()
		onlineText = "Offline"
	}

	// Build rows
	var rows [][]string
	if !status.Online {
		rows = append(rows, []string{styles.CrossMark(), "Connection", "failed", "-"})
	} else {
		for _, svc := range status.Services {
			icon := styles.CheckMark()
			statusText := svc.Status
			if !svc.Running {
				icon = styles.CrossMark()
			}
			rows = append(rows, []string{icon, svc.Name, statusText, status.Host})
		}
	}

	// Create table
	t := table.New().
		Border(lipgloss.RoundedBorder()).
		BorderStyle(lipgloss.NewStyle().Foreground(styles.Border)).
		Headers("", "Service", "Status", "Host").
		Rows(rows...).
		StyleFunc(func(row, col int) lipgloss.Style {
			base := lipgloss.NewStyle().Padding(0, 1)

			if row == table.HeaderRow {
				return base.Foreground(styles.Primary).Bold(true)
			}

			switch col {
			case 0: // Icon
				return base.Width(3)
			case 1: // Service name
				return base.Foreground(lipgloss.Color("#00d7ff")).Width(20)
			case 2: // Status
				if len(rows) > row && !status.Online {
					return base.Foreground(styles.Danger)
				}
				// Check if service is running
				if row < len(status.Services) && status.Services[row].Running {
					return base.Foreground(styles.Secondary) // Green
				}
				return base.Foreground(styles.Danger) // Red
			case 3: // Host
				return base.Foreground(styles.Muted).Width(20)
			}
			return base
		})

	// Title line
	titleLine := fmt.Sprintf("%s %s (%s)", onlineIcon, lipgloss.NewStyle().Bold(true).Foreground(styles.Primary).Render(name), onlineText)

	return titleLine + "\n" + t.String()
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

	client, err := ssh.GetClient(host, user, keyPath)
	if err != nil {
		return status
	}
	// Don't close - connection is pooled and reused

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
