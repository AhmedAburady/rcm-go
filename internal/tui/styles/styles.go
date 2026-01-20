package styles

import "github.com/charmbracelet/lipgloss"

var (
	// Colors
	Primary    = lipgloss.Color("#7D56F4")
	Secondary  = lipgloss.Color("#43BF6D")
	Danger     = lipgloss.Color("#FF79C6")
	Warning    = lipgloss.Color("#FFBD2E")
	Muted      = lipgloss.Color("#626262")
	White      = lipgloss.Color("#FAFAFA")
	Background = lipgloss.Color("#1a1a2e")
	Border     = lipgloss.Color("#4a4a6a")

	// Adaptive colors (dark/light mode)
	Subtle = lipgloss.AdaptiveColor{Light: "#D9DCCF", Dark: "#383838"}

	// Text styles
	Title = lipgloss.NewStyle().
		Bold(true).
		Foreground(Primary).
		MarginBottom(1)

	Subtitle = lipgloss.NewStyle().
			Foreground(Muted).
			Italic(true)

	// Status indicators
	StatusOK = lipgloss.NewStyle().
			Foreground(Secondary)

	StatusError = lipgloss.NewStyle().
			Foreground(Danger)

	StatusPending = lipgloss.NewStyle().
			Foreground(Warning)

	// Box styles
	Box = lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(Primary).
		Padding(1, 2)

	// Window box - main container with rounded borders
	WindowBox = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(Border).
			Padding(2, 4)

	// Header style inside window
	WindowTitle = lipgloss.NewStyle().
			Bold(true).
			Foreground(Primary).
			Align(lipgloss.Center).
			MarginBottom(1)

	// Selected item
	Selected = lipgloss.NewStyle().
			Foreground(White).
			Background(Primary).
			Bold(true).
			Padding(0, 1)

	// Help bar
	HelpBar = lipgloss.NewStyle().
		Foreground(Muted).
		MarginTop(1)

	// Help bar centered
	HelpBarCentered = lipgloss.NewStyle().
			Foreground(Muted).
			Align(lipgloss.Center)

	// Error message
	Error = lipgloss.NewStyle().
		Foreground(Danger).
		Bold(true)

	// Success message
	Success = lipgloss.NewStyle().
		Foreground(Secondary).
		Bold(true)

	// Info message
	Info = lipgloss.NewStyle().
		Foreground(Primary)

	// Warning message
	WarningText = lipgloss.NewStyle().
			Foreground(Warning).
			Bold(true)

	// Dimmed text
	Dimmed = lipgloss.NewStyle().
		Foreground(Muted)

	// SubtleText is a styled text for secondary info
	SubtleText = lipgloss.NewStyle().
			Foreground(Muted).
			Italic(true)

	// MenuItem styles
	MenuItem = lipgloss.NewStyle().
			Padding(0, 2)

	MenuItemSelected = lipgloss.NewStyle().
				Foreground(White).
				Background(Primary).
				Bold(true).
				Padding(0, 2)

	MenuItemDesc = lipgloss.NewStyle().
			Foreground(Muted).
			PaddingLeft(4)

	// KeyStyle for highlighting keyboard shortcuts
	KeyStyle = lipgloss.NewStyle().
			Foreground(Warning).
			Bold(true)
)

// StatusIcon returns the appropriate status icon
func StatusIcon(ok bool) string {
	if ok {
		return StatusOK.Render("●")
	}
	return StatusError.Render("●")
}

// PendingIcon returns a pending status icon
func PendingIcon() string {
	return StatusPending.Render("●")
}

// CheckMark returns a styled checkmark
func CheckMark() string {
	return StatusOK.Render("✓")
}

// CrossMark returns a styled cross
func CrossMark() string {
	return StatusError.Render("✗")
}

// CenterWindow creates a centered window box with given content and dimensions
func CenterWindow(content string, width, height, boxWidth int) string {
	// Create the box with centered content
	box := WindowBox.
		Width(boxWidth).
		Align(lipgloss.Center).
		Render(content)

	// Use lipgloss.Place for proper centering
	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, box)
}

// PlaceCenter centers content both horizontally and vertically
func PlaceCenter(width, height int, content string) string {
	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, content)
}
