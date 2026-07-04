package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/AhmedAburady/rcm-go/internal/config"
)

var configPath string

// Root returns the root command for the CLI.
func Root() *cobra.Command {
	root := &cobra.Command{
		Use:   "rcm",
		Short: "Rathole Caddy Manager",
		Long: `RCM is a CLI tool that simplifies managing Rathole tunnels
with Caddy reverse proxy integration.

It uses the Caddyfile as the source of truth, parsing service
definitions and generating rathole configurations automatically.`,
		// PersistentPreRun (not cobra.OnInitialize, which appends to a process-wide
		// list every Root() call) loads config fresh for the executing command.
		PersistentPreRun: func(_ *cobra.Command, _ []string) { initConfig() },
	}
	root.PersistentFlags().StringVarP(&configPath, "config", "c", "",
		"config file (default: ~/.config/rcm/config.yaml)")

	root.AddCommand(newListCmd(), newEditCmd(), newSyncCmd(), newStatusCmd(), newPullCmd(), newRestartCmd(), newVersionCmd())
	return root
}

var loadedCfg *config.Config

// loadCfg loads and caches the configuration for the process, surfacing any
// error recorded while reading the config file.
func loadCfg() (*config.Config, error) {
	if loadedCfg != nil {
		return loadedCfg, nil
	}
	if configErr != nil {
		return nil, configErr
	}
	c, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}
	loadedCfg = c
	return c, nil
}

// out writes a formatted line to the command's output stream.
func out(cmd *cobra.Command, format string, args ...any) {
	fmt.Fprintf(cmd.OutOrStdout(), format+"\n", args...)
}

var configErr error

func initConfig() {
	// Reset cached state so a fresh Root() invocation (notably in tests) does not
	// inherit a prior run's config or error.
	loadedCfg = nil
	configErr = nil

	if configPath != "" {
		viper.SetConfigFile(configPath)
	} else {
		home, err := os.UserHomeDir()
		cobra.CheckErr(err)

		viper.AddConfigPath(home + "/.config/rcm")
		viper.AddConfigPath(".")
		viper.SetConfigType("yaml")
		viper.SetConfigName("config")
	}

	viper.SetEnvPrefix("RCM")
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			home, _ := os.UserHomeDir()
			configErr = fmt.Errorf("config file not found\n\nCreate one at: %s/.config/rcm/config.yaml\n\nSee: https://github.com/AhmedAburady/rcm-go#configuration", home)
		} else {
			fmt.Fprintf(os.Stderr, "Error reading config: %v\n", err)
			os.Exit(1)
		}
	}
}
