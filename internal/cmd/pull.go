package cmd

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	"github.com/AhmedAburady/rcm-go/internal/config"
	"github.com/AhmedAburady/rcm-go/internal/parser"
	"github.com/AhmedAburady/rcm-go/internal/ssh"
	"github.com/AhmedAburady/rcm-go/internal/tui/views"
)

var pullCmd = &cobra.Command{
	Use:   "pull",
	Short: "Pull Caddyfile from remote server",
	Long: `Download the Caddyfile from the VPS to your local machine.

This is useful for syncing your local Caddyfile with the remote one,
especially when setting up a new machine or recovering from changes
made directly on the server.`,
	RunE: runPull,
}

var (
	pullForce bool
	pullPlain bool
)

func init() {
	rootCmd.AddCommand(pullCmd)
	pullCmd.Flags().BoolVarP(&pullForce, "force", "f", false, "Overwrite local file without confirmation")
	pullCmd.Flags().BoolVarP(&pullPlain, "plain", "p", false, "Plain text output (no TUI)")
}

func runPull(cmd *cobra.Command, args []string) error {
	if configErr != nil {
		return configErr
	}

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	if pullPlain {
		return runPullPlain(cfg)
	}

	// Launch TUI with main app, starting at pull view
	model := views.NewAppModelWithView(cfg, views.ViewPull)
	p := tea.NewProgram(model, tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		return fmt.Errorf("TUI error: %w", err)
	}

	return nil
}

func runPullPlain(cfg *config.Config) error {
	// Check if local file exists
	localPath := cfg.Paths.Caddyfile
	if _, err := os.Stat(localPath); err == nil && !pullForce {
		fmt.Printf("Local Caddyfile already exists at %s\n", localPath)
		fmt.Print("Overwrite? [y/N]: ")

		reader := bufio.NewReader(os.Stdin)
		response, _ := reader.ReadString('\n')
		response = strings.TrimSpace(strings.ToLower(response))

		if response != "y" && response != "yes" {
			fmt.Println("Aborted.")
			return nil
		}
	}

	// Connect to server
	fmt.Printf("Connecting to %s...\n", cfg.Server.Host)
	client, err := ssh.GetClient(cfg.Server.Host, cfg.Server.User, cfg.Server.SSHKey)
	if err != nil {
		return fmt.Errorf("connect to server: %w", err)
	}
	// Don't close - connection is pooled and reused

	// Download Caddyfile
	fmt.Printf("Downloading Caddyfile from %s...\n", cfg.Server.Caddyfile)
	content, err := client.DownloadContent(cfg.Server.Caddyfile)
	if err != nil {
		return fmt.Errorf("download caddyfile: %w", err)
	}

	// Ensure parent directory exists
	dir := filepath.Dir(localPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create directory %s: %w", dir, err)
	}

	// Write to local path
	if err := os.WriteFile(localPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("write local file: %w", err)
	}

	fmt.Printf("✓ Downloaded Caddyfile to %s\n", localPath)

	// Parse and show summary
	services, err := parser.ParseContent(content)
	if err == nil && len(services) > 0 {
		fmt.Printf("\nDiscovered %d services:\n", len(services))
		for _, s := range services {
			fmt.Printf("  • %s (%s)\n", s.Name, s.PrimaryDomain())
		}
	}

	return nil
}
