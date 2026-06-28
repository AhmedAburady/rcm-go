package parser

// Service represents a parsed service from Caddyfile
type Service struct {
	Name      string   // Service name from comment
	LocalAddr string   // Local address (e.g., 192.168.1.100:8080)
	VPSPort   int      // Port on VPS (from reverse_proxy)
	Domains   []string // Domain names
}

// PrimaryDomain returns the first domain or empty string
func (s *Service) PrimaryDomain() string {
	if len(s.Domains) > 0 {
		return s.Domains[0]
	}
	return ""
}
