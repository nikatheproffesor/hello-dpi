package doh

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// Provider represents a DoH endpoint
type Provider string

const (
	Cloudflare Provider = "https://1.1.1.1/dns-query"
	Google     Provider = "https://8.8.8.8/resolve"
	Quad9      Provider = "https://9.9.9.9/dns-query"
)

// Resolver provides DNS-over-HTTPS resolution with caching and multi-tier fallback
type Resolver struct {
	Endpoints  []string
	httpClient *http.Client
	cache      sync.Map // domain -> cacheEntry
	enabled    bool
}

type cacheEntry struct {
	ip        string
	expiresAt time.Time
}

type dohJSONResponse struct {
	Status int `json:"Status"`
	Answer []struct {
		Name string `json:"name"`
		Type int    `json:"type"` // 1 = A, 5 = CNAME, 28 = AAAA
		TTL  int    `json:"TTL"`
		Data string `json:"data"`
	} `json:"Answer"`
}

// NewResolver initializes a DNS-over-HTTPS resolver with multi-tier failover
func NewResolver(primaryEndpoint string, enabled bool) *Resolver {
	endpoints := []string{
		string(Cloudflare),
		string(Google),
		string(Quad9),
	}
	if primaryEndpoint != "" && primaryEndpoint != string(Cloudflare) {
		// Place custom primary endpoint first
		endpoints = append([]string{primaryEndpoint}, endpoints...)
	}

	return &Resolver{
		Endpoints: endpoints,
		enabled:   enabled,
		httpClient: &http.Client{
			Timeout: 3 * time.Second, // Responsive timeout so failover is fast
			Transport: &http.Transport{
				MaxIdleConns:        100,
				IdleConnTimeout:     90 * time.Second,
				DisableCompression:  true,
				TLSHandshakeTimeout: 2 * time.Second,
			},
		},
	}
}

// isLocalOrCaptiveDomain checks if a domain is an internal, local, or captive portal domain
func isLocalOrCaptiveDomain(host string) bool {
	h := strings.ToLower(strings.TrimSuffix(host, "."))
	if h == "localhost" || strings.HasSuffix(h, ".local") || strings.HasSuffix(h, ".lan") || strings.HasSuffix(h, ".home") {
		return true
	}
	// GSB WiFi (KYK) & captive portal detection domains
	if strings.Contains(h, "gsb.gov.tr") ||
		strings.Contains(h, "kyk.gov.tr") ||
		h == "captive.apple.com" ||
		h == "connectivitycheck.gstatic.com" ||
		h == "connectivitycheck.android.com" ||
		h == "msftconnecttest.com" ||
		h == "ipv6.msftconnecttest.com" ||
		strings.Contains(h, "routerlogin") ||
		strings.Contains(h, "modem") {
		return true
	}
	return false
}

// Resolve returns the IP address for the given hostname with resilient fallback
func (r *Resolver) Resolve(ctx context.Context, host string) (string, error) {
	// 1. Direct return if host is already an IP address
	if net.ParseIP(host) != nil {
		return host, nil
	}

	// 2. If disabled, or if this is a captive portal / local domain, use OS DNS directly
	if !r.enabled || isLocalOrCaptiveDomain(host) {
		ips, err := net.DefaultResolver.LookupHost(ctx, host)
		if err == nil && len(ips) > 0 {
			return ips[0], nil
		}
		return host, err
	}

	// 3. Check memory cache
	if val, ok := r.cache.Load(host); ok {
		entry := val.(cacheEntry)
		if time.Now().Before(entry.expiresAt) {
			return entry.ip, nil
		}
		r.cache.Delete(host)
	}

	// 4. Try DoH endpoints in sequence (Cloudflare -> Google -> Quad9)
	for _, endpoint := range r.Endpoints {
		ip, ttl, err := r.queryDoHEndpoint(ctx, endpoint, host)
		if err == nil && ip != "" {
			if ttl < 60 {
				ttl = 60
			}
			r.cache.Store(host, cacheEntry{
				ip:        ip,
				expiresAt: time.Now().Add(time.Duration(ttl) * time.Second),
			})
			return ip, nil
		}
	}

	// 5. Ultimate Fallback: System DNS (Vital for captive portal / restricted dorm networks)
	ips, err := net.DefaultResolver.LookupHost(ctx, host)
	if err == nil && len(ips) > 0 {
		return ips[0], nil
	}

	return host, fmt.Errorf("all DoH endpoints and system DNS failed for %s", host)
}

func (r *Resolver) queryDoHEndpoint(ctx context.Context, endpoint, host string) (string, int, error) {
	reqURL := fmt.Sprintf("%s?name=%s&type=A", endpoint, url.QueryEscape(host))
	req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
	if err != nil {
		return "", 0, err
	}
	req.Header.Set("Accept", "application/dns-json")

	resp, err := r.httpClient.Do(req)
	if err != nil {
		return "", 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", 0, fmt.Errorf("status %d", resp.StatusCode)
	}

	var dohResp dohJSONResponse
	if err := json.NewDecoder(resp.Body).Decode(&dohResp); err != nil {
		return "", 0, err
	}

	// 1. Look for A record (IPv4)
	for _, ans := range dohResp.Answer {
		if ans.Type == 1 && net.ParseIP(ans.Data) != nil {
			return ans.Data, ans.TTL, nil
		}
	}

	// 2. Look for AAAA record (IPv6) if no IPv4
	for _, ans := range dohResp.Answer {
		if ans.Type == 28 && net.ParseIP(ans.Data) != nil {
			return ans.Data, ans.TTL, nil
		}
	}

	return "", 0, fmt.Errorf("no A or AAAA record")
}
