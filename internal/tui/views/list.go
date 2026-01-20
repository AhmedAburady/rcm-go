package views

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"

	"github.com/ahmabora1/rcm/internal/config"
	"github.com/ahmabora1/rcm/internal/parser"
	"github.com/ahmabora1/rcm/internal/ssh"
	"github.com/ahmabora1/rcm/internal/tui/styles"
)

// ListState represents the view state
type ListState int

const (
	ListStateLoading ListState = iota
	ListStateReady
	ListStateError
)

// ServiceRow represents a merged service from local and remote
type ServiceRow struct {
	Name      string
	LocalAddr string
	VPSPort   int
	Domains   []string
	IsLocal   bool
	IsRemote  bool
}

// ListModel is the Bubbletea model for the list view
type ListModel struct {
	state       ListState
	config      *config.Config
	services    []ServiceRow
	spinner     spinner.Model
	err         error
	width       int
	height      int
	showHelp    bool
	selectedIdx int
}

type servicesLoadedMsg struct {
	services []ServiceRow
}

type listErrMsg struct {
	err error
}

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

func (m ListModel) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		m.loadServicesCmd(),
	)
}

func (m ListModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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

func (m ListModel) View() string {
	var lines []string

	// Title with count
	switch m.state {
	case ListStateLoading:
		lines = append(lines, styles.WindowTitle.Render("Services"))
		lines = append(lines, "")
		lines = append(lines, fmt.Sprintf("%s Loading...", m.spinner.View()))

	case ListStateError:
		lines = append(lines, styles.WindowTitle.Render("Services"))
		lines = append(lines, "")
		lines = append(lines, styles.Error.Render(fmt.Sprintf("Error: %v", m.err)))

	case ListStateReady:
		if len(m.services) == 0 {
			lines = append(lines, styles.WindowTitle.Render("Services"))
			lines = append(lines, "")
			lines = append(lines, styles.Dimmed.Render("No services found"))
		} else {
			title := fmt.Sprintf("Services (%d found)", len(m.services))
			lines = append(lines, styles.WindowTitle.Render(title))
			lines = append(lines, "")
			lines = append(lines, m.renderTable())
		}
	}

	// Help
	lines = append(lines, "")
	if m.showHelp {
		lines = append(lines, styles.Dimmed.Render("↑/↓ navigate  r refresh  ESC back"))
	} else {
		lines = append(lines, styles.Dimmed.Render("? help  ESC back"))
	}

	content := strings.Join(lines, "\n")
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, content)
}

func (m ListModel) renderTable() string {
	// Build rows
	rows := make([][]string, len(m.services))
	for i, svc := range m.services {
		// Get domains - show all joined by comma
		domains := strings.Join(svc.Domains, ", ")

		// Status checkmarks
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
			domains,
			localStatus,
			remoteStatus,
		}
	}

	selectedIdx := m.selectedIdx

	// Create table with lipgloss/table
	t := table.New().
		Border(lipgloss.RoundedBorder()).
		BorderStyle(lipgloss.NewStyle().Foreground(styles.Border)).
		Headers("Service", "Local Address", "VPS Port", "Domains", "Local", "Remote").
		Rows(rows...).
		StyleFunc(func(row, col int) lipgloss.Style {
			// Base style with padding
			base := lipgloss.NewStyle().Padding(0, 1)

			// Header row
			if row == table.HeaderRow {
				return base.
					Foreground(styles.Primary).
					Bold(true)
			}
			// Selected data row
			if row == selectedIdx {
				return base.
					Background(styles.Primary).
					Foreground(styles.White).
					Bold(true)
			}
			// Normal row - color code by column
			switch col {
			case 0: // Service name
				return base.Foreground(lipgloss.Color("#00d7ff")) // cyan
			case 1: // Local Address
				return base.Foreground(lipgloss.Color("#00ff87")) // green
			case 2: // VPS Port
				return base.Foreground(lipgloss.Color("#ffff00")) // yellow
			case 3: // Domains
				return base.Foreground(lipgloss.Color("#87afff")) // blue
			case 4, 5: // Local/Remote
				return base.Foreground(lipgloss.Color("#ffffff")).Align(lipgloss.Center)
			}
			return base.Foreground(lipgloss.Color("#cccccc"))
		})

	return t.String()
}

func (m ListModel) loadServicesCmd() tea.Cmd {
	return func() tea.Msg {
		// Fetch local services
		localServices := make(map[string]parser.Service)
		if m.config.Paths.Caddyfile != "" {
			services, err := parser.ParseFile(m.config.Paths.Caddyfile)
			if err == nil {
				for _, svc := range services {
					localServices[svc.Name] = svc
				}
			}
		}

		// Fetch remote services
		remoteServices := make(map[string]parser.Service)
		if m.config.Server.Host != "" && m.config.Server.Caddyfile != "" {
			client, err := ssh.GetClient(m.config.Server.Host, m.config.Server.User, m.config.Server.SSHKey)
			if err == nil {
				// Don't close - connection is pooled and reused
				content, err := client.DownloadFile(m.config.Server.Caddyfile)
				if err == nil {
					services, err := parser.ParseContent(content)
					if err == nil {
						for _, svc := range services {
							remoteServices[svc.Name] = svc
						}
					}
				}
			}
		}

		// Merge all service names
		allNames := make(map[string]bool)
		for name := range localServices {
			allNames[name] = true
		}
		for name := range remoteServices {
			allNames[name] = true
		}

		if len(allNames) == 0 {
			return listErrMsg{err: fmt.Errorf("no services found")}
		}

		// Build merged service list
		var result []ServiceRow
		for name := range allNames {
			localSvc, isLocal := localServices[name]
			remoteSvc, isRemote := remoteServices[name]

			// Use whichever exists for data
			var svc parser.Service
			if isLocal {
				svc = localSvc
			} else {
				svc = remoteSvc
			}

			result = append(result, ServiceRow{
				Name:      svc.Name,
				LocalAddr: svc.LocalAddr,
				VPSPort:   svc.VPSPort,
				Domains:   svc.Domains,
				IsLocal:   isLocal,
				IsRemote:  isRemote,
			})
		}

		// Sort by name
		sort.Slice(result, func(i, j int) bool {
			return result[i].Name < result[j].Name
		})

		return servicesLoadedMsg{services: result}
	}
}
