# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build & Test Commands

```bash
# Build
go build -o bin/rcm ./cmd/rcm

# Run tests
go test ./...

# Run single test
go test -run TestParseContent ./internal/parser

# Cross-compile
GOOS=linux GOARCH=amd64 go build -o rcm-linux ./cmd/rcm
GOOS=darwin GOARCH=arm64 go build -o rcm-darwin-arm64 ./cmd/rcm
```

## Project Overview

RCM-Go is a CLI/TUI tool for managing Rathole tunnels with Caddy reverse proxy. It parses a Caddyfile to extract service definitions, generates Rathole TOML configs, and deploys them via SSH to a VPS (server) and home machine (client).

## Architecture

```
cmd/rcm/main.go          # Entry point with signal handling
internal/
├── cmd/                 # Cobra commands (root, list, sync, pull, status, restart)
├── config/              # Viper-based config loading from ~/.config/rcm/config.yaml
├── parser/              # Caddyfile parser - extracts services from comment annotations
├── generator/           # Generates server.toml and client.toml using embedded templates
├── ssh/                 # SSH client, connection pool, and remote operations
└── tui/
    ├── views/           # Bubbletea models for each view (app, list, sync, status, pull, restart)
    ├── components/      # Reusable TUI components (servicetable)
    └── styles/          # Lipgloss styling constants
```

## Key Patterns

**TUI Architecture**: Each command can run in TUI mode (default) or plain text mode (`--plain`). The `AppModel` in `internal/tui/views/app.go` manages navigation between views. Each view (ListModel, SyncModel, etc.) implements the Bubbletea `Model` interface.

**Caddyfile Parsing**: Services are defined via comments above domain blocks:
```caddyfile
# servicename: 192.168.1.100:8080
domain.com {
    reverse_proxy localhost:5000
}
```
The parser extracts: service name, local address (from comment), VPS port (from reverse_proxy), and domains.

**Template Generation**: TOML configs are generated from embedded templates in `internal/generator/templates/` using Go's `text/template`.

**SSH Operations**: The `ssh` package provides a connection pool (`pool.go`) for reusing connections and operations like `RestartService`, `UploadConfig`, `GetSystemdStatus`.

## Config File

Located at `~/.config/rcm/config.yaml`. Required fields: `server.host`, `client.host`. The config is loaded via Viper in `internal/cmd/root.go:initConfig()`.
