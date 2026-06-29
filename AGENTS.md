# AGENTS.md

This file provides guidance to Codex (Codex.ai/code) when working with code in this repository.

## Build & Test Commands

```bash
# Build the runnable binary (or: just build)
go build -o rcm .

# Compile everything / lint / test
go build ./...
go vet ./...
go test ./...

# Run single test
go test -run TestParseContent ./internal/parser

# Apply safe Go modernizations
go fix ./...

# Cross-compile
GOOS=linux GOARCH=amd64 go build -o rcm-linux-amd64 .
GOOS=darwin GOARCH=arm64 go build -o rcm-darwin-arm64 .
```

Module: `github.com/AhmedAburady/rcm-go`, Go 1.26.

## Project Overview

RCM-Go is a flags-driven CLI tool for managing Rathole tunnels with Caddy reverse proxy. It parses a Caddyfile to extract service definitions, generates Rathole TOML configs, and deploys them via SSH to a VPS (server) and home machine (client).

## Architecture

```
main.go                  # Entry point: fang.Execute(cmd.Root()) + SSH cleanup
internal/
├── cmd/                 # Cobra command tree: Root() + newXxxCmd() factories (list, sync, pull, status, restart)
├── config/              # Viper-based config loading from ~/.config/rcm/config.yaml
├── parser/              # Caddyfile parser - extracts services from comment annotations
├── generator/           # Generates server.toml and client.toml using embedded templates
├── ssh/                 # SSH client, connection pool, and remote operations
└── ui/                  # Terminal presentation: lipgloss styles, status lines, width-aware tables, spinner
```

## Key Patterns

**CLI structure**: `main.go` runs `cmd.Root()` through Charm `fang` (styled help, `--version`, error rendering on top of Cobra). Each subcommand is a `newXxxCmd()` factory registered in `internal/cmd/root.go:Root()`. Commands load config via the cached `loadCfg()` helper and print through `out(c, …)` → `cmd.OutOrStdout()`. Tables and status lines come from the `ui` package (`RenderServices`, `RenderServicesSync`, `WithSpinner`, `Step`/`OK`/`Warn`/`Info`/`Fail`). `rcm list` defaults to a per-service local↔VPS Caddyfile sync check (`--no-check` skips it); `internal/cmd/drift.go` runs the probe.

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

Located at `~/.config/rcm/config.yaml`. Required fields: `server.host`, `client.host`. Viper reads the file in `internal/cmd/root.go:initConfig()`; commands obtain the parsed config through the cached `loadCfg()`.
