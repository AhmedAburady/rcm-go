package cmd

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/AhmedAburady/rcm-go/internal/config"
	"github.com/AhmedAburady/rcm-go/internal/tui/views"
)

var cfgFile string

var rootCmd = &cobra.Command{
	Use:   "rcm",
	Short: "Rathole Caddy Manager",
	Long: `RCM is a CLI tool that simplifies managing Rathole tunnels
with Caddy reverse proxy integration.

It uses the Caddyfile as the source of truth, parsing service
definitions and generating rathole configurations automatically.

Run without arguments to launch the interactive TUI.`,
	RunE: runApp,
}

// runApp launches the main TUI application
func runApp(cmd *cobra.Command, args []string) error {
	// Check for config file error first
	if configErr != nil {
		return configErr
	}

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	model := views.NewAppModel(cfg)
	p := tea.NewProgram(model, tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		return fmt.Errorf("TUI error: %w", err)
	}

	return nil
}

// Execute runs the root command
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", "",
		"config file (default: ~/.config/rcm/config.yaml)")
}

var configErr error

func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
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
			// Store the error for later - commands that need config will check this
			home, _ := os.UserHomeDir()
			configErr = fmt.Errorf("config file not found\n\nCreate one at: %s/.config/rcm/config.yaml\n\nSee: https://github.com/AhmedAburady/rcm-go#configuration", home)
		} else {
			fmt.Fprintf(os.Stderr, "Error reading config: %v\n", err)
			os.Exit(1)
		}
	}
}

// GetConfigError returns any config loading error
func GetConfigError() error {
	return configErr
}
