package views

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"

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
	stepUploading  // Concurrent: server + client
	stepRestarting // Concurrent: server + client
	stepComplete
	stepFailed
)

type taskStatus int

const (
	taskPending taskStatus = iota
	taskRunning
	taskDone
	taskFailed
)

// SyncServiceRow represents a service with local/remote status
type SyncServiceRow struct {
	Name      string
	LocalAddr string
	VPSPort   int
	Domain    string
	IsLocal   bool
	IsRemote  bool
}

// SyncModel is the Bubbletea model for the sync view
type SyncModel struct {
	config      *config.Config
	step        syncStep
	spinner     spinner.Model
	err         error
	errFriendly string
	dryRun      bool
	width       int
	height      int

	// Task status tracking
	parseStatus         taskStatus
	generateStatus      taskStatus
	uploadServerStatus  taskStatus
	uploadClientStatus  taskStatus
	restartServerStatus taskStatus
	restartClientStatus taskStatus
	restartCaddyStatus  taskStatus

	// Data passed between steps
	services    []parser.Service
	serviceRows []SyncServiceRow
	serverTOML  string
	clientTOML  string
}

type stepCompleteMsg struct {
	step        syncStep
	services    []parser.Service
	serviceRows []SyncServiceRow
	serverTOML  string
	clientTOML  string
}

type syncErrMsg struct {
	stepName string
	err      error
	friendly string // User-friendly error message
}

type uploadResultMsg struct {
	serverOK     bool
	serverFailed bool
	clientOK     bool
	clientFailed bool
	err          error
	friendly     string
}

type restartResultMsg struct {
	serverOK     bool
	serverFailed bool
	clientOK     bool
	clientFailed bool
	caddyOK      bool
	caddyFailed  bool
	err          error
	friendly     string
}

