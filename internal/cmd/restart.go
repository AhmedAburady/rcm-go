package cmd

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/AhmedAburady/rcm-go/internal/config"
	"github.com/AhmedAburady/rcm-go/internal/ssh"
	"github.com/AhmedAburady/rcm-go/internal/ui"
)

func newRestartCmd() *cobra.Command {
	var (
		server bool
		client bool
	)
	cmd := &cobra.Command{
		Use:   "restart",
		Short: "Restart services",
		Long: `Restart rathole and caddy services on VPS and/or client.

By default, restarts services on both machines. Use flags to
restart only specific machines.`,
		Args: cobra.NoArgs,
		RunE: func(c *cobra.Command, _ []string) error {
			cfg, err := loadCfg()
			if err != nil {
				return err
			}

			if !server && !client {
				server, client = true, true
			}

			// Best-effort across both machines: a server failure must not skip the
			// client, so the operator sees the full picture in one run (matches sync).
			var errs []error
			if server {
				out(c, "%s", ui.Heading("Server (%s)", cfg.Server.Host))
				if err := restartServerServices(c, cfg); err != nil {
					errs = append(errs, err)
				}
			}
			if client {
				out(c, "%s", ui.Heading("Client (%s)", cfg.Client.Host))
				if err := restartClientServices(c, cfg); err != nil {
					errs = append(errs, err)
				}
			}

			if len(errs) > 0 {
				return errors.Join(errs...)
			}
			out(c, "%s", ui.OK("All services restarted"))
			return nil
		},
	}
	cmd.Flags().BoolVarP(&server, "server", "s", false, "Restart server services only")
	cmd.Flags().BoolVar(&client, "client", false, "Restart client services only")
	return cmd
}

func restartServerServices(c *cobra.Command, cfg *config.Config) error {
	client, err := ssh.GetClient(cfg.Server.Host, cfg.Server.User, cfg.Server.SSHAuth())
	if err != nil {
		return fmt.Errorf("connect to server: %w", err)
	}

	if err := restartUnit(c, client, "rathole-server"); err != nil {
		return err
	}
	if err := restartCaddy(c, client, cfg.Server.CaddyComposeDir); err != nil {
		return err
	}
	return nil
}

func restartClientServices(c *cobra.Command, cfg *config.Config) error {
	client, err := ssh.GetClient(cfg.Client.Host, cfg.Client.User, cfg.Client.SSHAuth())
	if err != nil {
		return fmt.Errorf("connect to client: %w", err)
	}
	return restartUnit(c, client, "rathole-client")
}

func restartUnit(c *cobra.Command, client *ssh.Client, name string) error {
	out(c, "%s", ui.Step("Restarting %s …", name))
	if err := client.RestartService(name); err != nil {
		out(c, "  %s", ui.Fail("%s", name))
		return fmt.Errorf("restart %s: %w", name, err)
	}
	out(c, "  %s", ui.OK("%s", name))
	return nil
}
