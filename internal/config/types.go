package config

import (
	"os"
	"path/filepath"
)

// Config is the root configuration structure
type Config struct {
	Paths   PathsConfig   `mapstructure:"paths"`
	Server  ServerConfig  `mapstructure:"server"`
	Client  ClientConfig  `mapstructure:"client"`
	Rathole RatholeConfig `mapstructure:"rathole"`
}

// PathsConfig holds local path settings
type PathsConfig struct {
	Caddyfile string `mapstructure:"caddyfile"`
	SSHDir    string `mapstructure:"ssh_dir"`
}

// ServerConfig holds VPS connection settings
type ServerConfig struct {
	Host            string `mapstructure:"host"`
	User            string `mapstructure:"user"`
	SSHKey          string `mapstructure:"ssh_key"`
	RatholeConfig   string `mapstructure:"rathole_config"`
	Caddyfile       string `mapstructure:"caddyfile"`
	CaddyComposeDir string `mapstructure:"caddy_compose_dir"`
}

// ClientConfig holds home machine connection settings
type ClientConfig struct {
	Host          string `mapstructure:"host"`
	User          string `mapstructure:"user"`
	SSHKey        string `mapstructure:"ssh_key"`
	RatholeConfig string `mapstructure:"rathole_config"`
}

// RatholeConfig holds rathole-specific settings
type RatholeConfig struct {
	BindPort         int    `mapstructure:"bind_port"`
	Token            string `mapstructure:"token"`
	ServerPrivateKey string `mapstructure:"server_private_key"`
	ServerPublicKey  string `mapstructure:"server_public_key"`
}

// ExpandPath expands ~ to home directory
func ExpandPath(path string) string {
	if len(path) == 0 {
		return path
	}
	if path[0] == '~' {
		home, err := os.UserHomeDir()
		if err != nil {
			return path
		}
		return filepath.Join(home, path[1:])
	}
	return path
}
