package parser

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
)

// Service represents a parsed service from Caddyfile.
type Service struct {
	Name      string   // Service name from comment
	LocalAddr string   // Local address (e.g., 192.168.1.100:8080)
	VPSPort   int      // Port on VPS (from reverse_proxy)
	Domains   []string // Domain names
}

var (
	// Pattern: # service_name: local_addr
	serviceCommentRe = regexp.MustCompile(`^#\s*(\w[\w-]*\w|\w):\s*(.+)$`)

	// Pattern: domain.com, domain2.com { — allows scheme prefixes and explicit
	// ports/paths in the site address (e.g. http://example.com, example.com:8443).
	domainBlockRe = regexp.MustCompile(`^([a-zA-Z0-9.,\s\-_:/*]+?)\s*\{`)

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

		if line == "" {
			continue
		}

		// Only match annotations and domain blocks at the top level, so nested
		// directives like `tls force_automate {` aren't mistaken for a new block.
		if braceCount == 0 {
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
				currentDomains = parseDomains(matches[1])
				braceCount += netBraces(line)
				continue
			}
		}

		braceCount += netBraces(line)

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

// netBraces returns the net brace depth change for a line ({ as +1, } as -1),
// ignoring braces inside double-quoted strings and after a `#` comment so a
// stray brace in a comment or string literal can't desync the block counter.
// Backslash escapes inside a quoted string are honored, so an escaped quote
// (`\"`) does not prematurely end the string.
func netBraces(line string) int {
	n := 0
	inQuote := false
	for i := 0; i < len(line); i++ {
		c := line[i]
		if inQuote {
			switch c {
			case '\\':
				i++ // skip the escaped character (e.g. \" or \\)
			case '"':
				inQuote = false
			}
			continue
		}
		switch c {
		case '"':
			inQuote = true
		case '#':
			return n
		case '{':
			n++
		case '}':
			n--
		}
	}
	return n
}

func parseDomains(s string) []string {
	var domains []string
	for d := range strings.SplitSeq(s, ",") {
		d = strings.TrimSpace(d)
		if d != "" {
			domains = append(domains, d)
		}
	}
	return domains
}
