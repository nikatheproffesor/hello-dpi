package dns

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
	Cloudflare    Provider = "https://1.1.1.1/dns-query"
	CloudflareAlt Provider = "https://162.159.36.1/dns-query"
	AdGuard       Provider = "https://94.140.14.14/dns-query"
	AdGuardAlt    Provider = "https://94.140.15.15/dns-query"
	ControlD      Provider = "https://76.76.2.0/dns-query"
	Google        Provider = "https://8.8.8.8/resolve"
	GoogleAlt     Provider = "https://8.8.4.4/resolve"
	Quad9         Provider = "https://9.9.9.9/dns-query"
	Quad9Alt      Provider = "https://149.112.112.112/dns-query"
	DNSSB         Provider = "https://185.222.222.222/dns-query"
)

// Turkish ISP Poisoned IP prefix (BTK court-order block redirect)
const PoisonedBTKPrefix = "195.175.254."

// IsPoisonedIP returns true if the IP matches known ISP censorship redirection blocks
func IsPoisonedIP(ip string) bool {
	trimmed := strings.TrimSpace(ip)
	return strings.HasPrefix(trimmed, PoisonedBTKPrefix) ||
		trimmed == "0.0.0.0" ||
		trimmed == "127.0.0.1" ||
		trimmed == "::"
}

// Resolver provides DNS-over-HTTPS resolution with caching, anti-poisoning, and multi-tier fallback
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

// DefaultDoHEndpoints returns the resilient pool of DoH endpoints optimized for Turkish ISP / GSB WiFi resilience
func DefaultDoHEndpoints() []string {
	return []string{
		string(Cloudflare),
		string(AdGuard),       // Unblocked on KYK / GSB WiFi
		string(CloudflareAlt), // Unblocked anycast IP
		string(ControlD),      // Unblocked
		string(Google),
		string(AdGuardAlt),
		string(GoogleAlt),
		string(Quad9Alt),
		string(Quad9),
		string(DNSSB),
	}
}

// NewResolver initializes a DNS-over-HTTPS resolver with multi-tier failover
func NewResolver(primaryEndpoint string, enabled bool) *Resolver {
	endpoints := DefaultDoHEndpoints()
	if primaryEndpoint != "" && primaryEndpoint != string(Cloudflare) {
		// Prepend custom primary endpoint
		endpoints = append([]string{primaryEndpoint}, endpoints...)
	}

	return &Resolver{
		Endpoints: endpoints,
		enabled:   enabled,
		httpClient: &http.Client{
			Timeout: 2 * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:        100,
				IdleConnTimeout:     90 * time.Second,
				DisableCompression:  true,
				TLSHandshakeTimeout: 1500 * time.Millisecond,
			},
		},
	}
}

// IsLocalOrCaptiveDomain checks if a domain is an internal, local, or captive portal domain
func IsLocalOrCaptiveDomain(host string) bool {
	h := strings.ToLower(strings.TrimSuffix(host, "."))
	if colon := strings.IndexByte(h, ':'); colon != -1 {
		h = h[:colon]
	}
	if h == "localhost" || strings.HasSuffix(h, ".local") || strings.HasSuffix(h, ".lan") || strings.HasSuffix(h, ".home") || strings.HasSuffix(h, ".internal") {
		return true
	}
	// GSB WiFi (KYK) & captive portal detection domains (exact or proper subdomain suffix)
	if h == "gsb.gov.tr" || strings.HasSuffix(h, ".gsb.gov.tr") ||
		h == "kyk.gov.tr" || strings.HasSuffix(h, ".kyk.gov.tr") ||
		h == "captive.apple.com" ||
		h == "connectivitycheck.gstatic.com" ||
		h == "connectivitycheck.android.com" ||
		h == "msftconnecttest.com" ||
		h == "ipv6.msftconnecttest.com" ||
		h == "routerlogin.net" || strings.HasSuffix(h, ".routerlogin.net") ||
		h == "routerlogin.com" || strings.HasSuffix(h, ".routerlogin.com") ||
		h == "modem.local" || h == "modem.lan" {
		return true
	}
	return false
}

func isLocalOrCaptiveDomain(host string) bool {
	return IsLocalOrCaptiveDomain(host)
}


type resolveResult struct {
	ip  string
	ttl int
	err error
}

