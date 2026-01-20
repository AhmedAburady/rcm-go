package cmd

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	"github.com/AhmedAburady/rcm-go/internal/config"
	"github.com/AhmedAburady/rcm-go/internal/ssh"
	"github.com/AhmedAburady/rcm-go/internal/tui/views"
)

var restartCmd = &cobra.Command{
	Use:   "restart",
	Short: "Restart services",
	Long: `Restart rathole and caddy services on VPS and/or client.

By default, restarts services on both machines. Use flags to
restart only specific machines.`,
	RunE: runRestart,
}

var (
	restartServer bool
	restartClient bool
	restartPlain  bool
)

func init() {
	rootCmd.AddCommand(restartCmd)
	restartCmd.Flags().BoolVarP(&restartServer, "server", "s", false, "Restart server services only")
	restartCmd.Flags().BoolVar(&restartClient, "client", false, "Restart client services only")
	restartCmd.Flags().BoolVarP(&restartPlain, "plain", "p", false, "Plain text output (no TUI)")
}

func runRestart(cmd *cobra.Command, args []string) error {
	if configErr != nil {
		return configErr
	}

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	// If neither flag specified, restart both
	if !restartServer && !restartClient {
		restartServer = true
		restartClient = true
	}

	if restartPlain {
		return runRestartPlain(cfg)
	}

	// Launch TUI with main app, starting at restart view
	model := views.NewAppModelWithView(cfg, views.ViewRestart)
	p := tea.NewProgram(model, tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		return fmt.Errorf("TUI error: %w", err)
	}

	return nil
}

func runRestartPlain(cfg *config.Config) error {
	if restartServer {
		fmt.Printf("Restarting services on server (%s)...\n", cfg.Server.Host)
		if err := restartServerServices(cfg); err != nil {
			return err
		}
	}

	if restartClient {
		fmt.Printf("Restarting services on client (%s)...\n", cfg.Client.Host)
		if err := restartClientServices(cfg); err != nil {
			return err
		}
	}

	fmt.Println("\n✓ All services restarted successfully")
	return nil
}

func restartServerServices(cfg *config.Config) error {
	client, err := ssh.GetClient(cfg.Server.Host, cfg.Server.User, cfg.Server.SSHKey)
	if err != nil {
		return fmt.Errorf("connect to server: %w", err)
	}
	// Don't close - connection is pooled and reused

	fmt.Print("  Restarting rathole-server... ")
	if err := client.RestartService("rathole-server"); err != nil {
		fmt.Println("✗")
		return fmt.Errorf("restart rathole-server: %w", err)
	}
	fmt.Println("✓")

	if cfg.Server.CaddyComposeDir != "" {
		fmt.Print("  Restarting caddy... ")
		if err := client.RestartDockerCompose(cfg.Server.CaddyComposeDir); err != nil {
			fmt.Println("✗")
			return fmt.Errorf("restart caddy: %w", err)
		}
		fmt.Println("✓")
	}

	return nil
}

func restartClientServices(cfg *config.Config) error {
	client, err := ssh.GetClient(cfg.Client.Host, cfg.Client.User, cfg.Client.SSHKey)
	if err != nil {
		return fmt.Errorf("connect to client: %w", err)
	}
	// Don't close - connection is pooled and reused

	fmt.Print("  Restarting rathole-client... ")
	if err := client.RestartService("rathole-client"); err != nil {
		fmt.Println("✗")
		return fmt.Errorf("restart rathole-client: %w", err)
	}
	fmt.Println("✓")

	return nil
}
