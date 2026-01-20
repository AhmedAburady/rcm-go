package cmd

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	"github.com/ahmabora1/rcm/internal/config"
	"github.com/ahmabora1/rcm/internal/tui/views"
)

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Sync configuration to remote servers",
	Long: `Parse the local Caddyfile, generate rathole configs,
and deploy to both VPS and home client.

This command will:
1. Parse the local Caddyfile to extract service definitions
2. Generate server.toml and client.toml configs
3. Upload server.toml to the VPS
4. Upload client.toml to the home client
5. Restart rathole-server on VPS
6. Restart rathole-client on home machine`,
	RunE: runSync,
}

var syncDryRun bool

func init() {
	rootCmd.AddCommand(syncCmd)
	syncCmd.Flags().BoolVar(&syncDryRun, "dry-run", false, "Preview changes without deploying")
}

func runSync(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	model := views.NewSyncModel(cfg, syncDryRun)
	p := tea.NewProgram(model, tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		return fmt.Errorf("TUI error: %w", err)
	}

	return nil
}
