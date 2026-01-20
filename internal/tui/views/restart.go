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

type restartPhase int

const (
	restartPhaseSelect restartPhase = iota
	restartPhaseRunning
	restartPhaseComplete
	restartPhaseFailed
)

type restartOption int

const (
	restartOptRatholeServer restartOption = iota
	restartOptRatholeClient
	restartOptCaddy
	restartOptAll
)

// RestartModel is the Bubbletea model for the restart view
type RestartModel struct {
	config      *config.Config
	phase       restartPhase
	spinner     spinner.Model
	err         error
	errFriendly string
	width       int
	height      int

	// Selection menu
	menuIndex int
	options   []restartOption

	// What to restart (set from selection or CLI flags)
	restartRatholeServer bool
	restartRatholeClient bool
	restartCaddy         bool

	// Task status
	serverRatholeStatus taskStatus
	serverCaddyStatus   taskStatus
	clientRatholeStatus taskStatus
}

type restartDoneMsg struct {
	err                 error
	friendly            string
	serverRatholeDone   bool
	serverCaddyDone     bool
	clientRatholeDone   bool
	failedTask          string // which task failed
}

type restartTaskUpdateMsg struct {
	task   string // "serverRathole", "serverCaddy", "clientRathole"
	status taskStatus
}

func sendTaskUpdate(task string, status taskStatus) tea.Cmd {
	return func() tea.Msg {
		return restartTaskUpdateMsg{task: task, status: status}
	}
}

// NewRestartModel creates a new restart view model
func NewRestartModel(cfg *config.Config, server, client bool) RestartModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(styles.Primary)

	// Build options list
	options := []restartOption{
		restartOptRatholeServer,
		restartOptRatholeClient,
	}
	if cfg.Server.CaddyComposeDir != "" {
		options = append(options, restartOptCaddy)
	}
	options = append(options, restartOptAll)

	return RestartModel{
		config:  cfg,
		phase:   restartPhaseSelect,
		spinner: s,
		options: options,
		width:   80,
		height:  24,
	}
}

func (m RestartModel) optionLabel(opt restartOption) string {
	switch opt {
	case restartOptRatholeServer:
		return "Rathole Server"
	case restartOptRatholeClient:
		return "Rathole Client"
	case restartOptCaddy:
		return "Caddy"
	case restartOptAll:
		return "Restart All"
	}
	return ""
}

func (m RestartModel) optionDescription(opt restartOption) string {
	switch opt {
	case restartOptRatholeServer:
		return fmt.Sprintf("Restart rathole-server on %s", m.config.Server.Host)
	case restartOptRatholeClient:
		return fmt.Sprintf("Restart rathole-client on %s", m.config.Client.Host)
	case restartOptCaddy:
		return fmt.Sprintf("Restart Caddy (docker) on %s", m.config.Server.Host)
	case restartOptAll:
		return "Restart all services on both machines"
	}
	return ""
}

// Init initializes the model
func (m RestartModel) Init() tea.Cmd {
	return m.spinner.Tick
}

