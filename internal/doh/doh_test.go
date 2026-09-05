package doh

import (
	"context"
	"testing"
)

func TestIsLocalOrCaptiveDomain(t *testing.T) {
	cases := []struct {
		domain   string
		expected bool
	}{
		{"localhost", true},
		{"myrouter.local", true},
		{"wifi.gsb.gov.tr", true},
		{"portal.kyk.gov.tr", true},
		{"captive.apple.com", true},
		{"connectivitycheck.gstatic.com", true},
		{"msftconnecttest.com", true},
		{"discord.com", false},
		{"google.com", false},
		{"w2g.tv", false},
	}

	for _, tc := range cases {
		actual := isLocalOrCaptiveDomain(tc.domain)
		if actual != tc.expected {
			t.Errorf("isLocalOrCaptiveDomain(%q) = %v, expected %v", tc.domain, actual, tc.expected)
		}
	}
}

func TestResolveIPDirect(t *testing.T) {
	r := NewResolver("", true)
	ip, err := r.Resolve(context.Background(), "1.1.1.1")
	if err != nil || ip != "1.1.1.1" {
		t.Fatalf("expected 1.1.1.1, got %v (err: %v)", ip, err)
	}
}
