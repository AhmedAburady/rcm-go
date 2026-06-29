package cmd

import (
	"testing"

	"github.com/AhmedAburady/rcm-go/internal/parser"
	"github.com/AhmedAburady/rcm-go/internal/ui"
)

func TestServicesEqual(t *testing.T) {
	base := parser.Service{Name: "gitea", LocalAddr: "192.168.1.11:3000", VPSPort: 5001, Domains: []string{"git.example.com"}}

	tests := []struct {
		name string
		b    parser.Service
		want bool
	}{
		{"identical", base, true},
		{"different addr", parser.Service{Name: "gitea", LocalAddr: "192.168.1.99:3000", VPSPort: 5001, Domains: []string{"git.example.com"}}, false},
		{"different port", parser.Service{Name: "gitea", LocalAddr: "192.168.1.11:3000", VPSPort: 5009, Domains: []string{"git.example.com"}}, false},
		{"different domain", parser.Service{Name: "gitea", LocalAddr: "192.168.1.11:3000", VPSPort: 5001, Domains: []string{"other.example.com"}}, false},
		{"extra domain", parser.Service{Name: "gitea", LocalAddr: "192.168.1.11:3000", VPSPort: 5001, Domains: []string{"git.example.com", "x.example.com"}}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := servicesEqual(base, tt.b); got != tt.want {
				t.Errorf("servicesEqual = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRemoteSync(t *testing.T) {
	local := parser.Service{Name: "gitea", LocalAddr: "192.168.1.11:3000", VPSPort: 5001}
	same := local
	diff := parser.Service{Name: "gitea", LocalAddr: "192.168.1.11:9999", VPSPort: 5001}

	tests := []struct {
		name      string
		reachable bool
		hasLocal  bool
		hasRemote bool
		remote    parser.Service
		want      ui.Sync
	}{
		{"unreachable", false, true, true, same, ui.SyncUnknown},
		{"missing on remote", true, true, false, parser.Service{}, ui.SyncDrift},
		{"differs on remote", true, true, true, diff, ui.SyncDrift},
		{"matches remote", true, true, true, same, ui.SyncOK},
		{"remote only", true, false, true, same, ui.SyncDrift},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := remoteSync(tt.reachable, tt.hasLocal, tt.hasRemote, local, tt.remote); got != tt.want {
				t.Errorf("remoteSync = %v, want %v", got, tt.want)
			}
		})
	}
}
