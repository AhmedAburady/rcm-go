package cmd

import (
	"sync"

	"github.com/spf13/cobra"

	"github.com/AhmedAburady/rcm-go/internal/config"
	"github.com/AhmedAburady/rcm-go/internal/ssh"
	"github.com/AhmedAburady/rcm-go/internal/ui"
)

func newStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show service status",
		Long: `Check the health of rathole and caddy services on both machines.

Connects to the VPS and home client via SSH and checks rathole-server,
rathole-client, and caddy (docker compose on the VPS).`,
		Args: cobra.NoArgs,
		RunE: func(c *cobra.Command, _ []string) error {
			cfg, err := loadCfg()
			if err != nil {
				return err
			}

			// Probe both hosts concurrently; buffer each host's lines so output
			// stays in a stable server-then-client order regardless of timing.
			var serverLines, clientLines []string
			var wg sync.WaitGroup
			wg.Add(2)
			go func() { defer wg.Done(); serverLines = probeServer(cfg) }()
			go func() { defer wg.Done(); clientLines = probeClient(cfg) }()
			wg.Wait()

			for _, l := range append(serverLines, clientLines...) {
				out(c, "%s", l)
			}
			return nil
		},
	}
}

func probeServer(cfg *config.Config) []string {
	lines := []string{ui.Heading("Server (%s)", cfg.Server.Host)}
	client, err := ssh.GetClient(cfg.Server.Host, cfg.Server.User, cfg.Server.SSHAuth())
	if err != nil {
		return append(lines, "  "+ui.Fail("unable to connect: %v", err))
	}
	lines = append(lines, statusLine(serviceStatus(client, "rathole-server")))
	if cfg.Server.CaddyComposeDir != "" {
		running, status, _ := client.GetDockerComposeStatus(cfg.Server.CaddyComposeDir)
		lines = append(lines, statusLine("caddy (docker)", running, status))
	}
	return lines
}

func probeClient(cfg *config.Config) []string {
	lines := []string{ui.Heading("Client (%s)", cfg.Client.Host)}
	client, err := ssh.GetClient(cfg.Client.Host, cfg.Client.User, cfg.Client.SSHAuth())
	if err != nil {
		return append(lines, "  "+ui.Fail("unable to connect: %v", err))
	}
	return append(lines, statusLine(serviceStatus(client, "rathole-client")))
}

func serviceStatus(client *ssh.Client, name string) (string, bool, string) {
	running, status, _ := client.GetServiceStatus(name)
	return name, running, status
}

func statusLine(name string, running bool, status string) string {
	mark := ui.Cross()
	if running {
		mark = ui.Check()
	}
	return "  " + mark + " " + name + ": " + status
}
