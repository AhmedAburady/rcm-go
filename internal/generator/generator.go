package generator

import (
	"bytes"
	"embed"
	"fmt"
	"text/template"

	"github.com/AhmedAburady/rcm-go/internal/config"
	"github.com/AhmedAburady/rcm-go/internal/parser"
)

//go:embed templates/*.tmpl
var templateFS embed.FS

// GenerateServerTOML generates server.toml content
func GenerateServerTOML(cfg *config.Config, services []parser.Service) (string, error) {
	return executeTemplate("templates/server.toml.tmpl", map[string]interface{}{
		"Rathole":  cfg.Rathole,
		"Services": services,
	})
}

// GenerateClientTOML generates client.toml content
func GenerateClientTOML(cfg *config.Config, services []parser.Service) (string, error) {
	return executeTemplate("templates/client.toml.tmpl", map[string]interface{}{
		"Server":   cfg.Server,
		"Rathole":  cfg.Rathole,
		"Services": services,
	})
}

func executeTemplate(name string, data interface{}) (string, error) {
	tmplContent, err := templateFS.ReadFile(name)
	if err != nil {
		return "", fmt.Errorf("read template %s: %w", name, err)
	}

	tmpl, err := template.New(name).Parse(string(tmplContent))
	if err != nil {
		return "", fmt.Errorf("parse template %s: %w", name, err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("execute template %s: %w", name, err)
	}

	return buf.String(), nil
}
