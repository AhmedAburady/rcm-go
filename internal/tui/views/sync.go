package views

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/ahmabora1/rcm/internal/config"
	"github.com/ahmabora1/rcm/internal/generator"
	"github.com/ahmabora1/rcm/internal/parser"
	"github.com/ahmabora1/rcm/internal/ssh"
	"github.com/ahmabora1/rcm/internal/tui/styles"
)

type syncStep int

const (
	stepParsing syncStep = iota
	stepGenerating
	stepUploadingServer
	stepUploadingClient
	stepRestartingServer
	stepRestartingClient
	stepComplete
	stepFailed
)

var stepNames = []string{
	"Parsing Caddyfile...",
	"Generating configs...",
	"Uploading to server...",
	"Uploading to client...",
	"Restarting server services...",
	"Restarting client services...",
	"Complete!",
	"Failed",
}

// SyncModel is the Bubbletea model for the sync view
type SyncModel struct {
	config   *config.Config
	step     syncStep
	spinner  spinner.Model
	progress progress.Model
	logs     []string
	err      error
	dryRun   bool
	width    int
	height   int

	// Data passed between steps
	services   []parser.Service
	serverTOML string
	clientTOML string
}

type stepCompleteMsg struct {
	step syncStep
	log  string
}

type syncErrMsg struct {
	err error
}

// NewSyncModel creates a new sync view model
func NewSyncModel(cfg *config.Config, dryRun bool) SyncModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(styles.Primary)

	p := progress.New(
		progress.WithDefaultGradient(),
		progress.WithWidth(50),
	)

	return SyncModel{
		config:   cfg,
		step:     stepParsing,
		spinner:  s,
		progress: p,
		logs:     []string{},
		dryRun:   dryRun,
	}
}

// Init initializes the model
func (m SyncModel) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		m.runStep(stepParsing),
	)
}

