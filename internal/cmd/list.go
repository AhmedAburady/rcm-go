package cmd

import (
	"github.com/spf13/cobra"

	"github.com/AhmedAburady/rcm-go/internal/parser"
	"github.com/AhmedAburady/rcm-go/internal/ui"
)

func newListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all configured services",
		Long:  `Display all services parsed from the Caddyfile.`,
		Args:  cobra.NoArgs,
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

			out(c, "%s", ui.RenderServices(services))
			out(c, "%s", ui.Info("Total: %d services", len(services)))
			return nil
		},
	}
}
