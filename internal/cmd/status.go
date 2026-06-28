package cmd

import (
	"sync"
	"time"

	"github.com/spf13/cobra"

	"github.com/AhmedAburady/rcm-go/internal/config"
	"github.com/AhmedAburady/rcm-go/internal/ssh"
	"github.com/AhmedAburady/rcm-go/internal/ui"
)

// statusProbeTimeout bounds a single host's health probe so a hung SSH session
// (after the dial timeout) can't make `status` appear stuck forever.
const statusProbeTimeout = 20 * time.Second

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
			go func() {
				defer wg.Done()
				serverLines = runProbe(ui.Heading("Server (%s)", cfg.Server.Host), func() []string { return probeServer(cfg) })
			}()
			go func() {
				defer wg.Done()
				clientLines = runProbe(ui.Heading("Client (%s)", cfg.Client.Host), func() []string { return probeClient(cfg) })
			}()
			wg.Wait()

			for _, l := range append(serverLines, clientLines...) {
				out(c, "%s", l)
			}
			return nil
		},
	}
}

// runProbe prepends heading to the probe's lines, or reports a timeout if the
// probe does not return within statusProbeTimeout. A timed-out probe goroutine
// is abandoned and reaped at process exit.
func runProbe(heading string, probe func() []string) []string {
	done := make(chan []string, 1)
	go func() { done <- probe() }()
	select {
	case lines := <-done:
		return append([]string{heading}, lines...)
	case <-time.After(statusProbeTimeout):
		return []string{heading, "  " + ui.Fail("timed out after %s", statusProbeTimeout)}
	}
}

func probeServer(cfg *config.Config) []string {
	client, err := ssh.GetClient(cfg.Server.Host, cfg.Server.User, cfg.Server.SSHAuth())
	if err != nil {
		return []string{"  " + ui.Fail("unable to connect: %v", err)}
	}
	lines := []string{unitStatusLine(client, "rathole-server")}
	if cfg.Server.CaddyComposeDir != "" {
		running, status, err := client.GetDockerComposeStatus(cfg.Server.CaddyComposeDir)
		if err != nil {
			lines = append(lines, "  "+ui.Fail("caddy (docker): %v", err))
		} else {
			lines = append(lines, statusLine("caddy (docker)", running, status))
		}
	}
	return lines
}

func probeClient(cfg *config.Config) []string {
	client, err := ssh.GetClient(cfg.Client.Host, cfg.Client.User, cfg.Client.SSHAuth())
	if err != nil {
		return []string{"  " + ui.Fail("unable to connect: %v", err)}
	}
	return []string{unitStatusLine(client, "rathole-client")}
}

func unitStatusLine(client *ssh.Client, name string) string {
	running, status, err := client.GetServiceStatus(name)
	if err != nil {
		return "  " + ui.Fail("%s: %v", name, err)
	}
	return statusLine(name, running, status)
}

func statusLine(name string, running bool, status string) string {
	mark := ui.Cross()
	if running {
		mark = ui.Check()
	}
	return "  " + mark + " " + name + ": " + status
}
