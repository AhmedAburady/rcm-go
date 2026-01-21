package views

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/AhmedAburady/rcm-go/internal/config"
	"github.com/AhmedAburady/rcm-go/internal/tui/styles"
)

// AppView represents which view is active
type AppView int

const (
	ViewMenu AppView = iota
	ViewList
	ViewSync
	ViewSyncDryRun
	ViewStatus
	ViewPull
	ViewRestart
)

// MenuItem represents a menu option
type MenuItem struct {
	title       string
	description string
	view        AppView
}

// AppModel is the main application model with navigation
type AppModel struct {
	config      *config.Config
	currentView AppView
	initialView AppView
	menuItems   []MenuItem
	menuIndex   int
	width       int
	height      int

	// Sub-views
	listModel    ListModel
	syncModel    SyncModel
	statusModel  StatusModel
	pullModel    PullModel
	restartModel RestartModel
}

// NewAppModel creates the main app with menu
func NewAppModel(cfg *config.Config) AppModel {
	return NewAppModelWithView(cfg, ViewMenu)
}

// NewAppModelWithView creates the main app starting at a specific view
func NewAppModelWithView(cfg *config.Config, initialView AppView) AppModel {
	items := []MenuItem{
		{title: "List Services", description: "View all configured services", view: ViewList},
		{title: "Sync", description: "Deploy configuration to machines", view: ViewSync},
		{title: "Sync (Dry Run)", description: "Preview sync without deploying", view: ViewSyncDryRun},
		{title: "Status", description: "Check service health", view: ViewStatus},
		{title: "Restart", description: "Restart rathole and caddy services", view: ViewRestart},
		{title: "Pull", description: "Download Caddyfile from server", view: ViewPull},
		{title: "Exit", description: "Quit RCM", view: ViewMenu}, // Special: exit
	}

	m := AppModel{
		config:      cfg,
		currentView: initialView,
		menuItems:   items,
		menuIndex:   0,
		width:       80,
		height:      24,
		initialView: initialView,
	}

	// Pre-initialize the subview if starting with a non-menu view
	switch initialView {
	case ViewList:
		m.listModel = NewListModel(cfg)
	case ViewSync:
		m.syncModel = NewSyncModel(cfg, false)
	case ViewSyncDryRun:
		m.syncModel = NewSyncModel(cfg, true)
	case ViewStatus:
		m.statusModel = NewStatusModel(cfg)
	case ViewPull:
		m.pullModel = NewPullModel(cfg)
	case ViewRestart:
		m.restartModel = NewRestartModel(cfg, true, true)
	}

	return m
}

func (m AppModel) Init() tea.Cmd {
	// If starting with a specific view, return its Init command
	switch m.initialView {
	case ViewList:
		return m.listModel.Init()
	case ViewSync, ViewSyncDryRun:
		return m.syncModel.Init()
	case ViewStatus:
		return m.statusModel.Init()
	case ViewPull:
		return m.pullModel.Init()
	case ViewRestart:
		return m.restartModel.Init()
	}
	return nil
}

// GoBackMsg signals that a subview wants to return to the main menu
type GoBackMsg struct{}

func (m AppModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Always capture window size for the app model
	if wsMsg, ok := msg.(tea.WindowSizeMsg); ok {
		m.width = wsMsg.Width
		m.height = wsMsg.Height
	}

	// Check for go back message from subviews
	if _, ok := msg.(GoBackMsg); ok {
		m.currentView = ViewMenu
		return m, nil
	}

	// Route updates to current subview FIRST (except for ctrl+c)
	if m.currentView != ViewMenu {
		if keyMsg, ok := msg.(tea.KeyMsg); ok && keyMsg.String() == "ctrl+c" {
			return m, tea.Quit
		}

		var cmd tea.Cmd

		switch m.currentView {
		case ViewList:
			model, c := m.listModel.Update(msg)
			m.listModel = model.(ListModel)
			cmd = c
		case ViewSync, ViewSyncDryRun:
			model, c := m.syncModel.Update(msg)
			m.syncModel = model.(SyncModel)
			cmd = c
		case ViewStatus:
			model, c := m.statusModel.Update(msg)
			m.statusModel = model.(StatusModel)
			cmd = c
		case ViewPull:
			model, c := m.pullModel.Update(msg)
			m.pullModel = model.(PullModel)
			cmd = c
		case ViewRestart:
			model, c := m.restartModel.Update(msg)
			m.restartModel = model.(RestartModel)
			cmd = c
		}

		return m, cmd
	}

	// Handle main menu
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit

		case "q", "esc":
			return m, tea.Quit

		case "up", "k":
			if m.menuIndex > 0 {
				m.menuIndex--
			}

		case "down", "j":
			if m.menuIndex < len(m.menuItems)-1 {
				m.menuIndex++
			}

		case "enter":
			selectedItem := m.menuItems[m.menuIndex]
			// Exit item is special
			if selectedItem.title == "Exit" {
				return m, tea.Quit
			}
			m.currentView = selectedItem.view
			return m, m.initSubView(selectedItem.view)
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	}

	return m, nil
}

