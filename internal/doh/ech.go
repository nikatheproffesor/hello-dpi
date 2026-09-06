package doh

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// ECHInfo stores Encrypted Client Hello configuration details
type ECHInfo struct {
	Domain     string
	Supported  bool
	Config     string
	ExpiresAt  time.Time
}

// ECHCache manages thread-safe caching of ECH records
type ECHCache struct {
	mu    sync.RWMutex
	cache map[string]ECHInfo
}

var globalECHCache = &ECHCache{
	cache: make(map[string]ECHInfo),
}

// CheckECH queries DNS Type 65 (HTTPS Resource Record) via DoH to discover Encrypted Client Hello parameters
func (r *Resolver) CheckECH(ctx context.Context, host string) (bool, string) {
	cleanHost := strings.ToLower(strings.TrimSuffix(host, "."))
	if cleanHost == "" || isLocalOrCaptiveDomain(cleanHost) {
		return false, ""
	}

	globalECHCache.mu.RLock()
	if info, found := globalECHCache.cache[cleanHost]; found {
		if time.Now().Before(info.ExpiresAt) {
			globalECHCache.mu.RUnlock()
			return info.Supported, info.Config
		}
	}
	globalECHCache.mu.RUnlock()

	// Query HTTPS RR (Type 65) from Cloudflare DoH
	endpoint := string(Cloudflare)
	u, err := url.Parse(endpoint)
	if err != nil {
		return false, ""
	}

	q := u.Query()
	q.Set("name", cleanHost)
	q.Set("type", "HTTPS") // DNS type 65
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, "GET", u.String(), nil)
	if err != nil {
		return false, ""
	}
	req.Header.Set("Accept", "application/dns-json")

	resp, err := r.httpClient.Do(req)
	if err != nil {
		return false, ""
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return false, ""
	}

	var dohResp dohJSONResponse
	if err := json.NewDecoder(resp.Body).Decode(&dohResp); err != nil {
		return false, ""
	}

	supported := false
	config := ""

	for _, ans := range dohResp.Answer {
		if ans.Type == 65 && strings.Contains(ans.Data, "ech=") {
			supported = true
			parts := strings.Split(ans.Data, " ")
			for _, p := range parts {
				if strings.HasPrefix(p, "ech=") {
					config = strings.TrimPrefix(p, "ech=")
					break
				}
			}
			break
		}
	}

	globalECHCache.mu.Lock()
	globalECHCache.cache[cleanHost] = ECHInfo{
		Domain:    cleanHost,
		Supported: supported,
		Config:    config,
		ExpiresAt: time.Now().Add(10 * time.Minute),
	}
	globalECHCache.mu.Unlock()

	return supported, config
}