// Update handles messages
func (m SyncModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.progress.Width = m.width - 10
		if m.progress.Width > 60 {
			m.progress.Width = 60
		}

	case stepCompleteMsg:
		m.logs = append(m.logs, msg.log)
		m.step = msg.step + 1

		if m.step < stepComplete {
			return m, m.runStep(m.step)
		}
		return m, nil

	case syncErrMsg:
		m.step = stepFailed
		m.err = msg.err
		return m, nil

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		cmds = append(cmds, cmd)

	case progress.FrameMsg:
		progressModel, cmd := m.progress.Update(msg)
		m.progress = progressModel.(progress.Model)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

// View renders the UI
func (m SyncModel) View() string {
	var s strings.Builder

	s.WriteString("\n")
	title := "  RCM Sync"
	if m.dryRun {
		title += " (dry-run)"
	}
	s.WriteString(styles.Title.Render(title))
	s.WriteString("\n\n")

	// Progress bar
	percent := float64(m.step) / float64(stepComplete)
	s.WriteString("  ")
	s.WriteString(m.progress.ViewAs(percent))
	s.WriteString("\n\n")

	// Current step
	if m.step < stepComplete {
		s.WriteString(fmt.Sprintf("  %s %s\n", m.spinner.View(), stepNames[m.step]))
	} else if m.step == stepComplete {
		s.WriteString(styles.Success.Render("  ✓ " + stepNames[m.step]))
		s.WriteString("\n")
	} else {
		s.WriteString(styles.Error.Render(fmt.Sprintf("  ✗ Error: %v", m.err)))
		s.WriteString("\n")
	}

	// Logs
	s.WriteString("\n")
	for _, log := range m.logs {
		s.WriteString(fmt.Sprintf("  %s %s\n", styles.CheckMark(), log))
	}

	s.WriteString("\n")
	s.WriteString(styles.HelpBar.Render("  Press q to quit"))

	return s.String()
}

// runStep executes the current sync step
func (m *SyncModel) runStep(step syncStep) tea.Cmd {
	return func() tea.Msg {
		switch step {
		case stepParsing:
			services, err := parser.ParseFile(m.config.Paths.Caddyfile)
			if err != nil {
				return syncErrMsg{err: fmt.Errorf("parse caddyfile: %w", err)}
			}
			m.services = services
			return stepCompleteMsg{step: step, log: fmt.Sprintf("Parsed %d services from Caddyfile", len(services))}

		case stepGenerating:
			serverTOML, err := generator.GenerateServerTOML(m.config, m.services)
			if err != nil {
				return syncErrMsg{err: fmt.Errorf("generate server.toml: %w", err)}
			}
			m.serverTOML = serverTOML

			clientTOML, err := generator.GenerateClientTOML(m.config, m.services)
			if err != nil {
				return syncErrMsg{err: fmt.Errorf("generate client.toml: %w", err)}
			}
			m.clientTOML = clientTOML

			if m.dryRun {
				return stepCompleteMsg{step: step, log: "Generated configs (dry-run, not saved)"}
			}
			return stepCompleteMsg{step: step, log: "Generated server.toml and client.toml"}

		case stepUploadingServer:
			if m.dryRun {
				return stepCompleteMsg{step: step, log: "Would upload to server (dry-run)"}
			}

			client, err := ssh.NewClient(m.config.Server.Host, m.config.Server.User, m.config.Server.SSHKey)
			if err != nil {
				return syncErrMsg{err: fmt.Errorf("connect to server: %w", err)}
			}
			defer client.Close()

			if err := client.UploadContent(m.serverTOML, m.config.Server.RatholeConfig); err != nil {
				return syncErrMsg{err: fmt.Errorf("upload server.toml: %w", err)}
			}

			return stepCompleteMsg{step: step, log: fmt.Sprintf("Uploaded server.toml to %s", m.config.Server.Host)}

		case stepUploadingClient:
			if m.dryRun {
				return stepCompleteMsg{step: step, log: "Would upload to client (dry-run)"}
			}

			client, err := ssh.NewClient(m.config.Client.Host, m.config.Client.User, m.config.Client.SSHKey)
			if err != nil {
				return syncErrMsg{err: fmt.Errorf("connect to client: %w", err)}
			}
			defer client.Close()

			if err := client.UploadContent(m.clientTOML, m.config.Client.RatholeConfig); err != nil {
				return syncErrMsg{err: fmt.Errorf("upload client.toml: %w", err)}
			}

			return stepCompleteMsg{step: step, log: fmt.Sprintf("Uploaded client.toml to %s", m.config.Client.Host)}

		case stepRestartingServer:
			if m.dryRun {
				return stepCompleteMsg{step: step, log: "Would restart server services (dry-run)"}
			}

			client, err := ssh.NewClient(m.config.Server.Host, m.config.Server.User, m.config.Server.SSHKey)
			if err != nil {
				return syncErrMsg{err: fmt.Errorf("connect to server: %w", err)}
			}
			defer client.Close()

			if err := client.RestartService("rathole-server"); err != nil {
				return syncErrMsg{err: fmt.Errorf("restart rathole-server: %w", err)}
			}

			if m.config.Server.CaddyComposeDir != "" {
				if err := client.RestartDockerCompose(m.config.Server.CaddyComposeDir); err != nil {
					// Non-fatal, just log
				}
			}

			return stepCompleteMsg{step: step, log: "Restarted rathole-server"}

		case stepRestartingClient:
			if m.dryRun {
				return stepCompleteMsg{step: step, log: "Would restart client services (dry-run)"}
			}

			client, err := ssh.NewClient(m.config.Client.Host, m.config.Client.User, m.config.Client.SSHKey)
			if err != nil {
				return syncErrMsg{err: fmt.Errorf("connect to client: %w", err)}
			}
			defer client.Close()

			if err := client.RestartService("rathole-client"); err != nil {
				return syncErrMsg{err: fmt.Errorf("restart rathole-client: %w", err)}
			}

			return stepCompleteMsg{step: step, log: "Restarted rathole-client"}
		}

		return stepCompleteMsg{step: step, log: "Step completed"}
	}
}
