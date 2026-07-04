package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"strings"

	"github.com/spf13/cobra"

	"github.com/AhmedAburady/rcm-go/internal/ui"
)

func newEditCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "edit",
		Short: "Open the Caddyfile in your editor",
		Long:  `Open the Caddyfile in the editor configured via the $EDITOR environment variable.`,
		Args:  cobra.NoArgs,
		RunE: func(c *cobra.Command, _ []string) error {
			cfg, err := loadCfg()
			if err != nil {
				return err
			}

			// Resolve the editor from $EDITOR only. strings.Fields splits values
			// like "code -w" so flags are preserved as separate arguments.
			parts := strings.Fields(os.Getenv("EDITOR"))
			if len(parts) == 0 {
				out(c, "%s", ui.Warn("$EDITOR is not set"))
				out(c, "%s", ui.Info("Set it to your editor and try again, e.g. export EDITOR=vim"))
				return nil
			}

			args := append(parts[1:], cfg.Paths.Caddyfile)
			editor := exec.Command(parts[0], args...)
			editor.Stdin = os.Stdin
			editor.Stdout = os.Stdout
			editor.Stderr = os.Stderr

			signal.Ignore(os.Interrupt)
			defer signal.Reset(os.Interrupt)

			if err := editor.Run(); err != nil {
				return fmt.Errorf("edit: %w", err)
			}
			return nil
		},
	}
}
