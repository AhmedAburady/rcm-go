package cmd

import (
	"github.com/spf13/cobra"

	"github.com/AhmedAburady/rcm-go/internal/parser"
	"github.com/AhmedAburady/rcm-go/internal/ui"
)

func newListCmd() *cobra.Command {
	var noCheck bool
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all configured services",
		Long: `Display all services defined in the Caddyfile.

By default list also connects to the VPS, downloads its Caddyfile, and compares
it to your local Caddyfile (the source of truth) per service: LOCAL marks the
service present in your local Caddyfile, REMOTE marks it present and matching on
the VPS. Pass --no-check to skip the SSH probe for an instant, offline listing.`,
		Args: cobra.NoArgs,
		RunE: func(c *cobra.Command, _ []string) error {
			cfg, err := loadCfg()
			if err != nil {
				return err
			}

			services, err := parser.ParseFile(cfg.Paths.Caddyfile)
			if err != nil {
				return err
			}

			if len(services) == 0 {
				out(c, "%s", ui.Info("No services found in %s", cfg.Paths.Caddyfile))
				return nil
			}

			if noCheck {
				rendered, groupCount := ui.RenderServicesByDomain(services)
				out(c, "%s", rendered)
				out(c, "%s", ui.Info("Total: %d services · Root domains: %d (sync check skipped)", len(services), groupCount))
				return nil
			}

			var rows []ui.ServiceRow
			var probeErr error
			ui.WithSpinner("Checking deployment status …", func() {
				rows, probeErr = probeServiceSync(cfg, services)
			})

			rendered, groupCount := ui.RenderServicesSyncByDomain(rows)
			out(c, "%s", rendered)
			if probeErr != nil {
				out(c, "%s", ui.Warn("REMOTE status unavailable: %v", probeErr))
			}
			out(c, "%s", ui.Info("Total: %d services · Root domains: %d", len(rows), groupCount))
			return nil
		},
	}
	cmd.Flags().BoolVar(&noCheck, "no-check", false, "Skip the SSH sync check (offline, instant)")
	return cmd
}
