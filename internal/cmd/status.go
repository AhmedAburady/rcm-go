package cmd

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	"github.com/ahmabora1/rcm/internal/config"
	"github.com/ahmabora1/rcm/internal/ssh"
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

var statusPlain bool

func init() {
	rootCmd.AddCommand(statusCmd)
	statusCmd.Flags().BoolVarP(&statusPlain, "plain", "p", false, "Plain text output (no TUI)")
}

func runStatus(cmd *cobra.Command, args []string) error {
	if configErr != nil {
		return configErr
	}

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	if statusPlain {
		return runStatusPlain(cfg)
	}

	// Launch TUI with main app, starting at status view
	model := views.NewAppModelWithView(cfg, views.ViewStatus)
	p := tea.NewProgram(model, tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		return fmt.Errorf("TUI error: %w", err)
	}

	return nil
}

func runStatusPlain(cfg *config.Config) error {
	fmt.Println("SERVICE STATUS")
	fmt.Println(strings.Repeat("-", 60))

	// Check server
	fmt.Printf("\nServer (%s):\n", cfg.Server.Host)
	serverClient, err := ssh.GetClient(cfg.Server.Host, cfg.Server.User, cfg.Server.SSHKey)
	if err != nil {
		fmt.Printf("  ✗ Unable to connect: %v\n", err)
	} else {
		// Don't close - connection is pooled and reused

		// Check rathole-server
		running, status, _ := serverClient.GetServiceStatus("rathole-server")
		icon := "✗"
		if running {
			icon = "✓"
		}
		fmt.Printf("  %s rathole-server: %s\n", icon, status)

		// Check caddy if configured
		if cfg.Server.CaddyComposeDir != "" {
			running, status, _ := serverClient.GetDockerComposeStatus(cfg.Server.CaddyComposeDir)
			icon := "✗"
			if running {
				icon = "✓"
			}
			fmt.Printf("  %s caddy (docker): %s\n", icon, status)
		}
	}

	// Check client
	fmt.Printf("\nClient (%s):\n", cfg.Client.Host)
	clientClient, err := ssh.GetClient(cfg.Client.Host, cfg.Client.User, cfg.Client.SSHKey)
	if err != nil {
		fmt.Printf("  ✗ Unable to connect: %v\n", err)
	} else {
		// Don't close - connection is pooled and reused

		// Check rathole-client
		running, status, _ := clientClient.GetServiceStatus("rathole-client")
		icon := "✗"
		if running {
			icon = "✓"
		}
		fmt.Printf("  %s rathole-client: %s\n", icon, status)
	}

	return nil
}