// Resolve returns the clean IP address for the given hostname with resilient multi-tier fallback
func (r *Resolver) Resolve(ctx context.Context, host string) (string, error) {
	// 1. Direct return if host is already an IP address
	cleanHost := host
	if colon := strings.IndexByte(cleanHost, ':'); colon != -1 {
		cleanHost = cleanHost[:colon]
	}
	cleanHost = strings.Trim(cleanHost, "[]")

	if net.ParseIP(cleanHost) != nil {
		return cleanHost, nil
	}

	// 2. If disabled, or if this is a captive portal / local domain, use OS DNS directly
	if !r.enabled || isLocalOrCaptiveDomain(cleanHost) {
		ips, err := net.DefaultResolver.LookupHost(ctx, cleanHost)
		if err == nil && len(ips) > 0 {
			return ips[0], nil
		}
		return cleanHost, err
	}

	// 3. Check memory cache
	if val, ok := r.cache.Load(cleanHost); ok {
		entry := val.(cacheEntry)
		if time.Now().Before(entry.expiresAt) {
			return entry.ip, nil
		}
		r.cache.Delete(cleanHost)
	}

	// 4. Concurrent Hedged DoH Queries
	// To defeat GSB WiFi firewall blocking of specific IPs (1.1.1.1 / 8.8.8.8) without stalling,
	// query the top endpoints competitively in parallel. The fastest clean response wins.
	resultCh := make(chan resolveResult, len(r.Endpoints))
	dohCtx, dohCancel := context.WithTimeout(ctx, 2500*time.Millisecond)
	defer dohCancel()

	// Select top diverse candidates (Cloudflare, AdGuard, CloudflareAlt, ControlD, Google)
	candidates := r.Endpoints
	if len(candidates) > 5 {
		candidates = candidates[:5]
	}

	var activeQueries sync.WaitGroup
	for _, ep := range candidates {
		activeQueries.Add(1)
		go func(endpoint string) {
			defer activeQueries.Done()
			ip, ttl, err := r.queryDoHEndpoint(dohCtx, endpoint, cleanHost)
			if err == nil && ip != "" && !IsPoisonedIP(ip) {
				select {
				case resultCh <- resolveResult{ip: ip, ttl: ttl}:
				default:
				}
			}
		}(ep)
	}

	// Wait for first winning response or completion of all
	doneCh := make(chan struct{})
	go func() {
		activeQueries.Wait()
		close(doneCh)
	}()

	select {
	case win := <-resultCh:
		if win.ttl < 60 {
			win.ttl = 60
		}
		r.cache.Store(cleanHost, cacheEntry{
			ip:        win.ip,
			expiresAt: time.Now().Add(time.Duration(win.ttl) * time.Second),
		})
		return win.ip, nil
	case <-doneCh:
		// Check if any result arrived just before wait completed
		select {
		case win := <-resultCh:
			if win.ttl < 60 {
				win.ttl = 60
			}
			r.cache.Store(cleanHost, cacheEntry{
				ip:        win.ip,
				expiresAt: time.Now().Add(time.Duration(win.ttl) * time.Second),
			})
			return win.ip, nil
		default:
		}
	case <-dohCtx.Done():
	}

	// 5. Ultimate Fallback: System DNS (strictly validating against ISP DNS poisoning)
	ips, err := net.DefaultResolver.LookupHost(ctx, cleanHost)
	if err == nil && len(ips) > 0 {
		for _, ip := range ips {
			if !IsPoisonedIP(ip) {
				return ip, nil
			}
		}
	}

	return cleanHost, fmt.Errorf("all DoH endpoints and unpoisoned system DNS failed for %s", cleanHost)
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
		return "", 0, fmt.Errorf("doh status %d", resp.StatusCode)
	}

	var dohResp dohJSONResponse
	if err := json.NewDecoder(resp.Body).Decode(&dohResp); err != nil {
		return "", 0, err
	}

	if dohResp.Status != 0 || len(dohResp.Answer) == 0 {
		return "", 0, fmt.Errorf("no answer returned")
	}

	// Prioritize IPv4 (Type A = 1) then IPv6 (Type AAAA = 28), verifying valid IP format and anti-poisoning
	for _, ans := range dohResp.Answer {
		if ans.Type == 1 {
			trimmed := strings.TrimSpace(ans.Data)
			if ip := net.ParseIP(trimmed); ip != nil && !IsPoisonedIP(trimmed) {
				return ip.String(), ans.TTL, nil
			}
		}
	}

	for _, ans := range dohResp.Answer {
		if ans.Type == 28 {
			trimmed := strings.TrimSpace(ans.Data)
			if ip := net.ParseIP(trimmed); ip != nil && !IsPoisonedIP(trimmed) {
				return ip.String(), ans.TTL, nil
			}
		}
	}

	return "", 0, fmt.Errorf("no valid unpoisoned IP address found in DoH answer for %s", host)
}

