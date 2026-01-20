package generator

import (
	"strings"
	"testing"

	"github.com/ahmabora1/rcm/internal/config"
	"github.com/ahmabora1/rcm/internal/parser"
)

func TestGenerateServerTOML(t *testing.T) {
	cfg := &config.Config{
		Rathole: config.RatholeConfig{
			BindPort:         2333,
			Token:            "test-token",
			ServerPrivateKey: "test-private-key",
		},
	}

	services := []parser.Service{
		{Name: "web", LocalAddr: "192.168.1.10:8080", VPSPort: 8001},
		{Name: "api", LocalAddr: "192.168.1.10:3000", VPSPort: 8002},
	}

	output, err := GenerateServerTOML(cfg, services)
	if err != nil {
		t.Fatalf("GenerateServerTOML failed: %v", err)
	}

	// Check expected content
	if !strings.Contains(output, "bind_addr = \"0.0.0.0:2333\"") {
		t.Error("Missing bind_addr")
	}
	if !strings.Contains(output, "default_token = \"test-token\"") {
		t.Error("Missing default_token")
	}
	if !strings.Contains(output, "[server.services.web]") {
		t.Error("Missing web service")
	}
	if !strings.Contains(output, "[server.services.api]") {
		t.Error("Missing api service")
	}
}

func TestGenerateClientTOML(t *testing.T) {
	cfg := &config.Config{
		Server: config.ServerConfig{
			Host: "vps.example.com",
		},
		Rathole: config.RatholeConfig{
			BindPort:        2333,
			Token:           "test-token",
			ServerPublicKey: "test-public-key",
		},
	}

	services := []parser.Service{
		{Name: "web", LocalAddr: "192.168.1.10:8080", VPSPort: 8001},
	}

	output, err := GenerateClientTOML(cfg, services)
	if err != nil {
		t.Fatalf("GenerateClientTOML failed: %v", err)
	}

	// Check expected content
	if !strings.Contains(output, "remote_addr = \"vps.example.com:2333\"") {
		t.Error("Missing remote_addr")
	}
	if !strings.Contains(output, "[client.services.web]") {
		t.Error("Missing web service")
	}
	if !strings.Contains(output, "local_addr = \"192.168.1.10:8080\"") {
		t.Error("Missing local_addr")
	}
}