// Update handles messages
func (m RestartModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit

		case "q", "esc":
			// If in progress/complete/failed view, go back to selection
			if m.phase != restartPhaseSelect {
				m.phase = restartPhaseSelect
				m.err = nil
				m.errFriendly = ""
				m.restartRatholeServer = false
				m.restartRatholeClient = false
				m.restartCaddy = false
				m.serverRatholeStatus = taskPending
				m.serverCaddyStatus = taskPending
				m.clientRatholeStatus = taskPending
				return m, nil
			}
			// In selection phase, go back to main menu
			return m, func() tea.Msg { return GoBackMsg{} }

		case "up", "k":
			if m.phase == restartPhaseSelect && m.menuIndex > 0 {
				m.menuIndex--
			}

		case "down", "j":
			if m.phase == restartPhaseSelect && m.menuIndex < len(m.options)-1 {
				m.menuIndex++
			}

		case "enter":
			if m.phase == restartPhaseSelect {
				// Set what to restart based on selection
				selected := m.options[m.menuIndex]
				switch selected {
				case restartOptRatholeServer:
					m.restartRatholeServer = true
				case restartOptRatholeClient:
					m.restartRatholeClient = true
				case restartOptCaddy:
					m.restartCaddy = true
				case restartOptAll:
					m.restartRatholeServer = true
					m.restartRatholeClient = true
					if m.config.Server.CaddyComposeDir != "" {
						m.restartCaddy = true
					}
				}
				m.phase = restartPhaseRunning
				return m, m.startRestart()
			}
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case restartTaskUpdateMsg:
		switch msg.task {
		case "serverRathole":
			m.serverRatholeStatus = msg.status
		case "serverCaddy":
			m.serverCaddyStatus = msg.status
		case "clientRathole":
			m.clientRatholeStatus = msg.status
		}
		return m, nil

	case restartDoneMsg:
		// Update individual task statuses based on what completed
		if msg.serverRatholeDone {
			m.serverRatholeStatus = taskDone
		}
		if msg.serverCaddyDone {
			m.serverCaddyStatus = taskDone
		}
		if msg.clientRatholeDone {
			m.clientRatholeStatus = taskDone
		}

		// Mark failed task
		if msg.failedTask != "" {
			switch msg.failedTask {
			case "serverRathole":
				m.serverRatholeStatus = taskFailed
			case "serverCaddy":
				m.serverCaddyStatus = taskFailed
			case "clientRathole":
				m.clientRatholeStatus = taskFailed
			}
		}

		if msg.err != nil {
			m.phase = restartPhaseFailed
			m.err = msg.err
			m.errFriendly = msg.friendly
		} else {
			m.phase = restartPhaseComplete
		}
		return m, nil

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

// View renders the UI
func (m RestartModel) View() string {
	switch m.phase {
	case restartPhaseSelect:
		return m.renderSelectMenu()
	default:
		return m.renderProgress()
	}
}

func (m RestartModel) renderSelectMenu() string {
	var lines []string
	menuWidth := 90

	// Title
	lines = append(lines, styles.WindowTitle.Render("Restart Services"))
	lines = append(lines, "")
	lines = append(lines, styles.Dimmed.Render("Select what to restart:"))
	lines = append(lines, "")

	// Menu items
	for i, opt := range m.options {
		if i == m.menuIndex {
			// Selected item with highlight
			selectedStyle := lipgloss.NewStyle().
				Foreground(styles.White).
				Background(styles.Primary).
				Bold(true).
				Width(menuWidth)
			lines = append(lines, selectedStyle.Render("  "+m.optionLabel(opt)))
			// Description
			descStyle := lipgloss.NewStyle().
				Foreground(styles.White).
				Background(styles.Primary).
				Italic(true).
				Width(menuWidth)
			lines = append(lines, descStyle.Render("  "+m.optionDescription(opt)))
		} else {
			// Normal item
			normalStyle := lipgloss.NewStyle().
				Foreground(lipgloss.Color("#cccccc")).
				Width(menuWidth)
			lines = append(lines, normalStyle.Render("  "+m.optionLabel(opt)))
			// Description
			descStyle := lipgloss.NewStyle().
				Foreground(styles.Muted).
				Italic(true).
				Width(menuWidth)
			lines = append(lines, descStyle.Render("  "+m.optionDescription(opt)))
		}
		lines = append(lines, "")
	}

	// Help
	lines = append(lines, styles.Dimmed.Render("↑/↓: navigate  Enter: restart  ESC: back"))

	content := strings.Join(lines, "\n")

	// Wrap in fixed-size box
	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(styles.Border).
		Padding(1, 3).
		Width(100).
		Height(20)

	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, box.Render(content))
}

func (m RestartModel) renderProgress() string {
	var lines []string

	// Title
	var title string
	switch m.phase {
	case restartPhaseComplete:
		title = "Restart Complete"
	case restartPhaseFailed:
		title = "Restart Failed"
	default:
		title = "Restarting Services"
	}
	lines = append(lines, styles.WindowTitle.Render(title))
	lines = append(lines, "")

	// Server section
	if m.restartRatholeServer || m.restartCaddy {
		lines = append(lines, styles.Dimmed.Render("  Server"))
		if m.restartRatholeServer {
			lines = append(lines, m.renderTask("  Rathole server", m.serverRatholeStatus))
		}
		if m.restartCaddy {
			lines = append(lines, m.renderTask("  Caddy", m.serverCaddyStatus))
		}
		lines = append(lines, "")
	}

	// Client section
	if m.restartRatholeClient {
		lines = append(lines, styles.Dimmed.Render("  Client"))
		lines = append(lines, m.renderTask("  Rathole client", m.clientRatholeStatus))
		lines = append(lines, "")
	}

	// Error message
	if m.phase == restartPhaseFailed && m.err != nil {
		lines = append(lines, styles.Error.Render("  "+m.errFriendly))
	}

	// Success message
	if m.phase == restartPhaseComplete {
		lines = append(lines, styles.Success.Render("  Services restarted successfully"))
	}

	// Help
	lines = append(lines, "")
	lines = append(lines, styles.Dimmed.Render("ESC to go back"))

	content := strings.Join(lines, "\n")

	// Wrap in fixed-size box
	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(styles.Border).
		Padding(1, 3).
		Width(100).
		Height(20)

	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, box.Render(content))
}

