package cmd

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	"github.com/ahmabora1/rcm/internal/config"
	"github.com/ahmabora1/rcm/internal/tui/views"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show service status",
	Long: `Check the health of rathole and caddy services on both machines.

This command connects to both the VPS and home client via SSH
and checks the status of:
- rathole-server (on VPS)
- rathole-client (on home machine)
- caddy (docker compose on VPS)`,
	RunE: runStatus,
}

func init() {
	rootCmd.AddCommand(statusCmd)
}

func runStatus(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	model := views.NewStatusModel(cfg)
	p := tea.NewProgram(model, tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		return fmt.Errorf("TUI error: %w", err)
	}

	return nil
}
