package cmd

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"sort"

	"github.com/spf13/cobra"

	"github.com/AhmedAburady/rcm-go/internal/config"
	"github.com/AhmedAburady/rcm-go/internal/generator"
	"github.com/AhmedAburady/rcm-go/internal/parser"
	"github.com/AhmedAburady/rcm-go/internal/ssh"
	"github.com/AhmedAburady/rcm-go/internal/ui"
)

func newSyncCmd() *cobra.Command {
	var dryRun bool
	cmd := &cobra.Command{
		Use:   "sync",
		Short: "Sync configuration to remote servers",
		Long: `Parse the local Caddyfile, generate rathole configs, and deploy to both
the VPS and the home client.

Steps: parse the Caddyfile, generate server.toml and client.toml, upload them
(plus the Caddyfile to the VPS), then restart rathole on both machines and caddy
on the VPS.`,
		Args: cobra.NoArgs,
		RunE: func(c *cobra.Command, _ []string) error {
			cfg, err := loadCfg()
			if err != nil {
				return err
			}
			return runSync(c, cfg, dryRun)
		},
	}
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Preview changes without deploying")
	return cmd
}

func runSync(c *cobra.Command, cfg *config.Config, dryRun bool) error {
	out(c, "%s", ui.Step("Parsing %s …", cfg.Paths.Caddyfile))
	services, err := parser.ParseFile(cfg.Paths.Caddyfile)
	if err != nil {
		return fmt.Errorf("parse caddyfile: %w", err)
	}
	if len(services) == 0 {
		return fmt.Errorf("no services found in %s", cfg.Paths.Caddyfile)
	}
	sort.Slice(services, func(i, j int) bool { return services[i].Name < services[j].Name })

	out(c, "%s", ui.RenderServices(services))

	if dryRun {
		// Purely local preview: no SSH, so it is safe to run offline.
		out(c, "%s", ui.Info("dry-run: nothing deployed"))
		return nil
	}

	if err := config.ResolveRatholeSecrets(cfg); err != nil {
		return fmt.Errorf("resolve secrets: %w", err)
	}

	out(c, "%s", ui.Step("Generating configs …"))
	serverTOML, err := generator.GenerateServerTOML(cfg, services)
	if err != nil {
		return fmt.Errorf("generate server config: %w", err)
	}
	clientTOML, err := generator.GenerateClientTOML(cfg, services)
	if err != nil {
		return fmt.Errorf("generate client config: %w", err)
	}
	var caddyContent string
	if cfg.Server.Caddyfile != "" {
		b, err := os.ReadFile(cfg.Paths.Caddyfile)
		if err != nil {
			return fmt.Errorf("read local caddyfile: %w", err)
		}
		caddyContent = string(b)
	}

	serverClient, err := ssh.GetClient(cfg.Server.Host, cfg.Server.User, cfg.Server.SSHAuth())
	if err != nil {
		return fmt.Errorf("connect to server: %w", err)
	}
	clientClient, err := ssh.GetClient(cfg.Client.Host, cfg.Client.User, cfg.Client.SSHAuth())
	if err != nil {
		return fmt.Errorf("connect to client: %w", err)
	}

	// Compare the generated config against what's already on each host and
	// upload only what changed, so an unchanged sync is a no-op (no restarts).
	// Upload everything before restarting anything, so a restart failure on one
	// host never leaves the other host with stale (un-uploaded) config.
	serverChanged, err := uploadIfChanged(c, serverClient, serverTOML, cfg.Server.RatholeConfig, "rathole config (server)")
	if err != nil {
		return err
	}
	caddyChanged := false
	if cfg.Server.Caddyfile != "" {
		caddyChanged, err = uploadIfChanged(c, serverClient, caddyContent, cfg.Server.Caddyfile, "Caddyfile")
		if err != nil {
			return err
		}
	}
	clientChanged, err := uploadIfChanged(c, clientClient, clientTOML, cfg.Client.RatholeConfig, "rathole config (client)")
	if err != nil {
		return err
	}

	if !serverChanged && !caddyChanged && !clientChanged {
		out(c, "%s", ui.OK("Already up to date — nothing to deploy"))
		return nil
	}

	// Restart only the services whose config actually changed. Attempt every
	// needed restart even if one fails, so a single host's failure doesn't leave
	// the other end running an old process against new config.
	var restartErrs []error
	if serverChanged || caddyChanged {
		out(c, "%s", ui.Heading("Server (%s)", cfg.Server.Host))
		if serverChanged {
			if err := restartUnit(c, serverClient, "rathole-server"); err != nil {
				restartErrs = append(restartErrs, err)
			}
		}
		if caddyChanged {
			if err := restartCaddy(c, serverClient, cfg.Server.CaddyComposeDir); err != nil {
				restartErrs = append(restartErrs, err)
			}
		}
	}
	if clientChanged {
		out(c, "%s", ui.Heading("Client (%s)", cfg.Client.Host))
		if err := restartUnit(c, clientClient, "rathole-client"); err != nil {
			restartErrs = append(restartErrs, err)
		}
	}

	if len(restartErrs) > 0 {
		out(c, "%s", ui.Warn("partial restart — configs are uploaded but some services did not restart; both hosts may need attention"))
		return errors.Join(restartErrs...)
	}

	out(c, "%s", ui.OK("Synced %d services", len(services)))
	return nil
}

// uploadIfChanged uploads content to remotePath only when it differs from what's
// already there (compared by SHA-256), reporting whether it changed.
func uploadIfChanged(c *cobra.Command, client *ssh.Client, content, remotePath, label string) (bool, error) {
	if remote, ok := client.FileSHA256(remotePath); ok && remote == contentSHA256(content) {
		out(c, "%s", ui.Info("%s unchanged", label))
		return false, nil
	}
	out(c, "%s", ui.Step("Uploading %s …", label))
	if err := client.UploadContent(content, remotePath); err != nil {
		return false, fmt.Errorf("upload %s: %w", label, err)
	}
	return true, nil
}

func contentSHA256(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}
