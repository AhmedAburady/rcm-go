package parser

import (
	"testing"
)

func TestParseContent(t *testing.T) {
	caddyfile := `
# plex: 192.168.1.100:32400
plex.example.com {
    reverse_proxy localhost:8001
}

# homeassistant: 192.168.1.100:8123
ha.example.com, home.example.com {
    reverse_proxy localhost:8002
}

# nextcloud: 192.168.1.100:8080
cloud.example.com {
    reverse_proxy localhost:8003
}
`

	services, err := ParseContent(caddyfile)
	if err != nil {
		t.Fatalf("ParseContent failed: %v", err)
	}

	if len(services) != 3 {
		t.Errorf("Expected 3 services, got %d", len(services))
	}

	// Check plex service
	if services[0].Name != "plex" {
		t.Errorf("Expected service name 'plex', got '%s'", services[0].Name)
	}
	if services[0].LocalAddr != "192.168.1.100:32400" {
		t.Errorf("Expected local addr '192.168.1.100:32400', got '%s'", services[0].LocalAddr)
	}
	if services[0].VPSPort != 8001 {
		t.Errorf("Expected VPS port 8001, got %d", services[0].VPSPort)
	}
	if len(services[0].Domains) != 1 || services[0].Domains[0] != "plex.example.com" {
		t.Errorf("Expected domain 'plex.example.com', got %v", services[0].Domains)
	}

	// Check homeassistant service (multiple domains)
	if services[1].Name != "homeassistant" {
		t.Errorf("Expected service name 'homeassistant', got '%s'", services[1].Name)
	}
	if len(services[1].Domains) != 2 {
		t.Errorf("Expected 2 domains, got %d", len(services[1].Domains))
	}
}

func TestPrimaryDomain(t *testing.T) {
	s := Service{
		Name:    "test",
		Domains: []string{"first.example.com", "second.example.com"},
	}

	if s.PrimaryDomain() != "first.example.com" {
		t.Errorf("Expected 'first.example.com', got '%s'", s.PrimaryDomain())
	}

	s2 := Service{Name: "empty"}
	if s2.PrimaryDomain() != "" {
		t.Errorf("Expected empty string, got '%s'", s2.PrimaryDomain())
	}
}
