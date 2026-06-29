// Package ui centralizes terminal presentation: lipgloss styles, status lines,
// and the services table renderer. lipgloss auto-degrades color when stdout is
// not a terminal, so piped output stays clean.
package ui

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
	"github.com/charmbracelet/x/term"

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

func termWidth() int {
	if !term.IsTerminal(os.Stdout.Fd()) {
		return 0
	}
	w, _, err := term.GetSize(os.Stdout.Fd())
	if err != nil || w <= 0 {
		return 0
	}
	return w
}

func renderTable(headers []string, rows [][]string, style table.StyleFunc) string {
	t := table.New().
		Border(lipgloss.RoundedBorder()).
		BorderStyle(borderStyle).
		Headers(headers...).
		Rows(rows...).
		StyleFunc(style)

	s := t.String()
	if w := termWidth(); w > 0 && lipgloss.Width(s) > w {
		return t.Width(w).Wrap(true).String()
	}
	return s
}

func serviceCellStyle(r, c int) lipgloss.Style {
	if r == table.HeaderRow {
		return headerCellStyle
	}
	switch c {
	case 0:
		return cellStyle.Foreground(muted)
	case 1:
		return cellStyle.Foreground(cyan)
	case 2:
		return cellStyle.Foreground(green)
	case 3:
		return cellStyle.Foreground(yellow)
	default:
		return cellStyle.Foreground(text)
	}
}

func serviceRow(n int, s parser.Service) []string {
	domains := strings.Join(s.Domains, ", ")
	if domains == "" {
		domains = "—"
	}
	return []string{fmt.Sprintf("%d", n), s.Name, s.LocalAddr, fmt.Sprintf("%d", s.VPSPort), domains}
}

func RenderServices(services []parser.Service) string {
	rows := make([][]string, len(services))
	for i, s := range services {
		rows[i] = serviceRow(i+1, s)
	}
	return renderTable(
		[]string{"#", "SERVICE", "LOCAL ADDRESS", "VPS PORT", "DOMAINS"},
		rows, serviceCellStyle,
	)
}

type Sync int

const (
	SyncUnknown Sync = iota
	SyncOK
	SyncDrift
)

func (s Sync) mark() string {
	switch s {
	case SyncOK:
		return "✓"
	case SyncDrift:
		return "✗"
	default:
		return "?"
	}
}

func (s Sync) style() lipgloss.Style {
	st := cellStyle.Align(lipgloss.Center)
	switch s {
	case SyncOK:
		return st.Foreground(green)
	case SyncDrift:
		return st.Foreground(pink)
	default:
		return st.Foreground(muted)
	}
}

type ServiceRow struct {
	Service parser.Service
	Local   Sync
	Remote  Sync
}

func RenderServicesSync(rows []ServiceRow) string {
	tableRows := make([][]string, len(rows))
	for i, r := range rows {
		tableRows[i] = append(serviceRow(i+1, r.Service), r.Local.mark(), r.Remote.mark())
	}
	style := func(r, c int) lipgloss.Style {
		switch {
		case r == table.HeaderRow:
			return headerCellStyle
		case c == 5:
			return rows[r].Local.style()
		case c == 6:
			return rows[r].Remote.style()
		default:
			return serviceCellStyle(r, c)
		}
	}
	return renderTable(
		[]string{"#", "SERVICE", "LOCAL ADDRESS", "VPS PORT", "DOMAINS", "LOCAL", "REMOTE"},
		tableRows, style,
	)
}
