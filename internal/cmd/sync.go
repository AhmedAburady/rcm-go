package cmd

import (
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
		printSyncSummary(c, cfg, services)
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

	// Upload to both hosts before restarting anything, so a restart failure on
	// one host never leaves the other host with stale (un-uploaded) config.
	serverClient, err := uploadServer(c, cfg, serverTOML)
	if err != nil {
		return err
	}
	clientClient, err := uploadClient(c, cfg, clientTOML)
	if err != nil {
		return err
	}

	// Both configs are already uploaded; attempt every restart even if one
	// fails, so a single host's failure doesn't leave the other end running an
	// old process against the new config (a divergent, tunnel-down state).
	var restartErrs []error
	out(c, "%s", ui.Heading("Server (%s)", cfg.Server.Host))
	if err := restartUnit(c, serverClient, "rathole-server"); err != nil {
		restartErrs = append(restartErrs, err)
	}
	if err := restartCaddy(c, serverClient, cfg.Server.CaddyComposeDir); err != nil {
		restartErrs = append(restartErrs, err)
	}

	out(c, "%s", ui.Heading("Client (%s)", cfg.Client.Host))
	if err := restartUnit(c, clientClient, "rathole-client"); err != nil {
		restartErrs = append(restartErrs, err)
	}

	if len(restartErrs) > 0 {
		out(c, "%s", ui.Warn("partial restart — configs are uploaded but some services did not restart; both hosts may need attention"))
		return errors.Join(restartErrs...)
	}

	out(c, "%s", ui.OK("Deployed %d services", len(services)))
	return nil
}

func uploadServer(c *cobra.Command, cfg *config.Config, serverTOML string) (*ssh.Client, error) {
	out(c, "%s", ui.Step("Uploading rathole config to %s …", cfg.Server.Host))
	client, err := ssh.GetClient(cfg.Server.Host, cfg.Server.User, cfg.Server.SSHAuth())
	if err != nil {
		return nil, fmt.Errorf("connect to server: %w", err)
	}
	if err := client.UploadContent(serverTOML, cfg.Server.RatholeConfig); err != nil {
		return nil, fmt.Errorf("upload rathole config: %w", err)
	}

	if cfg.Server.Caddyfile != "" {
		caddyContent, err := os.ReadFile(cfg.Paths.Caddyfile)
		if err != nil {
			return nil, fmt.Errorf("read local caddyfile: %w", err)
		}
		out(c, "%s", ui.Step("Uploading Caddyfile …"))
		if err := client.UploadContent(string(caddyContent), cfg.Server.Caddyfile); err != nil {
			return nil, fmt.Errorf("upload caddyfile: %w", err)
		}
	}
	return client, nil
}

func uploadClient(c *cobra.Command, cfg *config.Config, clientTOML string) (*ssh.Client, error) {
	out(c, "%s", ui.Step("Uploading rathole config to %s …", cfg.Client.Host))
	client, err := ssh.GetClient(cfg.Client.Host, cfg.Client.User, cfg.Client.SSHAuth())
	if err != nil {
		return nil, fmt.Errorf("connect to client: %w", err)
	}
	if err := client.UploadContent(clientTOML, cfg.Client.RatholeConfig); err != nil {
		return nil, fmt.Errorf("upload rathole config: %w", err)
	}
	return client, nil
}

// printSyncSummary reports how many local services are new versus already present
// on the server. A failure to reach or parse the remote Caddyfile is surfaced as
// a warning rather than silently reporting every service as new.
func printSyncSummary(c *cobra.Command, cfg *config.Config, services []parser.Service) {
	if cfg.Server.Host == "" || cfg.Server.Caddyfile == "" {
		return
	}
	remoteSvcs, err := fetchRemoteServices(cfg)
	if err != nil {
		out(c, "%s", ui.Warn("could not read remote Caddyfile (%v); new/existing counts unavailable", err))
		return
	}

	remote := make(map[string]bool, len(remoteSvcs))
	for _, s := range remoteSvcs {
		remote[s.Name] = true
	}
	newCount := 0
	for _, s := range services {
		if !remote[s.Name] {
			newCount++
		}
	}
	out(c, "%s", ui.Info("%d new, %d existing", newCount, len(services)-newCount))
}
