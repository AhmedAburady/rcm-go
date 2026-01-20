package views

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"

	"github.com/AhmedAburady/rcm-go/internal/config"
	"github.com/AhmedAburady/rcm-go/internal/parser"
	"github.com/AhmedAburady/rcm-go/internal/ssh"
	"github.com/AhmedAburady/rcm-go/internal/tui/styles"
)

type pullStep int

const (
	pullStepConfirm pullStep = iota
	pullStepConnecting
	pullStepDownloading
	pullStepParsing
	pullStepSaving
	pullStepComplete
	pullStepFailed
)

// PullModel is the Bubbletea model for the pull view
type PullModel struct {
	config  *config.Config
	step    pullStep
	spinner spinner.Model
	logs    []string
	err     error
	width   int
	height  int

	// Downloaded content
	remoteCaddyfile string
	services        []parser.Service
	localExists     bool
}

type pullStepCompleteMsg struct {
	step            pullStep
	log             string
	remoteCaddyfile string           // Content from download step
	services        []parser.Service // Parsed services
}

type pullErrMsg struct {
	err error
}

// NewPullModel creates a new pull view model
func NewPullModel(cfg *config.Config) PullModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(styles.Primary)

	// Check if local file exists
	localExists := false
	if _, err := os.Stat(cfg.Paths.Caddyfile); err == nil {
		localExists = true
	}

	// Start at confirm step if local file exists, otherwise skip to connecting
	startStep := pullStepConnecting
	if localExists {
		startStep = pullStepConfirm
	}

	return PullModel{
		config:      cfg,
		step:        startStep,
		spinner:     s,
		logs:        []string{},
		localExists: localExists,
		width:       80,
		height:      24,
	}
}

// Init initializes the model
func (m PullModel) Init() tea.Cmd {
	// If on confirm step, just start spinner (wait for user input)
	if m.step == pullStepConfirm {
		return m.spinner.Tick
	}
	// Otherwise start the pull operation
	return tea.Batch(
		m.spinner.Tick,
		m.runStep(pullStepConnecting),
	)
}

// Update handles messages
func (m PullModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "q", "esc":
			return m, func() tea.Msg { return GoBackMsg{} }
		case "y", "Y":
			// Confirm overwrite and start pull
			if m.step == pullStepConfirm {
				m.step = pullStepConnecting
				return m, m.runStep(pullStepConnecting)
			}
		case "n", "N":
			// Cancel on confirm step
			if m.step == pullStepConfirm {
				return m, func() tea.Msg { return GoBackMsg{} }
			}
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case pullStepCompleteMsg:
		m.logs = append(m.logs, msg.log)
		m.step = msg.step + 1

		// Store data from message onto model
		if msg.remoteCaddyfile != "" {
			m.remoteCaddyfile = msg.remoteCaddyfile
		}
		if msg.services != nil {
			m.services = msg.services
		}

		if m.step < pullStepComplete {
			return m, m.runStep(m.step)
		}
		return m, nil

	case pullErrMsg:
		m.step = pullStepFailed
		m.err = msg.err
		return m, nil

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

// View renders the UI
func (m PullModel) View() string {
	var lines []string

	// Title
	switch m.step {
	case pullStepConfirm:
		lines = append(lines, styles.WindowTitle.Render("Confirm Overwrite"))
	case pullStepComplete:
		lines = append(lines, styles.WindowTitle.Render("Pull Complete"))
	case pullStepFailed:
		lines = append(lines, styles.Error.Render("Pull Failed"))
	default:
		lines = append(lines, styles.WindowTitle.Render("Pull Caddyfile"))
	}
	lines = append(lines, "")

	// Confirmation prompt
	if m.step == pullStepConfirm {
		lines = append(lines, styles.WarningText.Render("  Local Caddyfile already exists:"))
		lines = append(lines, "")
		lines = append(lines, styles.Dimmed.Render("  "+m.config.Paths.Caddyfile))
		lines = append(lines, "")
		lines = append(lines, "  Pulling from server will overwrite this file.")
		lines = append(lines, "")
		lines = append(lines, "")
		lines = append(lines, fmt.Sprintf("  Press %s to overwrite, %s to cancel",
			styles.KeyStyle.Render("y"),
			styles.KeyStyle.Render("n")))
	} else {
		// Progress table (only show when not on confirm step)
		lines = append(lines, m.renderProgressTable())
	}

	// Services table after completion
	if m.step == pullStepComplete && len(m.services) > 0 {
		lines = append(lines, "")
		lines = append(lines, m.renderServicesTable())
		lines = append(lines, "")
		lines = append(lines, styles.Success.Render(fmt.Sprintf("  Saved to %s", m.config.Paths.Caddyfile)))
	}

	// Error message if failed
	if m.step == pullStepFailed && m.err != nil {
		lines = append(lines, "")
		lines = append(lines, m.renderErrorBox())
	}

	// Help
	lines = append(lines, "")
	lines = append(lines, styles.Dimmed.Render("ESC go back"))

	content := strings.Join(lines, "\n")

	// Wrap in fixed-size box
	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(styles.Border).
		Padding(1, 3).
		Width(100).
		Height(22)

	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, box.Render(content))
}

