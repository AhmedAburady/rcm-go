package cmd

import (
	"github.com/spf13/cobra"

	"github.com/AhmedAburady/rcm-go/internal/config"
	"github.com/AhmedAburady/rcm-go/internal/parser"
	"github.com/AhmedAburady/rcm-go/internal/ssh"
	"github.com/AhmedAburady/rcm-go/internal/ui"
)

// fetchRemoteServices downloads and parses the server's Caddyfile, returning the
// services it defines. Shared by `pull` and `sync --dry-run`.
func fetchRemoteServices(cfg *config.Config) ([]parser.Service, error) {
	client, err := ssh.GetClient(cfg.Server.Host, cfg.Server.User, cfg.Server.SSHAuth())
	if err != nil {
		return nil, err
	}
	content, err := client.DownloadContent(cfg.Server.Caddyfile)
	if err != nil {
		return nil, err
	}
	return parser.ParseContent(content)
}

// restartCaddy restarts the Caddy docker-compose stack on the server when one is
// configured. Shared by `sync` and `restart`.
func restartCaddy(c *cobra.Command, client *ssh.Client, composeDir string) error {
	if composeDir == "" {
		return nil
	}
	out(c, "%s", ui.Step("Restarting caddy …"))
	if err := client.RestartDockerCompose(composeDir); err != nil {
		out(c, "  %s", ui.Fail("caddy"))
		return err
	}
	out(c, "  %s", ui.OK("caddy"))
	return nil
}