// NewSyncModel creates a new sync view model
func NewSyncModel(cfg *config.Config, dryRun bool) SyncModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(styles.Primary)

	return SyncModel{
		config:      cfg,
		step:        stepParsing,
		spinner:     s,
		dryRun:      dryRun,
		width:       80,
		height:      24,
		parseStatus: taskRunning, // Start with parsing running
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
		case "ctrl+c":
			return m, tea.Quit
		case "q", "esc":
			return m, func() tea.Msg { return GoBackMsg{} }
		case "enter", "s":
			// Start actual sync from dry run preview
			if m.dryRun && m.step == stepComplete {
				m.dryRun = false
				m.step = stepUploading
				m.uploadServerStatus = taskRunning
				m.uploadClientStatus = taskRunning
				return m, tea.Batch(m.spinner.Tick, m.runStep(stepUploading))
			}
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case stepCompleteMsg:
		// Store data from message
		if msg.services != nil {
			m.services = msg.services
		}
		if msg.serviceRows != nil {
			m.serviceRows = msg.serviceRows
		}
		if msg.serverTOML != "" {
			m.serverTOML = msg.serverTOML
		}
		if msg.clientTOML != "" {
			m.clientTOML = msg.clientTOML
		}

		// Update task status based on completed step
		switch msg.step {
		case stepParsing:
			m.parseStatus = taskDone
			m.generateStatus = taskRunning
		case stepGenerating:
			m.generateStatus = taskDone
		case stepUploading:
			m.uploadServerStatus = taskDone
			m.uploadClientStatus = taskDone
			m.restartServerStatus = taskRunning
			m.restartClientStatus = taskRunning
			if m.config.Server.CaddyComposeDir != "" {
				m.restartCaddyStatus = taskRunning
			}
		case stepRestarting:
			m.restartServerStatus = taskDone
			m.restartClientStatus = taskDone
			if m.config.Server.CaddyComposeDir != "" {
				m.restartCaddyStatus = taskDone
			}
		}

		m.step = msg.step + 1

		// For dry run, stop after generating
		if m.dryRun && m.step > stepGenerating {
			m.step = stepComplete
			return m, nil
		}

		if m.step < stepComplete {
			return m, m.runStep(m.step)
		}
		return m, nil

	case syncErrMsg:
		m.step = stepFailed
		m.err = msg.err
		m.errFriendly = msg.friendly

		switch msg.stepName {
		case "Parse":
			m.parseStatus = taskFailed
		case "Generate":
			m.generateStatus = taskFailed
		}
		return m, nil

	case uploadResultMsg:
		m.step = stepFailed
		m.err = msg.err
		m.errFriendly = msg.friendly

		if msg.serverOK {
			m.uploadServerStatus = taskDone
		} else if msg.serverFailed {
			m.uploadServerStatus = taskFailed
		}

		if msg.clientOK {
			m.uploadClientStatus = taskDone
		} else if msg.clientFailed {
			m.uploadClientStatus = taskFailed
		}
		return m, nil

	case restartResultMsg:
		m.step = stepFailed
		m.err = msg.err
		m.errFriendly = msg.friendly

		// Mark based on actual results
		if msg.serverOK {
			m.restartServerStatus = taskDone
		} else if msg.serverFailed {
			m.restartServerStatus = taskFailed
		}

		if msg.clientOK {
			m.restartClientStatus = taskDone
		} else if msg.clientFailed {
			m.restartClientStatus = taskFailed
		}

		if msg.caddyOK {
			m.restartCaddyStatus = taskDone
		} else if msg.caddyFailed {
			m.restartCaddyStatus = taskFailed
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
func (m SyncModel) View() string {
	var content string

	if m.dryRun && m.step >= stepComplete {
		// Dry run complete - show preview
		content = m.renderDryRunView()
	} else {
		// Show sync progress in a box
		content = m.renderSyncBox()
	}

	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, content)
}

func (m SyncModel) renderSyncBox() string {
	var lines []string

	// Title
	var title string
	switch m.step {
	case stepComplete:
		title = "Sync Complete"
	case stepFailed:
		title = "Sync Failed"
	default:
		title = "Sync"
	}
	lines = append(lines, styles.WindowTitle.Render(title))
	lines = append(lines, "")

	// Progress tasks
	lines = append(lines, m.renderTask("Parse Caddyfile", m.parseStatus))
	lines = append(lines, m.renderTask("Generate configs", m.generateStatus))
	lines = append(lines, "")
	lines = append(lines, styles.Dimmed.Render("  Deploy"))
	lines = append(lines, m.renderTask("  Upload server", m.uploadServerStatus))
	lines = append(lines, m.renderTask("  Upload client", m.uploadClientStatus))
	lines = append(lines, "")
	lines = append(lines, styles.Dimmed.Render("  Restart"))
	lines = append(lines, m.renderTask("  Rathole server", m.restartServerStatus))
	lines = append(lines, m.renderTask("  Rathole client", m.restartClientStatus))
	if m.config.Server.CaddyComposeDir != "" {
		lines = append(lines, m.renderTask("  Caddy", m.restartCaddyStatus))
	}

	// Error message if failed
	if m.step == stepFailed && m.err != nil {
		lines = append(lines, "")
		lines = append(lines, m.renderErrorBox())
	}

	// Success message
	if m.step == stepComplete {
		lines = append(lines, "")
		lines = append(lines, styles.Success.Render(fmt.Sprintf("  ✓ Deployed %d services", len(m.services))))
	}

	// Help
	lines = append(lines, "")
	lines = append(lines, styles.Dimmed.Render("ESC go back"))

	content := strings.Join(lines, "\n")

	// Wrap in fixed-size box so it doesn't jump around
	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(styles.Border).
		Padding(1, 3).
		Width(100).
		Height(20)

	return box.Render(content)
}

func (m SyncModel) renderTask(name string, status taskStatus) string {
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

func (m SyncModel) renderErrorBox() string {
	// Simple error message without nested box
	friendlyMsg := styles.Error.Render("  " + m.errFriendly)

	// Add hint based on error type
	hint := ""
	errStr := m.err.Error()
	if strings.Contains(errStr, "sudo") && strings.Contains(errStr, "password") {
		hint = "\n" + styles.Dimmed.Render("  Hint: Use passwordless sudo or root")
	} else if strings.Contains(errStr, "could not be found") || strings.Contains(errStr, "not found") {
		hint = "\n" + styles.Dimmed.Render("  Hint: Service may not be installed")
	}

	return friendlyMsg + hint
}

func (m SyncModel) renderDryRunView() string {
	var lines []string

	// Title
	lines = append(lines, styles.WindowTitle.Render("Sync Preview"))
	lines = append(lines, "")

	// Services table
	lines = append(lines, m.renderSyncTable())
	lines = append(lines, "")

	// Summary
	newCount := 0
	updateCount := 0
	for _, svc := range m.serviceRows {
		if svc.IsLocal && !svc.IsRemote {
			newCount++
		} else if svc.IsLocal && svc.IsRemote {
			updateCount++
		}
	}

	green := lipgloss.NewStyle().Foreground(lipgloss.Color("#00ff87"))
	yellow := lipgloss.NewStyle().Foreground(lipgloss.Color("#ffff00"))

	if newCount > 0 {
		lines = append(lines, fmt.Sprintf("  %s %d new", green.Render("●"), newCount))
	}
	if updateCount > 0 {
		lines = append(lines, fmt.Sprintf("  %s %d update", yellow.Render("●"), updateCount))
	}

	lines = append(lines, "")
	lines = append(lines, styles.Dimmed.Render(fmt.Sprintf("  Server: %s", m.config.Server.Host)))
	lines = append(lines, styles.Dimmed.Render(fmt.Sprintf("  Client: %s", m.config.Client.Host)))
	lines = append(lines, "")

	// Sync button - one full button
	btnLine := lipgloss.NewStyle().
		Background(styles.Primary).
		Foreground(styles.White).
		Bold(true).
		Padding(0, 2).
		Render("SYNC NOW")
	// Pad left to center the whole thing
	padding := (60 - lipgloss.Width(btnLine)) / 2
	if padding > 0 {
		btnLine = strings.Repeat(" ", padding) + btnLine
	}
	lines = append(lines, btnLine)
	lines = append(lines, "")
	escText := styles.Dimmed.Render("ESC cancel")
	escPadding := (60 - lipgloss.Width(escText)) / 2
	if escPadding > 0 {
		escText = strings.Repeat(" ", escPadding) + escText
	}
	lines = append(lines, escText)

	content := strings.Join(lines, "\n")

	// Wrap in box like other views
	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(styles.Border).
		Padding(1, 3)

	return box.Render(content)
}

func (m SyncModel) renderSyncTable() string {
	rows := make([][]string, len(m.serviceRows))
	for i, svc := range m.serviceRows {
		localStatus := styles.CrossMark()
		if svc.IsLocal {
			localStatus = styles.CheckMark()
		}
		remoteStatus := styles.CrossMark()
		if svc.IsRemote {
			remoteStatus = styles.CheckMark()
		}

		rows[i] = []string{
			svc.Name,
			svc.LocalAddr,
			fmt.Sprintf("%d", svc.VPSPort),
			localStatus,
			remoteStatus,
		}
	}

	t := table.New().
		Border(lipgloss.RoundedBorder()).
		BorderStyle(lipgloss.NewStyle().Foreground(styles.Border)).
		Headers("Service", "Local Address", "Port", "Local", "Remote").
		Rows(rows...).
		StyleFunc(func(row, col int) lipgloss.Style {
			base := lipgloss.NewStyle().Padding(0, 1)

			if row == table.HeaderRow {
				return base.Foreground(styles.Primary).Bold(true)
			}

			switch col {
			case 0:
				return base.Foreground(lipgloss.Color("#00d7ff"))
			case 1:
				return base.Foreground(lipgloss.Color("#00ff87"))
			case 2:
				return base.Foreground(lipgloss.Color("#ffff00"))
			case 3, 4:
				return base.Align(lipgloss.Center)
			}
			return base.Foreground(lipgloss.Color("#cccccc"))
		})

	return t.String()
}

// expandTilde expands ~ to home directory
func expandTilde(path string) string {
	if len(path) > 0 && path[0] == '~' {
		home, _ := os.UserHomeDir()
		return home + path[1:]
	}
	return path
}

// runStep executes the current sync step
func (m *SyncModel) runStep(step syncStep) tea.Cmd {
	return func() tea.Msg {
		switch step {
		case stepParsing:
			m.parseStatus = taskRunning

			// Fetch local and remote concurrently
			type parseResult struct {
				services map[string]parser.Service
				isLocal  bool
				err      error
			}

			resultCh := make(chan parseResult, 2)

			// Local
			go func() {
				localServices := make(map[string]parser.Service)
				services, err := parser.ParseFile(m.config.Paths.Caddyfile)
				if err == nil {
					for _, svc := range services {
						localServices[svc.Name] = svc
					}
				}
				resultCh <- parseResult{services: localServices, isLocal: true, err: err}
			}()

			// Remote
			go func() {
				remoteServices := make(map[string]parser.Service)
				if m.config.Server.Host != "" && m.config.Server.Caddyfile != "" {
					client, err := ssh.GetClient(m.config.Server.Host, m.config.Server.User, m.config.Server.SSHKey)
					if err == nil {
						// Don't close - connection is pooled and reused
						content, err := client.DownloadFile(m.config.Server.Caddyfile)
						if err == nil {
							services, _ := parser.ParseContent(content)
							for _, svc := range services {
								remoteServices[svc.Name] = svc
							}
						}
					}
				}
				resultCh <- parseResult{services: remoteServices, isLocal: false}
			}()

			// Collect results
			var localServices, remoteServices map[string]parser.Service
			var localErr error
			for i := 0; i < 2; i++ {
				result := <-resultCh
				if result.isLocal {
					localServices = result.services
					localErr = result.err
				} else {
					remoteServices = result.services
				}
			}

			if localErr != nil {
				return syncErrMsg{stepName: "Parse", err: localErr, friendly: "Couldn't parse local Caddyfile"}
			}

			// Build service rows
			var serviceRows []SyncServiceRow
			var services []parser.Service
			for name, svc := range localServices {
				_, isRemote := remoteServices[name]
				domain := ""
				if len(svc.Domains) > 0 {
					domain = svc.Domains[0]
				}
				serviceRows = append(serviceRows, SyncServiceRow{
					Name:      svc.Name,
					LocalAddr: svc.LocalAddr,
					VPSPort:   svc.VPSPort,
					Domain:    domain,
					IsLocal:   true,
					IsRemote:  isRemote,
				})
				services = append(services, svc)
			}

			sort.Slice(serviceRows, func(i, j int) bool {
				return serviceRows[i].Name < serviceRows[j].Name
			})

			return stepCompleteMsg{step: step, services: services, serviceRows: serviceRows}

		case stepGenerating:
			serverTOML, err := generator.GenerateServerTOML(m.config, m.services)
			if err != nil {
				return syncErrMsg{stepName: "Generate", err: err, friendly: "Couldn't generate server config"}
			}

			clientTOML, err := generator.GenerateClientTOML(m.config, m.services)
			if err != nil {
				return syncErrMsg{stepName: "Generate", err: err, friendly: "Couldn't generate client config"}
			}
			return stepCompleteMsg{step: step, serverTOML: serverTOML, clientTOML: clientTOML}

		case stepUploading:
			// Upload to server AND client concurrently
			type taskResult struct {
				name     string
				err      error
				friendly string
			}
			resultCh := make(chan taskResult, 2)

			// Upload to server (rathole config + Caddyfile)
			go func() {
				client, err := ssh.GetClient(m.config.Server.Host, m.config.Server.User, m.config.Server.SSHKey)
				if err != nil {
					resultCh <- taskResult{name: "server", err: err, friendly: fmt.Sprintf("Couldn't connect to server (%s)", m.config.Server.Host)}
					return
				}
				// Don't close - connection is pooled and reused

				// Upload rathole server config
				if err := client.UploadContent(m.serverTOML, m.config.Server.RatholeConfig); err != nil {
					resultCh <- taskResult{name: "server", err: err, friendly: "Couldn't upload rathole config to server"}
					return
				}

				// Upload Caddyfile to server
				if m.config.Server.Caddyfile != "" {
					localCaddyPath := expandTilde(m.config.Paths.Caddyfile)
					caddyContent, err := os.ReadFile(localCaddyPath)
					if err != nil {
						resultCh <- taskResult{name: "server", err: err, friendly: "Couldn't read local Caddyfile"}
						return
					}
					if err := client.UploadContent(string(caddyContent), m.config.Server.Caddyfile); err != nil {
						resultCh <- taskResult{name: "server", err: err, friendly: "Couldn't upload Caddyfile to server"}
						return
					}
				}

				resultCh <- taskResult{name: "server", err: nil}
			}()

			// Upload to client
			go func() {
				client, err := ssh.GetClient(m.config.Client.Host, m.config.Client.User, m.config.Client.SSHKey)
				if err != nil {
					resultCh <- taskResult{name: "client", err: err, friendly: fmt.Sprintf("Couldn't connect to client (%s)", m.config.Client.Host)}
					return
				}
				// Don't close - connection is pooled and reused

				if err := client.UploadContent(m.clientTOML, m.config.Client.RatholeConfig); err != nil {
					resultCh <- taskResult{name: "client", err: err, friendly: "Couldn't upload config to client"}
					return
				}
				resultCh <- taskResult{name: "client", err: nil}
			}()

			// Collect both results
			var firstErr *taskResult
			serverOK, serverFailed := false, false
			clientOK, clientFailed := false, false

			for i := 0; i < 2; i++ {
				r := <-resultCh
				if r.name == "server" {
					if r.err == nil {
						serverOK = true
					} else {
						serverFailed = true
						if firstErr == nil {
							firstErr = &r
						}
					}
				} else {
					if r.err == nil {
						clientOK = true
					} else {
						clientFailed = true
						if firstErr == nil {
							firstErr = &r
						}
					}
				}
			}

			if firstErr != nil {
				return uploadResultMsg{
					serverOK:     serverOK,
					serverFailed: serverFailed,
					clientOK:     clientOK,
					clientFailed: clientFailed,
					err:          firstErr.err,
					friendly:     firstErr.friendly,
				}
			}

			return stepCompleteMsg{step: step}

		case stepRestarting:
			// Restart server, client, and caddy concurrently
			type taskResult struct {
				name     string
				err      error
				friendly string
			}

			taskCount := 2
			if m.config.Server.CaddyComposeDir != "" {
				taskCount = 3
			}
			resultCh := make(chan taskResult, taskCount)

			// Restart rathole-server
			go func() {
				client, err := ssh.GetClient(m.config.Server.Host, m.config.Server.User, m.config.Server.SSHKey)
				if err != nil {
					resultCh <- taskResult{name: "server", err: err, friendly: fmt.Sprintf("Couldn't connect to server (%s)", m.config.Server.Host)}
					return
				}
				// Don't close - connection is pooled and reused

				if err := client.RestartService("rathole-server"); err != nil {
					resultCh <- taskResult{name: "server", err: err, friendly: "Couldn't restart rathole on server"}
					return
				}
				resultCh <- taskResult{name: "server", err: nil}
			}()

			// Restart rathole-client
			go func() {
				client, err := ssh.GetClient(m.config.Client.Host, m.config.Client.User, m.config.Client.SSHKey)
				if err != nil {
					resultCh <- taskResult{name: "client", err: err, friendly: fmt.Sprintf("Couldn't connect to client (%s)", m.config.Client.Host)}
					return
				}
				// Don't close - connection is pooled and reused

				if err := client.RestartService("rathole-client"); err != nil {
					resultCh <- taskResult{name: "client", err: err, friendly: "Couldn't restart rathole on client"}
					return
				}
				resultCh <- taskResult{name: "client", err: nil}
			}()

			// Restart Caddy (if configured)
			if m.config.Server.CaddyComposeDir != "" {
				go func() {
					client, err := ssh.GetClient(m.config.Server.Host, m.config.Server.User, m.config.Server.SSHKey)
					if err != nil {
						resultCh <- taskResult{name: "caddy", err: err, friendly: fmt.Sprintf("Couldn't connect to server (%s)", m.config.Server.Host)}
						return
					}
					// Don't close - connection is pooled and reused

					if err := client.RestartDockerCompose(m.config.Server.CaddyComposeDir); err != nil {
						resultCh <- taskResult{name: "caddy", err: err, friendly: "Couldn't restart Caddy"}
						return
					}
					resultCh <- taskResult{name: "caddy", err: nil}
				}()
			}

			// Collect all results
			var firstErr *taskResult
			serverOK, serverFailed := false, false
			clientOK, clientFailed := false, false
			caddyOK, caddyFailed := false, false

			for i := 0; i < taskCount; i++ {
				r := <-resultCh
				switch r.name {
				case "server":
					if r.err == nil {
						serverOK = true
					} else {
						serverFailed = true
						if firstErr == nil {
							firstErr = &r
						}
					}
				case "client":
					if r.err == nil {
						clientOK = true
					} else {
						clientFailed = true
						if firstErr == nil {
							firstErr = &r
						}
					}
				case "caddy":
					if r.err == nil {
						caddyOK = true
					} else {
						caddyFailed = true
						if firstErr == nil {
							firstErr = &r
						}
					}
				}
			}

			if firstErr != nil {
				return restartResultMsg{
					serverOK:     serverOK,
					serverFailed: serverFailed,
					clientOK:     clientOK,
					clientFailed: clientFailed,
					caddyOK:      caddyOK,
					caddyFailed:  caddyFailed,
					err:          firstErr.err,
					friendly:     firstErr.friendly,
				}
			}

			return stepCompleteMsg{step: step}
		}

		return stepCompleteMsg{step: step}
	}
}
