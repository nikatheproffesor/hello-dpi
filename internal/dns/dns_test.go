package dns

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
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

func TestGSBCaptiveDomains(t *testing.T) {
	gsbDomains := []string{
		"gsb.gov.tr",
		"wifi.gsb.gov.tr",
		"portal.gsb.gov.tr",
		"giris.gsb.gov.tr",
		"auth.gsb.gov.tr",
		"kyk.gov.tr",
		"wifi.kyk.gov.tr",
		"captive.apple.com",
		"connectivitycheck.gstatic.com",
		"msftconnecttest.com",
	}

	for _, d := range gsbDomains {
		if !IsLocalOrCaptiveDomain(d) {
			t.Errorf("Expected domain %q to be identified as local/captive domain", d)
		}
	}

	nonCaptive := []string{
		"discord.com",
		"roblox.com",
		"youtube.com",
		"google.com",
	}
	for _, d := range nonCaptive {
		if IsLocalOrCaptiveDomain(d) {
			t.Errorf("Domain %q should NOT be identified as captive domain", d)
		}
	}
}

func TestAntiPoisonCheck(t *testing.T) {
	poisoned := []string{
		"195.175.254.2",
		"195.175.254.10",
		"0.0.0.0",
		"127.0.0.1",
		"::",
	}
	for _, ip := range poisoned {
		if !IsPoisonedIP(ip) {
			t.Errorf("Expected %q to be detected as poisoned", ip)
		}
	}

	clean := []string{
		"1.1.1.1",
		"8.8.8.8",
		"162.159.138.232",
		"142.250.185.206",
	}
	for _, ip := range clean {
		if IsPoisonedIP(ip) {
			t.Errorf("Expected clean IP %q to NOT be poisoned", ip)
		}
	}
}

func TestMockDoHFailover(t *testing.T) {
	// Server 1 returns 500 error (simulating blocked DoH endpoint on GSB WiFi)
	s1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer s1.Close()

	// Server 2 returns valid JSON answer
	s2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/dns-json")
		fmt.Fprintf(w, `{"Status":0,"Answer":[{"name":"example.com","type":1,"TTL":300,"data":"93.184.216.34"}]}`)
	}))
	defer s2.Close()

	r := &Resolver{
		Endpoints:  []string{s1.URL, s2.URL},
		httpClient: s1.Client(),
		enabled:    true,
	}

	ip, err := r.Resolve(context.Background(), "example.com")
	if err != nil {
		t.Fatalf("Expected failover to succeed, got error: %v", err)
	}
	if ip != "93.184.216.34" {
		t.Fatalf("Expected 93.184.216.34, got %s", ip)
	}
}

