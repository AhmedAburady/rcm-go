package cmd

import "github.com/spf13/cobra"

// Build information, injected via -ldflags at build time (see Makefile).
var (
	Version   = "dev"
	Commit    = "none"
	BuildDate = "unknown"
)

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		Args:  cobra.NoArgs,
		Run: func(c *cobra.Command, _ []string) {
			out(c, "rcm %s", Version)
			out(c, "  commit: %s", Commit)
			out(c, "  built:  %s", BuildDate)
		},
	}
}
