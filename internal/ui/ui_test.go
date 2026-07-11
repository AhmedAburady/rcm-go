package ui

import (
	"strings"
	"testing"

	"github.com/AhmedAburady/rcm-go/internal/parser"
)

func TestRenderServicesByDomain(t *testing.T) {
	services := []parser.Service{
		{Name: "zeta", LocalAddr: "10.0.0.2:2", VPSPort: 5002, Domains: []string{"z.example.com", "shared.example.com"}},
		{Name: "alpha", LocalAddr: "10.0.0.1:1", VPSPort: 5001, Domains: []string{"a.example.com", "shared.example.com"}},
		{Name: "bare", LocalAddr: "10.0.0.3:3", VPSPort: 5003},
	}

	domains, groups := groupServicesByDomain(services)
	if len(domains) != 2 || len(groups["example.com"]) != 2 {
		t.Fatalf("unexpected grouping: domains=%v example.com=%v", domains, groups["example.com"])
	}
	if groups["example.com"][0].Name != "alpha" || groups["example.com"][1].Name != "zeta" {
		t.Fatalf("services not sorted or deduplicated: %v", groups["example.com"])
	}

	got, groupCount := RenderServicesByDomain(services)
	if groupCount != 2 {
		t.Fatalf("group count = %d, want 2", groupCount)
	}
	headings := []string{"(no domain)", "example.com"}
	last := -1
	for _, heading := range headings {
		pos := strings.Index(got, heading)
		if pos < 0 {
			t.Fatalf("missing domain heading %q in:\n%s", heading, got)
		}
		if pos <= last {
			t.Fatalf("domain heading %q is out of order in:\n%s", heading, got)
		}
		last = pos
	}
	if strings.Count(got, "\nexample.com\n") != 1 {
		t.Fatalf("root domain should have one heading in:\n%s", got)
	}
	for _, hostname := range []string{"a.example.com", "shared.example.com", "z.example.com"} {
		if !strings.Contains(got, hostname) {
			t.Fatalf("grouped tables must preserve hostname %q in:\n%s", hostname, got)
		}
	}
}

func TestRenderServicesSyncByDomain(t *testing.T) {
	rows := []ServiceRow{{
		Service: parser.Service{Name: "app", LocalAddr: "10.0.0.1:1", VPSPort: 5001, Domains: []string{"app.at3ch.com", "www.at3ch.com"}},
		Local:   SyncOK, Remote: SyncDrift,
	}}

	domains, groups := groupByDomain(rows, func(row ServiceRow) string { return row.Service.Name }, func(row ServiceRow) []string { return row.Service.Domains })
	if len(domains) != 1 || len(groups["at3ch.com"]) != 1 {
		t.Fatalf("sync rows not deduplicated: domains=%v group=%v", domains, groups["at3ch.com"])
	}

	got, groupCount := RenderServicesSyncByDomain(rows)
	if groupCount != 1 {
		t.Fatalf("group count = %d, want 1", groupCount)
	}
	for _, want := range []string{"at3ch.com", "LOCAL", "REMOTE", "✓", "✗"} {
		if !strings.Contains(got, want) {
			t.Fatalf("missing %q in:\n%s", want, got)
		}
	}
}

func TestRootDomain(t *testing.T) {
	tests := map[string]string{
		"affine.at3ch.com":        "at3ch.com",
		"at3ch.com":               "at3ch.com",
		"https://www.aburady.com": "aburady.com",
		"https://*.example.com":   "example.com",
		"*.example.com:8443":      "example.com",
		"www.books.amazon.co.uk":  "amazon.co.uk",
		"app.github.io":           "app.github.io",
	}
	for input, want := range tests {
		if got := rootDomain(input); got != want {
			t.Errorf("rootDomain(%q) = %q, want %q", input, got, want)
		}
	}
}
