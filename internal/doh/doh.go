package doh

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
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

// Resolver provides DNS-over-HTTPS resolution with caching
type Resolver struct {
	Endpoint   string
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
		Type int    `json:"type"` // 1 = A, 28 = AAAA
		TTL  int    `json:"TTL"`
		Data string `json:"data"`
	} `json:"Answer"`
}

// NewResolver initializes a DNS-over-HTTPS resolver
func NewResolver(endpoint string, enabled bool) *Resolver {
	if endpoint == "" {
		endpoint = string(Cloudflare)
	}
	return &Resolver{
		Endpoint: endpoint,
		enabled:  enabled,
		httpClient: &http.Client{
			Timeout: 4 * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:        100,
				IdleConnTimeout:     90 * time.Second,
				DisableCompression: true,
			},
		},
	}
}

// Resolve returns the IP address for the given hostname
func (r *Resolver) Resolve(ctx context.Context, host string) (string, error) {
	if !r.enabled {
		// Use standard OS DNS
		ips, err := net.DefaultResolver.LookupHost(ctx, host)
		if err != nil || len(ips) == 0 {
			return host, err
		}
		return ips[0], nil
	}

	// Check if host is already an IP address
	if net.ParseIP(host) != nil {
		return host, nil
	}

	// Check cache
	if val, ok := r.cache.Load(host); ok {
		entry := val.(cacheEntry)
		if time.Now().Before(entry.expiresAt) {
			return entry.ip, nil
		}
		r.cache.Delete(host)
	}

	// Query DoH
	reqURL := fmt.Sprintf("%s?name=%s&type=A", r.Endpoint, url.QueryEscape(host))
	req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
	if err != nil {
		return host, err
	}
	req.Header.Set("Accept", "application/dns-json")

	resp, err := r.httpClient.Do(req)
	if err != nil {
		// Fallback to default resolver on error
		ips, fbErr := net.DefaultResolver.LookupHost(ctx, host)
		if fbErr == nil && len(ips) > 0 {
			return ips[0], nil
		}
		return host, err
	}
	defer resp.Body.Close()

	var dohResp dohJSONResponse
	if err := json.NewDecoder(resp.Body).Decode(&dohResp); err != nil {
		return host, err
	}

	for _, ans := range dohResp.Answer {
		if ans.Type == 1 && net.ParseIP(ans.Data) != nil {
			ttl := ans.TTL
			if ttl < 60 {
				ttl = 60
			}
			r.cache.Store(host, cacheEntry{
				ip:        ans.Data,
				expiresAt: time.Now().Add(time.Duration(ttl) * time.Second),
			})
			return ans.Data, nil
		}
	}

	return host, fmt.Errorf("no A record found for %s via DoH", host)
}
