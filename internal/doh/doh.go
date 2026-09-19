package doh

import (
	"context"

	"github.com/hellodpi/hellodpi/internal/dns"
)

// Provider represents a DoH endpoint
type Provider = dns.Provider

const (
	Cloudflare    Provider = dns.Cloudflare
	CloudflareAlt Provider = dns.CloudflareAlt
	AdGuard       Provider = dns.AdGuard
	AdGuardAlt    Provider = dns.AdGuardAlt
	ControlD      Provider = dns.ControlD
	Google        Provider = dns.Google
	GoogleAlt     Provider = dns.GoogleAlt
	Quad9         Provider = dns.Quad9
	Quad9Alt      Provider = dns.Quad9Alt
	DNSSB         Provider = dns.DNSSB
)

// Resolver provides DNS-over-HTTPS resolution with caching and multi-tier fallback
type Resolver struct {
	inner *dns.Resolver
}

// NewResolver initializes a DNS-over-HTTPS resolver with multi-tier failover
func NewResolver(primaryEndpoint string, enabled bool) *Resolver {
	return &Resolver{
		inner: dns.NewResolver(primaryEndpoint, enabled),
	}
}

// Resolve returns the IP address for the given hostname with resilient fallback
func (r *Resolver) Resolve(ctx context.Context, host string) (string, error) {
	return r.inner.Resolve(ctx, host)
}

// IsLocalOrCaptiveDomain checks if a domain is an internal, local, or captive portal domain
func IsLocalOrCaptiveDomain(host string) bool {
	return dns.IsLocalOrCaptiveDomain(host)
}

func isLocalOrCaptiveDomain(host string) bool {
	return dns.IsLocalOrCaptiveDomain(host)
}