func (m RestartModel) renderTask(name string, status taskStatus) string {
	var icon string
	var text string

	switch status {
	case taskDone:
		icon = styles.CheckMark()
		text = name
	case taskRunning:
		icon = m.spinner.View()
		text = name
	case taskFailed:
		icon = styles.CrossMark()
		text = styles.Error.Render(name)
	default: // taskPending
		icon = styles.Dimmed.Render("○")
		text = styles.Dimmed.Render(name)
	}

	return fmt.Sprintf("  %s %s", icon, text)
}

func (m RestartModel) startRestart() tea.Cmd {
	// Set initial running states and start the chain
	var cmds []tea.Cmd

	if m.restartRatholeServer {
		cmds = append(cmds, sendTaskUpdate("serverRathole", taskRunning))
	}
	if m.restartCaddy {
		cmds = append(cmds, sendTaskUpdate("serverCaddy", taskRunning))
	}
	if m.restartRatholeClient {
		cmds = append(cmds, sendTaskUpdate("clientRathole", taskRunning))
	}

	// Start the actual restart process
	cmds = append(cmds, m.doRestart())

	return tea.Batch(cmds...)
}

func (m RestartModel) doRestart() tea.Cmd {
	return func() tea.Msg {
		var done restartDoneMsg

		// Restart server services
		if m.restartRatholeServer || m.restartCaddy {
			client, err := ssh.GetClient(m.config.Server.Host, m.config.Server.User, m.config.Server.SSHKey)
			if err != nil {
				done.err = err
				done.friendly = fmt.Sprintf("Couldn't connect to server (%s)", m.config.Server.Host)
				if m.restartRatholeServer {
					done.failedTask = "serverRathole"
				} else {
					done.failedTask = "serverCaddy"
				}
				return done
			}
			// Don't close - connection is pooled and reused

			// Restart rathole-server
			if m.restartRatholeServer {
				if err := client.RestartService("rathole-server"); err != nil {
					done.err = err
					done.friendly = "Couldn't restart rathole-server"
					done.failedTask = "serverRathole"
					return done
				}
				// Verify service is running
				running, status, _ := client.GetServiceStatus("rathole-server")
				if !running {
					done.err = fmt.Errorf("service not running: %s", status)
					done.friendly = fmt.Sprintf("rathole-server failed to start (%s)", status)
					done.failedTask = "serverRathole"
					return done
				}
				done.serverRatholeDone = true
			}

			// Restart caddy if selected
			if m.restartCaddy {
				if err := client.RestartDockerCompose(m.config.Server.CaddyComposeDir); err != nil {
					done.err = err
					done.friendly = "Couldn't restart Caddy"
					done.failedTask = "serverCaddy"
					return done
				}
				// Verify container is running
				running, status, _ := client.GetDockerComposeStatus(m.config.Server.CaddyComposeDir)
				if !running {
					done.err = fmt.Errorf("container not running: %s", status)
					done.friendly = fmt.Sprintf("Caddy failed to start (%s)", status)
					done.failedTask = "serverCaddy"
					return done
				}
				done.serverCaddyDone = true
			}
		}

		// Restart client services
		if m.restartRatholeClient {
			client, err := ssh.GetClient(m.config.Client.Host, m.config.Client.User, m.config.Client.SSHKey)
			if err != nil {
				done.err = err
				done.friendly = fmt.Sprintf("Couldn't connect to client (%s)", m.config.Client.Host)
				done.failedTask = "clientRathole"
				return done
			}
			// Don't close - connection is pooled and reused

			if err := client.RestartService("rathole-client"); err != nil {
				done.err = err
				done.friendly = "Couldn't restart rathole-client"
				done.failedTask = "clientRathole"
				return done
			}
			// Verify service is running
			running, status, _ := client.GetServiceStatus("rathole-client")
			if !running {
				done.err = fmt.Errorf("service not running: %s", status)
				done.friendly = fmt.Sprintf("rathole-client failed to start (%s)", status)
				done.failedTask = "clientRathole"
				return done
			}
			done.clientRatholeDone = true
		}

		return done
	}
}
