package cmd

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/AhmedAburady/rcm-go/internal/parser"
	"github.com/AhmedAburady/rcm-go/internal/ssh"
	"github.com/AhmedAburady/rcm-go/internal/ui"
)

func newPullCmd() *cobra.Command {
	var force bool
	cmd := &cobra.Command{
		Use:   "pull",
		Short: "Pull Caddyfile from remote server",
		Long: `Download the Caddyfile from the VPS to your local machine.

Useful for syncing your local Caddyfile with the remote one, especially
when setting up a new machine or recovering from changes made directly
on the server.`,
		Args: cobra.NoArgs,
		RunE: func(c *cobra.Command, _ []string) error {
			cfg, err := loadCfg()
			if err != nil {
				return err
			}

			localPath := cfg.Paths.Caddyfile

			out(c, "%s", ui.Step("Downloading Caddyfile from %s …", cfg.Server.Host))
			client, err := ssh.GetClient(cfg.Server.Host, cfg.Server.User, cfg.Server.SSHAuth())
			if err != nil {
				return fmt.Errorf("connect to server: %w", err)
			}

			content, err := client.DownloadContent(cfg.Server.Caddyfile)
			if err != nil {
				return fmt.Errorf("download caddyfile: %w", err)
			}

			// Compare against the local file: skip the write (and the overwrite
			// prompt) when it already matches, and only prompt when it differs.
			if existing, err := os.ReadFile(localPath); err == nil {
				if string(existing) == content {
					out(c, "%s", ui.OK("Already up to date — local Caddyfile matches remote"))
					return nil
				}
				if !force {
					out(c, "%s", ui.Warn("Local Caddyfile differs from remote at %s", localPath))
					fmt.Fprint(c.OutOrStdout(), "Overwrite? [y/N]: ")

					reader := bufio.NewReader(c.InOrStdin())
					response, _ := reader.ReadString('\n')
					response = strings.TrimSpace(strings.ToLower(response))
					if response != "y" && response != "yes" {
						out(c, "%s", ui.Info("Aborted."))
						return nil
					}
				}
			}

			dir := filepath.Dir(localPath)
			if err := os.MkdirAll(dir, 0o755); err != nil {
				return fmt.Errorf("create directory %s: %w", dir, err)
			}
			if err := os.WriteFile(localPath, []byte(content), 0o644); err != nil {
				return fmt.Errorf("write local file: %w", err)
			}

			out(c, "%s", ui.OK("Downloaded Caddyfile to %s", localPath))

			services, err := parser.ParseContent(content)
			if err == nil && len(services) > 0 {
				out(c, "")
				out(c, "%s", ui.RenderServices(services))
			}
			return nil
		},
	}
	cmd.Flags().BoolVarP(&force, "force", "f", false, "Overwrite local file without confirmation")
	return cmd
}
