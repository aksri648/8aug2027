package api

import (
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

func ValidateURLForSSRF(rawURL string) (*url.URL, error) {
	if rawURL == "" {
		return nil, fmt.Errorf("URL cannot be empty")
	}

	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("invalid URL format: %w", err)
	}

	scheme := strings.ToLower(u.Scheme)
	if scheme != "http" && scheme != "https" {
		return nil, fmt.Errorf("unsupported scheme: %s (only http/https allowed)", scheme)
	}

	hostname := u.Hostname()
	if hostname == "" {
		return nil, fmt.Errorf("invalid hostname in URL")
	}

	// Direct IP check
	if ip := net.ParseIP(hostname); ip != nil {
		if isForbiddenIP(ip) {
			return nil, fmt.Errorf("access to internal/private IP %s is prohibited", ip.String())
		}
		return u, nil
	}

	// Check for forbidden hostname strings
	lowerHost := strings.ToLower(hostname)
	if lowerHost == "localhost" || strings.HasSuffix(lowerHost, ".local") || strings.HasSuffix(lowerHost, ".internal") {
		return nil, fmt.Errorf("access to local/internal host %s is prohibited", hostname)
	}

	// Perform DNS resolution
	ips, err := net.LookupIP(hostname)
	if err == nil {
		for _, ip := range ips {
			if isForbiddenIP(ip) {
				return nil, fmt.Errorf("access to internal/private IP %s is prohibited", ip.String())
			}
		}
	}

	return u, nil
}

func isForbiddenIP(ip net.IP) bool {
	if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsMulticast() || ip.IsUnspecified() {
		return true
	}

	// Additional cloud metadata / private IPv4 checks
	ip4 := ip.To4()
	if ip4 != nil {
		// 169.254.169.254 cloud metadata IP
		if ip4[0] == 169 && ip4[1] == 254 {
			return true
		}
		// 127.0.0.0/8 loopback
		if ip4[0] == 127 {
			return true
		}
		// 0.0.0.0/8
		if ip4[0] == 0 {
			return true
		}
	}

	return false
}

func NewSSRFProtectedClient(timeout time.Duration) *http.Client {
	return &http.Client{
		Timeout: timeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 5 {
				return fmt.Errorf("too many redirects")
			}
			_, err := ValidateURLForSSRF(req.URL.String())
			if err != nil {
				return fmt.Errorf("redirect blocked by SSRF protection: %w", err)
			}
			return nil
		},
	}
}
