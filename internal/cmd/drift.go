package cmd

import (
	"fmt"
	"slices"
	"time"

	"github.com/AhmedAburady/rcm-go/internal/config"
	"github.com/AhmedAburady/rcm-go/internal/parser"
	"github.com/AhmedAburady/rcm-go/internal/ssh"
	"github.com/AhmedAburady/rcm-go/internal/ui"
)

const driftProbeTimeout = 20 * time.Second

func probeServiceSync(cfg *config.Config, services []parser.Service) ([]ui.ServiceRow, error) {
	remote, err := remoteCaddyServices(cfg)
	reachable := err == nil

	local := make(map[string]parser.Service, len(services))
	names := make([]string, 0, len(services))
	for _, s := range services {
		local[s.Name] = s
		names = append(names, s.Name)
	}
	for name := range remote {
		if _, seen := local[name]; !seen {
			names = append(names, name)
		}
	}
	slices.Sort(names)

	rows := make([]ui.ServiceRow, 0, len(names))
	for _, name := range names {
		ls, hasL := local[name]
		rs, hasR := remote[name]
		svc := ls
		if !hasL {
			svc = rs
		}
		localState := ui.SyncDrift
		if hasL {
			localState = ui.SyncOK
		}
		rows = append(rows, ui.ServiceRow{
			Service: svc,
			Local:   localState,
			Remote:  remoteSync(reachable, hasL, hasR, ls, rs),
		})
	}
	return rows, err
}

func remoteSync(reachable, hasLocal, hasRemote bool, local, remote parser.Service) ui.Sync {
	if !reachable {
		return ui.SyncUnknown
	}
	if !hasRemote || !hasLocal || !servicesEqual(local, remote) {
		return ui.SyncDrift
	}
	return ui.SyncOK
}

func remoteCaddyServices(cfg *config.Config) (map[string]parser.Service, error) {
	if cfg.Server.Caddyfile == "" {
		return nil, fmt.Errorf("server.caddyfile is not set in the config")
	}

	type result struct {
		services map[string]parser.Service
		err      error
	}
	ch := make(chan result, 1)
	go func() {
		client, err := ssh.GetClient(cfg.Server.Host, cfg.Server.User, cfg.Server.SSHAuth())
		if err != nil {
			ch <- result{nil, fmt.Errorf("connect to server: %w", err)}
			return
		}
		content, err := client.DownloadContent(cfg.Server.Caddyfile)
		if err != nil {
			ch <- result{nil, fmt.Errorf("download caddyfile: %w", err)}
			return
		}
		parsed, err := parser.ParseContent(content)
		if err != nil {
			ch <- result{nil, fmt.Errorf("parse remote caddyfile: %w", err)}
			return
		}
		m := make(map[string]parser.Service, len(parsed))
		for _, s := range parsed {
			m[s.Name] = s
		}
		ch <- result{m, nil}
	}()

	select {
	case r := <-ch:
		return r.services, r.err
	case <-time.After(driftProbeTimeout):
		return nil, fmt.Errorf("timed out after %s", driftProbeTimeout)
	}
}

func servicesEqual(a, b parser.Service) bool {
	if a.Name != b.Name || a.LocalAddr != b.LocalAddr || a.VPSPort != b.VPSPort {
		return false
	}
	da, db := slices.Clone(a.Domains), slices.Clone(b.Domains)
	slices.Sort(da)
	slices.Sort(db)
	return slices.Equal(da, db)
}
