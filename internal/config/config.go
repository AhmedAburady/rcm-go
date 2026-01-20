package config

import (
	"fmt"

	"github.com/spf13/viper"
)

// Load reads and validates the configuration
func Load() (*Config, error) {
	var cfg Config

	// Set defaults
	viper.SetDefault("paths.ssh_dir", "~/.ssh")
	viper.SetDefault("server.user", "root")
	viper.SetDefault("rathole.bind_port", 2333)

	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	// Expand paths
	cfg.Paths.Caddyfile = ExpandPath(cfg.Paths.Caddyfile)
	cfg.Paths.SSHDir = ExpandPath(cfg.Paths.SSHDir)
	cfg.Server.SSHKey = ExpandPath(cfg.Server.SSHKey)
	cfg.Client.SSHKey = ExpandPath(cfg.Client.SSHKey)

	// Validate required fields
	if cfg.Server.Host == "" {
		return nil, fmt.Errorf("server.host is required")
	}
	if cfg.Client.Host == "" {
		return nil, fmt.Errorf("client.host is required")
	}

	return &cfg, nil
}

// ConfigPath returns the path of the loaded config file
func ConfigPath() string {
	return viper.ConfigFileUsed()
}
