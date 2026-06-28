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

func TestParseNestedDirectives(t *testing.T) {
	// Site blocks with nested directives (tls/dns/transport) must not let those
	// inner `{` lines be mistaken for the domain block.
	caddyfile := `
# cloud: 192.168.1.245:5212
cloud.at3ch.com {
    tls force_automate {
        dns cloudflare {env.CF_API_TOKEN}
    }
    reverse_proxy 127.0.0.1:5013 {
        header_up Host {host}
        transport http {
            versions 1.1
        }
    }
}
`

	services, err := ParseContent(caddyfile)
	if err != nil {
		t.Fatalf("ParseContent failed: %v", err)
	}
	if len(services) != 1 {
		t.Fatalf("Expected 1 service, got %d", len(services))
	}
	s := services[0]
	if s.Name != "cloud" {
		t.Errorf("Expected name 'cloud', got '%s'", s.Name)
	}
	if s.VPSPort != 5013 {
		t.Errorf("Expected VPS port 5013, got %d", s.VPSPort)
	}
	if len(s.Domains) != 1 || s.Domains[0] != "cloud.at3ch.com" {
		t.Errorf("Expected domain 'cloud.at3ch.com', got %v", s.Domains)
	}
}

func TestParseStrayBraceInCommentDoesNotDropServices(t *testing.T) {
	// A '}' inside a comment must not desync the brace counter and suppress
	// every service defined after it.
	caddyfile := `
# a: 10.0.0.1:80
a.com {
    reverse_proxy localhost:5000
    # remember to close the } block
}

# b: 10.0.0.2:80
b.com {
    reverse_proxy localhost:5001
}
`
	services, err := ParseContent(caddyfile)
	if err != nil {
		t.Fatalf("ParseContent failed: %v", err)
	}
	if len(services) != 2 {
		t.Fatalf("Expected 2 services, got %d: %+v", len(services), services)
	}
}

func TestParseEscapedQuoteInBlockDoesNotDropServices(t *testing.T) {
	// An escaped quote inside a quoted directive value must not end the string
	// early and let a following '}' be counted as real syntax.
	caddyfile := `
# a: 10.0.0.1:80
a.com {
    respond "literal brace \" } still quoted"
    reverse_proxy localhost:5000
}

# b: 10.0.0.2:80
b.com {
    reverse_proxy localhost:5001
}
`
	services, err := ParseContent(caddyfile)
	if err != nil {
		t.Fatalf("ParseContent failed: %v", err)
	}
	if len(services) != 2 {
		t.Fatalf("Expected 2 services, got %d: %+v", len(services), services)
	}
}

func TestParseSiteAddressWithPort(t *testing.T) {
	caddyfile := `
# c: 10.0.0.3:80
c.com:8443 {
    reverse_proxy localhost:5002
}
`
	services, err := ParseContent(caddyfile)
	if err != nil {
		t.Fatalf("ParseContent failed: %v", err)
	}
	if len(services) != 1 {
		t.Fatalf("Expected 1 service, got %d", len(services))
	}
	if len(services[0].Domains) != 1 || services[0].Domains[0] != "c.com:8443" {
		t.Errorf("Expected domain 'c.com:8443', got %v", services[0].Domains)
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
