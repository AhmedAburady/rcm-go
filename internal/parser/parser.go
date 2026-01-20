package parser

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
)

var (
	// Pattern: # service_name: local_addr
	serviceCommentRe = regexp.MustCompile(`^#\s*(\w[\w-]*\w|\w):\s*(.+)$`)

	// Pattern: domain.com, domain2.com {
	domainBlockRe = regexp.MustCompile(`^([a-zA-Z0-9.,\s\-_]+)\s*\{`)

	// Pattern: reverse_proxy [http://]localhost|127.0.0.1:PORT
	reverseProxyRe = regexp.MustCompile(`reverse_proxy\s+(?:https?://)?(?:localhost|127\.0\.0\.1):(\d+)`)
)

// ParseFile parses a Caddyfile and extracts services
func ParseFile(path string) ([]Service, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open caddyfile: %w", err)
	}
	defer file.Close()

	return Parse(bufio.NewScanner(file))
}

// ParseContent parses Caddyfile content from string
func ParseContent(content string) ([]Service, error) {
	return Parse(bufio.NewScanner(strings.NewReader(content)))
}

// Parse parses Caddyfile from a scanner
func Parse(scanner *bufio.Scanner) ([]Service, error) {
	var services []Service
	serviceMap := make(map[string]*Service)

	var pendingService *struct {
		name      string
		localAddr string
	}
	var currentDomains []string
	braceCount := 0

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines
		if line == "" {
			continue
		}

		// Check for service comment
		if matches := serviceCommentRe.FindStringSubmatch(line); matches != nil {
			pendingService = &struct {
				name      string
				localAddr string
			}{
				name:      matches[1],
				localAddr: strings.TrimSpace(matches[2]),
			}
			continue
		}

		// Check for domain block start
		if matches := domainBlockRe.FindStringSubmatch(line); matches != nil {
			domainStr := matches[1]
			currentDomains = parseDomains(domainStr)
			braceCount++
			continue
		}

		// Track braces
		braceCount += strings.Count(line, "{") - strings.Count(line, "}")

		// Check for reverse_proxy inside a block
		if braceCount > 0 && pendingService != nil {
			if matches := reverseProxyRe.FindStringSubmatch(line); matches != nil {
				port, _ := strconv.Atoi(matches[1])

				// Check if service already exists (multiple domains)
				if existing, ok := serviceMap[pendingService.name]; ok {
					existing.Domains = append(existing.Domains, currentDomains...)
				} else {
					svc := &Service{
						Name:      pendingService.name,
						LocalAddr: pendingService.localAddr,
						VPSPort:   port,
						Domains:   currentDomains,
					}
					serviceMap[pendingService.name] = svc
					services = append(services, *svc)
				}
			}
		}

		// Reset when block closes
		if braceCount == 0 {
			pendingService = nil
			currentDomains = nil
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan error: %w", err)
	}

	// Update services slice with any domain additions from serviceMap
	for i := range services {
		if updated, ok := serviceMap[services[i].Name]; ok {
			services[i] = *updated
		}
	}

	return services, nil
}

func parseDomains(s string) []string {
	var domains []string
	for _, d := range strings.Split(s, ",") {
		d = strings.TrimSpace(d)
		if d != "" {
			domains = append(domains, d)
		}
	}
	return domains
}
