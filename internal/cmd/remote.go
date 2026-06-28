package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/AhmedAburady/rcm-go/internal/ssh"
	"github.com/AhmedAburady/rcm-go/internal/ui"
)

// restartCaddy restarts the Caddy docker-compose stack on the server when one is
// configured. Shared by `sync` and `restart`.
func restartCaddy(c *cobra.Command, client *ssh.Client, composeDir string) error {
	if composeDir == "" {
		return nil
	}
	out(c, "%s", ui.Step("Restarting caddy …"))
	if err := client.RestartDockerCompose(composeDir); err != nil {
		out(c, "  %s", ui.Fail("caddy"))
		return fmt.Errorf("restart caddy: %w", err)
	}
	out(c, "  %s", ui.OK("caddy"))
	return nil
}
