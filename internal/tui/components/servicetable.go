package components

import (
	"fmt"

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/lipgloss"

	"github.com/ahmabora1/rcm/internal/parser"
	"github.com/ahmabora1/rcm/internal/tui/styles"
)

// ServiceStatus holds the status of a service
type ServiceStatus struct {
	Local  bool
	Remote bool
}

// NewServiceTable creates a table for displaying services
func NewServiceTable(services []parser.Service, statuses map[string]ServiceStatus, height int) table.Model {
	columns := []table.Column{
		{Title: "Service", Width: 15},
		{Title: "Local Address", Width: 22},
		{Title: "VPS Port", Width: 10},
		{Title: "Domain", Width: 28},
		{Title: "Local", Width: 7},
		{Title: "Remote", Width: 7},
	}

	rows := make([]table.Row, len(services))
	for i, s := range services {
		localStatus := styles.CheckMark()
		remoteStatus := styles.CheckMark()

		if status, ok := statuses[s.Name]; ok {
			if !status.Local {
				localStatus = styles.CrossMark()
			}
			if !status.Remote {
				remoteStatus = styles.CrossMark()
			}
		}

		rows[i] = table.Row{
			s.Name,
			s.LocalAddr,
			fmt.Sprintf("%d", s.VPSPort),
			truncate(s.PrimaryDomain(), 26),
			localStatus,
			remoteStatus,
		}
	}

	t := table.New(
		table.WithColumns(columns),
		table.WithRows(rows),
		table.WithFocused(true),
		table.WithHeight(height),
	)

	// Apply styles
	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(styles.Primary).
		BorderBottom(true).
		Bold(true).
		Foreground(styles.Primary)

	s.Selected = s.Selected.
		Foreground(lipgloss.Color("229")).
		Background(styles.Primary).
		Bold(true)

	t.SetStyles(s)

	return t
}

// NewSimpleServiceTable creates a simpler table without status columns
func NewSimpleServiceTable(services []parser.Service, height int) table.Model {
	columns := []table.Column{
		{Title: "Service", Width: 18},
		{Title: "Local Address", Width: 24},
		{Title: "VPS Port", Width: 10},
		{Title: "Domain", Width: 35},
	}

	rows := make([]table.Row, len(services))
	for i, s := range services {
		rows[i] = table.Row{
			s.Name,
			s.LocalAddr,
			fmt.Sprintf("%d", s.VPSPort),
			truncate(s.PrimaryDomain(), 33),
		}
	}

	t := table.New(
		table.WithColumns(columns),
		table.WithRows(rows),
		table.WithFocused(true),
		table.WithHeight(height),
	)

	// Apply styles
	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(styles.Primary).
		BorderBottom(true).
		Bold(true).
		Foreground(styles.Primary)

	s.Selected = s.Selected.
		Foreground(lipgloss.Color("229")).
		Background(styles.Primary).
		Bold(true)

	t.SetStyles(s)

	return t
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}
