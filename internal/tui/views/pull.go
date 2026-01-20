package views

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/ahmabora1/rcm/internal/config"
	"github.com/ahmabora1/rcm/internal/parser"
	"github.com/ahmabora1/rcm/internal/ssh"
	"github.com/ahmabora1/rcm/internal/tui/styles"
)

type pullStep int

const (
	pullStepConnecting pullStep = iota
	pullStepDownloading
	pullStepParsing
	pullStepSaving
	pullStepComplete
	pullStepFailed
)

var pullStepNames = []string{
	"Connecting to server...",
	"Downloading Caddyfile...",
	"Parsing services...",
	"Saving locally...",
	"Complete!",
	"Failed",
}

// PullModel is the Bubbletea model for the pull view
type PullModel struct {
	config   *config.Config
	step     pullStep
	spinner  spinner.Model
	logs     []string
	err      error
	width    int
	height   int

	// Downloaded content
	remoteCaddyfile string
	services        []parser.Service
	localExists     bool
}

type pullStepCompleteMsg struct {
	step pullStep
	log  string
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

	return PullModel{
		config:      cfg,
		step:        pullStepConnecting,
		spinner:     s,
		logs:        []string{},
		localExists: localExists,
		width:       80,
		height:      24,
	}
}

// Init initializes the model
func (m PullModel) Init() tea.Cmd {
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
		case "ctrl+c", "q":
			return m, tea.Quit
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case pullStepCompleteMsg:
		m.logs = append(m.logs, msg.log)
		m.step = msg.step + 1

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
	lines = append(lines, styles.WindowTitle.Render("Pull Caddyfile"))
	lines = append(lines, "")

	// Current step
	if m.step < pullStepComplete {
		lines = append(lines, fmt.Sprintf("%s %s", m.spinner.View(), pullStepNames[m.step]))
	} else if m.step == pullStepComplete {
		lines = append(lines, styles.Success.Render("✓ "+pullStepNames[m.step]))
	} else {
		lines = append(lines, styles.Error.Render(fmt.Sprintf("✗ Error: %v", m.err)))
	}

	// Logs
	if len(m.logs) > 0 {
		lines = append(lines, "")
		for _, log := range m.logs {
			lines = append(lines, fmt.Sprintf("%s %s", styles.CheckMark(), log))
		}
	}

	// Show services summary after completion
	if m.step == pullStepComplete && len(m.services) > 0 {
		lines = append(lines, "")
		lines = append(lines, styles.SubtleText.Render("Discovered services:"))
		for _, svc := range m.services {
			lines = append(lines, fmt.Sprintf("  • %s → %s", svc.Name, svc.LocalAddr))
		}
	}

	// Help text
	lines = append(lines, "")
	lines = append(lines, styles.Dimmed.Render("ESC to go back"))

	content := strings.Join(lines, "\n")
	return styles.CenterWindow(content, m.width, m.height, 52)
}

// runStep executes the current pull step
func (m *PullModel) runStep(step pullStep) tea.Cmd {
	return func() tea.Msg {
		switch step {
		case pullStepConnecting:
			client, err := ssh.NewClient(m.config.Server.Host, m.config.Server.User, m.config.Server.SSHKey)
			if err != nil {
				return pullErrMsg{err: fmt.Errorf("connect to server: %w", err)}
			}
			client.Close()
			return pullStepCompleteMsg{step: step, log: fmt.Sprintf("Connected to %s", m.config.Server.Host)}

		case pullStepDownloading:
			client, err := ssh.NewClient(m.config.Server.Host, m.config.Server.User, m.config.Server.SSHKey)
			if err != nil {
				return pullErrMsg{err: fmt.Errorf("connect to server: %w", err)}
			}
			defer client.Close()

			content, err := client.DownloadFile(m.config.Server.Caddyfile)
			if err != nil {
				return pullErrMsg{err: fmt.Errorf("download Caddyfile: %w", err)}
			}
			m.remoteCaddyfile = content
			return pullStepCompleteMsg{step: step, log: fmt.Sprintf("Downloaded Caddyfile (%d bytes)", len(content))}

		case pullStepParsing:
			services, err := parser.ParseContent(m.remoteCaddyfile)
			if err != nil {
				return pullErrMsg{err: fmt.Errorf("parse Caddyfile: %w", err)}
			}
			m.services = services
			return pullStepCompleteMsg{step: step, log: fmt.Sprintf("Found %d services", len(services))}

		case pullStepSaving:
			// Ensure directory exists
			dir := config.ExpandPath("~/.config/rcm")
			if err := os.MkdirAll(dir, 0755); err != nil {
				return pullErrMsg{err: fmt.Errorf("create config dir: %w", err)}
			}

			// Write the file
			if err := os.WriteFile(m.config.Paths.Caddyfile, []byte(m.remoteCaddyfile), 0644); err != nil {
				return pullErrMsg{err: fmt.Errorf("save Caddyfile: %w", err)}
			}

			msg := fmt.Sprintf("Saved to %s", m.config.Paths.Caddyfile)
			if m.localExists {
				msg += " (overwritten)"
			}
			return pullStepCompleteMsg{step: step, log: msg}
		}

		return pullStepCompleteMsg{step: step, log: "Step completed"}
	}
}