func (m AppModel) View() string {
	switch m.currentView {
	case ViewMenu:
		return m.renderMenu()
	case ViewList:
		return m.listModel.View()
	case ViewSync, ViewSyncDryRun:
		return m.syncModel.View()
	case ViewStatus:
		return m.statusModel.View()
	case ViewPull:
		return m.pullModel.View()
	case ViewRestart:
		return m.restartModel.View()
	}
	return ""
}

func (m AppModel) renderMenu() string {
	var lines []string
	menuWidth := 90 // Inner content width

	// Gradient banner
	lines = append(lines, styles.RenderGradientBanner())
	lines = append(lines, styles.SubtleText.Render("Rathole Caddy Manager"))
	lines = append(lines, "")

	// Menu items
	for i, item := range m.menuItems {
		if i == m.menuIndex {
			// Selected item with highlight - full width
			selectedStyle := lipgloss.NewStyle().
				Foreground(styles.White).
				Background(styles.Primary).
				Bold(true).
				Width(menuWidth)
			lines = append(lines, selectedStyle.Render("  "+item.title))
			// Description for selected - same highlight
			descStyle := lipgloss.NewStyle().
				Foreground(styles.White).
				Background(styles.Primary).
				Italic(true).
				Width(menuWidth)
			lines = append(lines, descStyle.Render("  "+item.description))
		} else {
			// Normal item
			normalStyle := lipgloss.NewStyle().
				Foreground(lipgloss.Color("#cccccc")).
				Width(menuWidth)
			lines = append(lines, normalStyle.Render("  "+item.title))
			// Description
			descStyle := lipgloss.NewStyle().
				Foreground(styles.Muted).
				Italic(true).
				Width(menuWidth)
			lines = append(lines, descStyle.Render("  "+item.description))
		}
		lines = append(lines, "")
	}

	// Help text
	lines = append(lines, styles.Dimmed.Render("↑/↓: navigate  Enter: select  q: quit"))

	content := strings.Join(lines, "\n")

	// Wrap in fixed-size box
	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(styles.Border).
		Padding(1, 3).
		Width(100).
		Height(27)

	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, box.Render(content))
}

func (m *AppModel) initSubView(view AppView) tea.Cmd {
	switch view {
	case ViewList:
		m.listModel = NewListModel(m.config)
		m.listModel.width = m.width
		m.listModel.height = m.height
		return m.listModel.Init()
	case ViewSync:
		m.syncModel = NewSyncModel(m.config, false)
		m.syncModel.width = m.width
		m.syncModel.height = m.height
		return m.syncModel.Init()
	case ViewSyncDryRun:
		m.syncModel = NewSyncModel(m.config, true)
		m.syncModel.width = m.width
		m.syncModel.height = m.height
		return m.syncModel.Init()
	case ViewStatus:
		m.statusModel = NewStatusModel(m.config)
		m.statusModel.width = m.width
		m.statusModel.height = m.height
		return m.statusModel.Init()
	case ViewPull:
		m.pullModel = NewPullModel(m.config)
		m.pullModel.width = m.width
		m.pullModel.height = m.height
		return m.pullModel.Init()
	case ViewRestart:
		m.restartModel = NewRestartModel(m.config, true, true) // Restart both by default
		m.restartModel.width = m.width
		m.restartModel.height = m.height
		return m.restartModel.Init()
	}
	return nil
}
