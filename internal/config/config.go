package config

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/AhmedAburady/rcm-go/internal/ssh"
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

	// Resolve connectivity references (paths, hosts, users, key paths, agent
	// sockets). These are needed by every command that connects, and in practice
	// contain no 1Password references — so this is prompt-free. Deployment
	// secrets in the rathole block are deliberately NOT resolved here; they are
	// resolved at the point the generator consumes them (ResolveRatholeSecrets),
	// so commands that never generate configs never authenticate to 1Password
	// for secrets they don't use.
	if err := resolveRefs(&cfg.Paths, &cfg.Server, &cfg.Client); err != nil {
		return nil, fmt.Errorf("resolve config: %w", err)
	}

	// Expand paths
	cfg.Paths.Caddyfile = ExpandPath(cfg.Paths.Caddyfile)
	cfg.Paths.SSHDir = ExpandPath(cfg.Paths.SSHDir)

	// Handle SSH keys - if just a filename, combine with ssh_dir
	cfg.Server.SSHKey = resolveSSHKey(cfg.Server.SSHKey, cfg.Paths.SSHDir)
	cfg.Client.SSHKey = resolveSSHKey(cfg.Client.SSHKey, cfg.Paths.SSHDir)

	// Expand ~ in agent socket overrides (op:// / ${ENV} already resolved above)
	cfg.Server.SSHAgentSocket = ExpandPath(cfg.Server.SSHAgentSocket)
	cfg.Client.SSHAgentSocket = ExpandPath(cfg.Client.SSHAgentSocket)

	// Validate required fields
	if cfg.Server.Host == "" {
		return nil, fmt.Errorf("server.host is required")
	}
	if cfg.Client.Host == "" {
		return nil, fmt.Errorf("client.host is required")
	}
	if err := validateSSHAuth("server", cfg.Server.SSHAgent, cfg.Server.SSHKey); err != nil {
		return nil, err
	}
	if err := validateSSHAuth("client", cfg.Client.SSHAgent, cfg.Client.SSHKey); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// validateSSHAuth enforces that exactly one SSH auth mode is configured for a
// host: either ssh_agent (agent mode) or ssh_key (file mode), never both and
// never neither. Setting both is rejected rather than silently preferring the
// agent, so a leftover ssh_key (or stray ssh_agent) surfaces as a clear error
// instead of an unexpected identity.
func validateSSHAuth(host string, useAgent bool, keyPath string) error {
	switch {
	case useAgent && keyPath != "":
		return fmt.Errorf("%s: ssh_agent and ssh_key are mutually exclusive — set only one", host)
	case !useAgent && keyPath == "":
		return fmt.Errorf("%s: set ssh_key (file-based) or ssh_agent: true", host)
	}
	return nil
}

// ResolveRatholeSecrets resolves the op:// / ${ENV} references in the rathole
// block (token and Noise keys). These secrets are consumed only when generating
// rathole configs, so resolution — and the 1Password authentication it may
// trigger — is deferred to that point rather than performed eagerly in Load.
// Call it immediately before generation. Idempotent: once resolved, the values
// no longer look like references, so repeat calls are no-ops.
func ResolveRatholeSecrets(cfg *Config) error {
	if err := resolveRefs(&cfg.Rathole); err != nil {
		return fmt.Errorf("resolve rathole secrets: %w", err)
	}
	return nil
}

// SSHAuth builds the SSH auth descriptor for the server connection.
func (s ServerConfig) SSHAuth() ssh.AuthConfig {
	return ssh.AuthConfig{KeyPath: s.SSHKey, UseAgent: s.SSHAgent, AgentSocket: s.SSHAgentSocket}
}

// SSHAuth builds the SSH auth descriptor for the client connection.
func (c ClientConfig) SSHAuth() ssh.AuthConfig {
	return ssh.AuthConfig{KeyPath: c.SSHKey, UseAgent: c.SSHAgent, AgentSocket: c.SSHAgentSocket}
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
