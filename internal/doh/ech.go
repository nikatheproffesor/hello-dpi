package doh

import (
	"context"

	"github.com/hellodpi/hellodpi/internal/dns"
)

// ECHInfo stores Encrypted Client Hello configuration details
type ECHInfo = dns.ECHInfo

// CheckECH queries DNS Type 65 (HTTPS Resource Record) via DoH to discover Encrypted Client Hello parameters
func (r *Resolver) CheckECH(ctx context.Context, host string) (bool, string) {
	return r.inner.CheckECH(ctx, host)
}

