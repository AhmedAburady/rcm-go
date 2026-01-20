package styles

import "github.com/charmbracelet/lipgloss"

var (
	// Colors
	Primary   = lipgloss.Color("#7D56F4")
	Secondary = lipgloss.Color("#43BF6D")
	Danger    = lipgloss.Color("#FF5F56")
	Warning   = lipgloss.Color("#FFBD2E")
	Muted     = lipgloss.Color("#626262")
	White     = lipgloss.Color("#FAFAFA")

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

	// Dimmed text
	Dimmed = lipgloss.NewStyle().
		Foreground(Muted)
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