func (m PullModel) renderProgressTable() string {
	steps := []struct {
		step pullStep
		name string
	}{
		{pullStepConnecting, "Connect to server"},
		{pullStepDownloading, "Download Caddyfile"},
		{pullStepParsing, "Parse services"},
		{pullStepSaving, "Save locally"},
	}

	var rows [][]string
	for _, s := range steps {
		var icon string
		var status string

		if m.step > s.step {
			icon = styles.CheckMark()
			status = "Done"
		} else if m.step == s.step && m.step < pullStepComplete {
			icon = m.spinner.View()
			status = "Running"
		} else if m.step == pullStepFailed && s.step == m.step-1 {
			icon = styles.CrossMark()
			status = "Failed"
		} else {
			icon = styles.Dimmed.Render("○")
			status = "Pending"
		}

		rows = append(rows, []string{icon, s.name, status})
	}

	t := table.New().
		Border(lipgloss.RoundedBorder()).
		BorderStyle(lipgloss.NewStyle().Foreground(styles.Border)).
		Headers("", "Step", "Status").
		Rows(rows...).
		StyleFunc(func(row, col int) lipgloss.Style {
			base := lipgloss.NewStyle().Padding(0, 1)

			if row == table.HeaderRow {
				return base.Foreground(styles.Primary).Bold(true)
			}

			switch col {
			case 0: // Icon
				return base.Width(3)
			case 1: // Step name
				return base.Foreground(lipgloss.Color("#00d7ff")).Width(25)
			case 2: // Status
				if row < len(rows) {
					switch rows[row][2] {
					case "Done":
						return base.Foreground(styles.Secondary)
					case "Failed":
						return base.Foreground(styles.Danger)
					case "Running":
						return base.Foreground(styles.Warning)
					}
				}
				return base.Foreground(styles.Muted)
			}
			return base
		})

	return t.String()
}

func (m PullModel) renderServicesTable() string {
	var rows [][]string
	for _, svc := range m.services {
		domain := ""
		if len(svc.Domains) > 0 {
			domain = svc.Domains[0]
		}
		rows = append(rows, []string{svc.Name, svc.LocalAddr, fmt.Sprintf("%d", svc.VPSPort), domain})
	}

	t := table.New().
		Border(lipgloss.RoundedBorder()).
		BorderStyle(lipgloss.NewStyle().Foreground(styles.Border)).
		Headers("Service", "Local Address", "Port", "Domain").
		Rows(rows...).
		StyleFunc(func(row, col int) lipgloss.Style {
			base := lipgloss.NewStyle().Padding(0, 1)

			if row == table.HeaderRow {
				return base.Foreground(styles.Primary).Bold(true)
			}

			switch col {
			case 0: // Service name
				return base.Foreground(lipgloss.Color("#00d7ff"))
			case 1: // Local address
				return base.Foreground(lipgloss.Color("#00ff87"))
			case 2: // Port
				return base.Foreground(lipgloss.Color("#ffff00"))
			case 3: // Domain
				return base.Foreground(styles.Muted)
			}
			return base
		})

	return t.String()
}

func (m PullModel) renderErrorBox() string {
	errMsg := m.err.Error()
	friendly := "Pull failed"

	if strings.Contains(errMsg, "No such file") {
		friendly = "Caddyfile not found on server"
	} else if strings.Contains(errMsg, "connect") {
		friendly = "Couldn't connect to server"
	}

	return styles.Error.Render("  " + friendly)
}

func expandTildePull(path string) string {
	if len(path) > 0 && path[0] == '~' {
		home, _ := os.UserHomeDir()
		return home + path[1:]
	}
	return path
}

// runStep executes the current pull step
func (m *PullModel) runStep(step pullStep) tea.Cmd {
	return func() tea.Msg {
		switch step {
		case pullStepConnecting:
			// Connect and download in one step to reduce SSH connections
			client, err := ssh.GetClient(m.config.Server.Host, m.config.Server.User, m.config.Server.SSHKey)
			if err != nil {
				return pullErrMsg{err: fmt.Errorf("connect to server: %w", err)}
			}
			// Don't close - connection is pooled and reused

			content, err := client.DownloadFile(m.config.Server.Caddyfile)
			if err != nil {
				return pullErrMsg{err: fmt.Errorf("download Caddyfile: %w", err)}
			}
			// Skip directly to downloading complete (we did both steps)
			return pullStepCompleteMsg{
				step:            pullStepDownloading, // Report as downloading complete
				log:             fmt.Sprintf("Downloaded from %s (%d bytes)", m.config.Server.Host, len(content)),
				remoteCaddyfile: content,
			}

		case pullStepDownloading:
			// This step is now handled by pullStepConnecting
			return pullStepCompleteMsg{step: step, log: "Already downloaded"}

		case pullStepParsing:
			services, err := parser.ParseContent(m.remoteCaddyfile)
			if err != nil {
				return pullErrMsg{err: fmt.Errorf("parse Caddyfile: %w", err)}
			}
			// Pass services through message
			return pullStepCompleteMsg{
				step:     step,
				log:      fmt.Sprintf("Found %d services", len(services)),
				services: services,
			}

		case pullStepSaving:
			// Expand tilde in path
			savePath := expandTildePull(m.config.Paths.Caddyfile)

			// Ensure directory exists
			dir := filepath.Dir(savePath)
			if err := os.MkdirAll(dir, 0755); err != nil {
				return pullErrMsg{err: fmt.Errorf("create config dir: %w", err)}
			}

			// Write the file
			if err := os.WriteFile(savePath, []byte(m.remoteCaddyfile), 0644); err != nil {
				return pullErrMsg{err: fmt.Errorf("save Caddyfile: %w", err)}
			}

			return pullStepCompleteMsg{step: step, log: "Saved"}
		}

		return pullStepCompleteMsg{step: step, log: "Step completed"}
	}
}
