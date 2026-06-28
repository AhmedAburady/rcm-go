// Package ui centralizes terminal presentation: lipgloss styles, status lines,
// and the services table renderer. lipgloss auto-degrades color when stdout is
// not a terminal, so piped output stays clean.
package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"

	"github.com/AhmedAburady/rcm-go/internal/parser"
)

var (
	accent = lipgloss.Color("#7D56F4")
	green  = lipgloss.Color("#00ff87")
	cyan   = lipgloss.Color("#00d7ff")
	yellow = lipgloss.Color("#ffff00")
	pink   = lipgloss.Color("#FF79C6")
	muted  = lipgloss.Color("#626262")
	text   = lipgloss.Color("#cccccc")

	headingStyle = lipgloss.NewStyle().Bold(true).Foreground(accent)
	accentStyle  = lipgloss.NewStyle().Foreground(accent)
	successStyle = lipgloss.NewStyle().Foreground(green)
	warnStyle    = lipgloss.NewStyle().Foreground(yellow)
	errorStyle   = lipgloss.NewStyle().Foreground(pink)
	mutedStyle   = lipgloss.NewStyle().Foreground(muted)

	borderStyle     = lipgloss.NewStyle().Foreground(muted)
	headerCellStyle = lipgloss.NewStyle().Bold(true).Foreground(accent).Padding(0, 1)
	cellStyle       = lipgloss.NewStyle().Padding(0, 1)
)

// Heading renders a bold accented title.
func Heading(format string, a ...any) string { return headingStyle.Render(fmt.Sprintf(format, a...)) }

// Step renders an in-progress action line ("▸ …").
func Step(format string, a ...any) string {
	return accentStyle.Render("▸ ") + fmt.Sprintf(format, a...)
}

// OK renders a success line ("✓ …").
func OK(format string, a ...any) string {
	return successStyle.Render("✓ ") + fmt.Sprintf(format, a...)
}

// Fail renders a failure line ("✗ …").
func Fail(format string, a ...any) string {
	return errorStyle.Render("✗ ") + fmt.Sprintf(format, a...)
}

// Warn renders a cautionary line ("! …").
func Warn(format string, a ...any) string {
	return warnStyle.Render("! ") + fmt.Sprintf(format, a...)
}

// Info renders a muted, secondary line.
func Info(format string, a ...any) string { return mutedStyle.Render(fmt.Sprintf(format, a...)) }

// Check returns a styled checkmark; Cross returns a styled cross.
func Check() string { return successStyle.Render("✓") }
func Cross() string { return errorStyle.Render("✗") }

// RenderServices renders parsed services as a bordered, color-coded table:
// service, local address, VPS port, and domains.
func RenderServices(services []parser.Service) string {
	rows := make([][]string, len(services))
	for i, s := range services {
		domains := strings.Join(s.Domains, ", ")
		if domains == "" {
			domains = "—"
		}
		rows[i] = []string{s.Name, s.LocalAddr, fmt.Sprintf("%d", s.VPSPort), domains}
	}

	t := table.New().
		Border(lipgloss.RoundedBorder()).
		BorderStyle(borderStyle).
		Headers("SERVICE", "LOCAL ADDRESS", "VPS PORT", "DOMAINS").
		Rows(rows...).
		StyleFunc(func(r, c int) lipgloss.Style {
			if r == table.HeaderRow {
				return headerCellStyle
			}
			switch c {
			case 0:
				return cellStyle.Foreground(cyan)
			case 1:
				return cellStyle.Foreground(green)
			case 2:
				return cellStyle.Foreground(yellow)
			default:
				return cellStyle.Foreground(text)
			}
		})
	return t.String()
}
