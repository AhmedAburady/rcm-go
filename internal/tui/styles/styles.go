package styles

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

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

// GradientColors defines colors for gradient effects
var GradientColors = []string{
	"#FF6B6B", // Coral red
	"#FF8E53", // Orange
	"#FEC89A", // Peach
	"#98D8C8", // Mint
	"#7D56F4", // Purple (primary)
	"#A855F7", // Violet
	"#EC4899", // Pink
}

// RCMBanner is the block-style ASCII art for RCM
var RCMBanner = []string{
	"██████╗  ██████╗███╗   ███╗",
	"██╔══██╗██╔════╝████╗ ████║",
	"██████╔╝██║     ██╔████╔██║",
	"██╔══██╗██║     ██║╚██╔╝██║",
	"██║  ██║╚██████╗██║ ╚═╝ ██║",
	"╚═╝  ╚═╝ ╚═════╝╚═╝     ╚═╝",
}

// hexToRGB converts a hex color string to RGB values
func hexToRGB(hex string) (int, int, int) {
	hex = strings.TrimPrefix(hex, "#")
	var r, g, b int
	if len(hex) == 6 {
		r = hexVal(hex[0:2])
		g = hexVal(hex[2:4])
		b = hexVal(hex[4:6])
	}
	return r, g, b
}

func hexVal(s string) int {
	var val int
	for _, c := range s {
		val *= 16
		if c >= '0' && c <= '9' {
			val += int(c - '0')
		} else if c >= 'a' && c <= 'f' {
			val += int(c - 'a' + 10)
		} else if c >= 'A' && c <= 'F' {
			val += int(c - 'A' + 10)
		}
	}
	return val
}

// interpolateColor blends two colors based on t (0.0 to 1.0)
func interpolateColor(c1, c2 string, t float64) string {
	r1, g1, b1 := hexToRGB(c1)
	r2, g2, b2 := hexToRGB(c2)

	r := int(float64(r1) + t*(float64(r2)-float64(r1)))
	g := int(float64(g1) + t*(float64(g2)-float64(g1)))
	b := int(float64(b1) + t*(float64(b2)-float64(b1)))

	return sprintf("#%02x%02x%02x", r, g, b)
}

func sprintf(format string, a ...interface{}) string {
	// Simple hex formatter to avoid importing fmt
	if format == "#%02x%02x%02x" && len(a) == 3 {
		r, g, b := a[0].(int), a[1].(int), a[2].(int)
		hex := []byte{'#', 0, 0, 0, 0, 0, 0}
		hexChars := "0123456789abcdef"
		hex[1] = hexChars[r/16]
		hex[2] = hexChars[r%16]
		hex[3] = hexChars[g/16]
		hex[4] = hexChars[g%16]
		hex[5] = hexChars[b/16]
		hex[6] = hexChars[b%16]
		return string(hex)
	}
	return ""
}

// RenderGradientBanner renders the RCM banner with a horizontal gradient
func RenderGradientBanner() string {
	if len(RCMBanner) == 0 {
		return ""
	}

	// Find the max width of the banner
	maxWidth := 0
	for _, line := range RCMBanner {
		lineWidth := len([]rune(line))
		if lineWidth > maxWidth {
			maxWidth = lineWidth
		}
	}

	colors := GradientColors
	var result strings.Builder

	for i, line := range RCMBanner {
		runes := []rune(line)
		for j, r := range runes {
			// Calculate position in gradient (0.0 to 1.0)
			t := float64(j) / float64(maxWidth)

			// Find which color segment we're in
			segmentCount := len(colors) - 1
			segment := int(t * float64(segmentCount))
			if segment >= segmentCount {
				segment = segmentCount - 1
			}

			// Calculate position within segment
			segmentT := (t * float64(segmentCount)) - float64(segment)

			// Interpolate between colors
			colorHex := interpolateColor(colors[segment], colors[segment+1], segmentT)

			// Apply color to character
			style := lipgloss.NewStyle().Foreground(lipgloss.Color(colorHex))
			result.WriteString(style.Render(string(r)))
		}
		if i < len(RCMBanner)-1 {
			result.WriteString("\n")
		}
	}

	return result.String()
}
