package proxy

import (
	"fmt"
	"net"
	"net/url"
	"strings"
)

// ValidateProxyURL validates a proxy URL to prevent SSRF attacks.
// Only allows http, https, and socks5 schemes targeting non-private addresses.
func ValidateProxyURL(rawURL string) error {
	if rawURL == "" {
		return nil
	}

	u, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("invalid proxy URL: %w", err)
	}

	// Scheme whitelist.
	switch strings.ToLower(u.Scheme) {
	case "http", "https", "socks5", "socks5h":
		// OK
	default:
		return fmt.Errorf("unsupported proxy scheme: %s", u.Scheme)
	}

	host := u.Hostname()
	if host == "" {
		return fmt.Errorf("proxy URL missing host")
	}

	// Resolve hostname to IPs and check each.
	ips, err := net.LookupHost(host)
	if err != nil {
		// If DNS resolution fails, check the literal hostname.
		if ip := net.ParseIP(host); ip != nil {
			if isBlockedIP(ip) {
				return fmt.Errorf("proxy address %s is not allowed", host)
			}
		}
		// Allow unresolvable hostnames (could be accessible via custom DNS).
		return nil
	}

	for _, ipStr := range ips {
		ip := net.ParseIP(ipStr)
		if ip != nil && isBlockedIP(ip) {
			return fmt.Errorf("proxy address %s resolves to blocked IP %s", host, ipStr)
		}
	}

	return nil
}

// isBlockedIP returns true for loopback, private, and link-local addresses.
func isBlockedIP(ip net.IP) bool {
	return ip.IsLoopback() ||
		ip.IsPrivate() ||
		ip.IsLinkLocalUnicast() ||
		ip.IsLinkLocalMulticast() ||
		isMetadataIP(ip)
}

// isMetadataIP checks for cloud metadata service IPs (169.254.169.254).
func isMetadataIP(ip net.IP) bool {
	return ip.Equal(net.ParseIP("169.254.169.254"))
}
