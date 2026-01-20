package config

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
)

// Load reads and validates the configuration
func Load() (*Config, error) {
	var cfg Config

	// Check if config file was loaded
	if viper.ConfigFileUsed() == "" {
		return nil, fmt.Errorf("no config file loaded")
	}

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

	// Handle SSH keys - if just a filename, combine with ssh_dir
	cfg.Server.SSHKey = resolveSSHKey(cfg.Server.SSHKey, cfg.Paths.SSHDir)
	cfg.Client.SSHKey = resolveSSHKey(cfg.Client.SSHKey, cfg.Paths.SSHDir)

	// Validate required fields
	if cfg.Server.Host == "" {
		return nil, fmt.Errorf("server.host is required")
	}
	if cfg.Client.Host == "" {
		return nil, fmt.Errorf("client.host is required")
	}

	return &cfg, nil
}

// resolveSSHKey resolves the SSH key path
// If it's just a filename, combine with sshDir
// If it starts with ~ or /, treat as full path
func resolveSSHKey(keyPath, sshDir string) string {
	if keyPath == "" {
		return ""
	}

	// If it's already a full path or starts with ~, expand it
	if strings.HasPrefix(keyPath, "~") || strings.HasPrefix(keyPath, "/") {
		return ExpandPath(keyPath)
	}

	// Otherwise, it's just a filename - combine with ssh_dir
	return filepath.Join(sshDir, keyPath)
}

// ConfigPath returns the path of the loaded config file
func ConfigPath() string {
	return viper.ConfigFileUsed()
}
