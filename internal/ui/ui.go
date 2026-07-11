// Package ui centralizes terminal presentation: lipgloss styles, status lines,
// and the services table renderer. lipgloss auto-degrades color when stdout is
// not a terminal, so piped output stays clean.
package ui

import (
	"fmt"
	"net"
	"net/url"
	"os"
	"slices"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
	"github.com/charmbracelet/x/term"
	"golang.org/x/net/publicsuffix"

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

const noDomain = "(no domain)"

func rootDomain(domain string) string {
	domain = strings.TrimSpace(domain)
	if parsed, err := url.Parse(domain); err == nil && parsed.Hostname() != "" {
		domain = parsed.Hostname()
	} else {
		if host, _, err := net.SplitHostPort(domain); err == nil {
			domain = host
		}
	}
	domain = strings.TrimPrefix(domain, "*.")
	domain = strings.TrimSuffix(domain, ".")
	domain = strings.ToLower(domain)
	if net.ParseIP(domain) != nil {
		return domain
	}
	root, err := publicsuffix.EffectiveTLDPlusOne(domain)
	if err != nil {
		return domain
	}
	return root
}

func groupByDomain[T any](items []T, name func(T) string, itemDomains func(T) []string) ([]string, map[string][]T) {
	domainNames := make([]string, 0)
	groups := make(map[string][]T)
	seen := make(map[string]map[string]bool)
	for _, item := range items {
		hostnames := itemDomains(item)
		if len(hostnames) == 0 {
			hostnames = []string{noDomain}
		}
		for _, hostname := range hostnames {
			domain := noDomain
			if hostname != noDomain {
				domain = rootDomain(hostname)
			}
			if _, exists := groups[domain]; !exists {
				domainNames = append(domainNames, domain)
				seen[domain] = make(map[string]bool)
			}
			itemName := name(item)
			if seen[domain][itemName] {
				continue
			}
			seen[domain][itemName] = true
			groups[domain] = append(groups[domain], item)
		}
	}
	slices.Sort(domainNames)
	for _, domain := range domainNames {
		slices.SortFunc(groups[domain], func(a, b T) int {
			return strings.Compare(name(a), name(b))
		})
	}
	return domainNames, groups
}

func serviceDomainsInRoot(service parser.Service, domain string) string {
	if domain == noDomain {
		return "—"
	}
	hostnames := make([]string, 0, len(service.Domains))
	for _, hostname := range service.Domains {
		if rootDomain(hostname) == domain {
			hostnames = append(hostnames, hostname)
		}
	}
	slices.Sort(hostnames)
	return strings.Join(hostnames, ", ")
}

func renderDomainGroups[T any](domains []string, groups map[string][]T, headers []string, row func(string, int, T) []string, style func([]T) table.StyleFunc) string {
	var b strings.Builder
	for i, domain := range domains {
		if i > 0 {
			b.WriteString("\n\n")
		}
		b.WriteString(headingStyle.Render(domain))
		b.WriteByte('\n')
		group := groups[domain]
		rows := make([][]string, len(group))
		for j, item := range group {
			rows[j] = row(domain, j+1, item)
		}
		b.WriteString(renderTable(headers, rows, style(group)))
	}
	return b.String()
}

func serviceGroupRow(domain string, n int, service parser.Service) []string {
	return []string{fmt.Sprintf("%d", n), service.Name, service.LocalAddr, fmt.Sprintf("%d", service.VPSPort), serviceDomainsInRoot(service, domain)}
}

func groupServicesByDomain(services []parser.Service) ([]string, map[string][]parser.Service) {
	return groupByDomain(services, func(service parser.Service) string { return service.Name }, func(service parser.Service) []string { return service.Domains })
}

// RenderServicesByDomain renders one service table beneath each root-domain
// heading and returns the number of rendered groups.
func RenderServicesByDomain(services []parser.Service) (string, int) {
	domains, groups := groupServicesByDomain(services)
	return renderDomainGroups(domains, groups,
		[]string{"#", "SERVICE", "LOCAL ADDRESS", "VPS PORT", "DOMAINS"},
		serviceGroupRow,
		func([]parser.Service) table.StyleFunc { return serviceCellStyle },
	), len(domains)
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

// RenderServicesSyncByDomain renders sync status grouped beneath domain headings
// and returns the number of rendered groups.
func RenderServicesSyncByDomain(rows []ServiceRow) (string, int) {
	domains, groups := groupByDomain(rows, func(row ServiceRow) string { return row.Service.Name }, func(row ServiceRow) []string { return row.Service.Domains })
	return renderDomainGroups(domains, groups,
		[]string{"#", "SERVICE", "LOCAL ADDRESS", "VPS PORT", "DOMAINS", "LOCAL", "REMOTE"},
		func(domain string, n int, row ServiceRow) []string {
			return append(serviceGroupRow(domain, n, row.Service), row.Local.mark(), row.Remote.mark())
		},
		func(group []ServiceRow) table.StyleFunc {
			return func(r, c int) lipgloss.Style {
				switch {
				case r == table.HeaderRow:
					return headerCellStyle
				case c == 5:
					return group[r].Local.style()
				case c == 6:
					return group[r].Remote.style()
				default:
					return serviceCellStyle(r, c)
				}
			}
		},
	), len(domains)
}
