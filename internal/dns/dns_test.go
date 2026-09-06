package dns

import (
	"context"
	"testing"
)

func TestResolverDirectIP(t *testing.T) {
	r := NewResolver("", true)
	ip, err := r.Resolve(context.Background(), "1.1.1.1")
	if err != nil {
		t.Fatalf("Resolve direct IP failed: %v", err)
	}
	if ip != "1.1.1.1" {
		t.Fatalf("Expected 1.1.1.1, got %s", ip)
	}
}

func TestResolverCaptiveDomain(t *testing.T) {
	r := NewResolver("", true)
	// Local domain should bypass DoH directly to system DNS
	ip, _ := r.Resolve(context.Background(), "localhost")
	if ip == "" {
		t.Fatalf("Expected non-empty resolution for localhost")
	}
}

func TestAntiPoisonCheck(t *testing.T) {
	if !IsPoisonedIP("195.175.254.2") {
		t.Errorf("Expected 195.175.254.2 to be detected as poisoned")
	}
	if IsPoisonedIP("1.1.1.1") {
		t.Errorf("Expected 1.1.1.1 to NOT be poisoned")
	}
}
