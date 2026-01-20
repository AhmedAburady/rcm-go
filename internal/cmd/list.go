package cmd

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	"github.com/AhmedAburady/rcm-go/internal/config"
	"github.com/AhmedAburady/rcm-go/internal/parser"
	"github.com/AhmedAburady/rcm-go/internal/tui/views"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all configured services",
	Long:  `Display all services parsed from the Caddyfile with their status.`,
	RunE:  runList,
}

var listPlain bool

func init() {
	rootCmd.AddCommand(listCmd)
	listCmd.Flags().BoolVarP(&listPlain, "plain", "p", false, "Plain text output (no TUI)")
}

func runList(cmd *cobra.Command, args []string) error {
	if configErr != nil {
		return configErr
	}

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	if listPlain {
		return runListPlain(cfg)
	}

	// Launch TUI with main app, starting at list view
	model := views.NewAppModelWithView(cfg, views.ViewList)
	p := tea.NewProgram(model, tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		return fmt.Errorf("TUI error: %w", err)
	}

	return nil
}

func runListPlain(cfg *config.Config) error {
	services, err := parser.ParseFile(cfg.Paths.Caddyfile)
	if err != nil {
		return err
	}

	if len(services) == 0 {
		fmt.Println("No services found in Caddyfile")
		return nil
	}

	fmt.Printf("%-15s %-22s %-10s %s\n", "SERVICE", "LOCAL ADDRESS", "VPS PORT", "DOMAINS")
	fmt.Println(strings.Repeat("-", 75))

	for _, s := range services {
		domains := strings.Join(s.Domains, ", ")
		fmt.Printf("%-15s %-22s %-10d %s\n",
			s.Name, s.LocalAddr, s.VPSPort, domains)
	}

	fmt.Printf("\nTotal: %d services\n", len(services))
	return nil
}
