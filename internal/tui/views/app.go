package views

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/ahmabora1/rcm/internal/config"
	"github.com/ahmabora1/rcm/internal/tui/styles"
)

// AppView represents which view is active
type AppView int

const (
	ViewMenu AppView = iota
	ViewList
	ViewSync
	ViewStatus
	ViewPull
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
	menuItems   []MenuItem
	menuIndex   int
	width       int
	height      int

	// Sub-views
	listModel   ListModel
	syncModel   SyncModel
	statusModel StatusModel
	pullModel   PullModel
}

// NewAppModel creates the main app with menu
func NewAppModel(cfg *config.Config) AppModel {
	items := []MenuItem{
		{title: "List Services", description: "View all configured services", view: ViewList},
		{title: "Sync", description: "Deploy configuration to machines", view: ViewSync},
		{title: "Status", description: "Check service health", view: ViewStatus},
		{title: "Pull", description: "Download Caddyfile from server", view: ViewPull},
	}

	return AppModel{
		config:      cfg,
		currentView: ViewMenu,
		menuItems:   items,
		menuIndex:   0,
		width:       80,
		height:      24,
	}
}

func (m AppModel) Init() tea.Cmd {
	return nil
}

func (m AppModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit

		case "q", "esc":
			// If in sub-view, go back to menu
			if m.currentView != ViewMenu {
				m.currentView = ViewMenu
				return m, nil
			}
			// If in menu, quit
			return m, tea.Quit

		case "up", "k":
			if m.currentView == ViewMenu && m.menuIndex > 0 {
				m.menuIndex--
			}

		case "down", "j":
			if m.currentView == ViewMenu && m.menuIndex < len(m.menuItems)-1 {
				m.menuIndex++
			}

		case "enter":
			if m.currentView == ViewMenu {
				selectedItem := m.menuItems[m.menuIndex]
				m.currentView = selectedItem.view
				return m, m.initSubView(selectedItem.view)
			}
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	}

	// Route updates to current view
	var cmd tea.Cmd
	switch m.currentView {
	case ViewMenu:
		// Menu doesn't need updates
	case ViewList:
		model, c := m.listModel.Update(msg)
		m.listModel = model.(ListModel)
		cmd = c
	case ViewSync:
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
	}

	return m, cmd
}

func (m AppModel) View() string {
	switch m.currentView {
	case ViewMenu:
		return m.renderMenu()
	case ViewList:
		return m.listModel.View()
	case ViewSync:
		return m.syncModel.View()
	case ViewStatus:
		return m.statusModel.View()
	case ViewPull:
		return m.pullModel.View()
	}
	return ""
}

func (m AppModel) renderMenu() string {
	var lines []string

	// Title
	lines = append(lines, styles.WindowTitle.Render("RCM"))
	lines = append(lines, styles.SubtleText.Render("Rathole Caddy Manager"))
	lines = append(lines, "")

	// Menu items
	for i, item := range m.menuItems {
		if i == m.menuIndex {
			// Selected item with highlight
			selectedStyle := lipgloss.NewStyle().
				Foreground(styles.White).
				Background(styles.Primary).
				Bold(true).
				Padding(0, 3)
			lines = append(lines, selectedStyle.Render(item.title))
		} else {
			// Normal item
			normalStyle := lipgloss.NewStyle().
				Foreground(lipgloss.Color("#cccccc")).
				Padding(0, 3)
			lines = append(lines, normalStyle.Render(item.title))
		}

		// Description below item
		descStyle := lipgloss.NewStyle().
			Foreground(styles.Muted).
			Italic(true)
		lines = append(lines, descStyle.Render(item.description))
		lines = append(lines, "")
	}

	// Help text
	lines = append(lines, styles.Dimmed.Render("↑/↓: navigate  Enter: select  q: quit"))

	content := strings.Join(lines, "\n")
	return styles.CenterWindow(content, m.width, m.height, 52)
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
	}
	return nil
}
