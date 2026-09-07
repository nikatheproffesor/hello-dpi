package ech

import (
	"testing"
)

func TestECHConfigRegistration(t *testing.T) {
	mgr := NewManager()

	cfg, ok := mgr.GetECHConfig("discord.com")
	if !ok || cfg == nil {
		t.Fatalf("expected preset ECH config for discord.com")
	}
	if cfg.PublicName != "cloudflare.com" {
		t.Errorf("expected public name cloudflare.com, got %s", cfg.PublicName)
	}

	custom := &ECHConfig{
		Version:    0xfe0d,
		ConfigID:   0x02,
		PublicName: "custom.cdn.net",
	}
	mgr.RegisterECHConfig("example.org", custom)

	c, ok := mgr.GetECHConfig("example.org")
	if !ok || c.PublicName != "custom.cdn.net" {
		t.Errorf("failed to retrieve custom ECH config")
	}
}

func TestEncapsulateOuterClientHello(t *testing.T) {
	mgr := NewManager()
	cfg, ok := mgr.GetECHConfig("discord.com")
	if !ok {
		t.Fatalf("missing config")
	}

	innerHello := []byte("synthetic-inner-client-hello-data-discord.com")
	outerHello, err := EncapsulateOuterClientHello(innerHello, cfg, "cloudflare.com")
	if err != nil {
		t.Fatalf("EncapsulateOuterClientHello failed: %v", err)
	}

	if len(outerHello) < 50 {
		t.Fatalf("outerHello too short: %d bytes", len(outerHello))
	}

	// Must have record header 0x16 (Handshake)
	if outerHello[0] != 0x16 {
		t.Errorf("expected outerHello to start with 0x16, got %x", outerHello[0])
	}

	// Must detect ECH extension in outer ClientHello
	if !HasECHExtension(outerHello) {
		t.Errorf("outer ClientHello must contain ECH extension 0xfe0d")
	}
}
