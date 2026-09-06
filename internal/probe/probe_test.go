package probe

import (
	"testing"
)

func TestBuildClientHello(t *testing.T) {
	ch := BuildClientHello("discord.com")
	if len(ch) < 50 {
		t.Fatalf("ClientHello too short: %d bytes", len(ch))
	}

	// Verify TLS record header
	if ch[0] != 0x16 {
		t.Errorf("Expected TLS record type 0x16, got 0x%02x", ch[0])
	}
	if ch[1] != 0x03 || ch[2] != 0x01 {
		t.Errorf("Expected TLS 1.0 record version 0x0301, got 0x%02x%02x", ch[1], ch[2])
	}

	// Verify Handshake type ClientHello (0x01) at offset 5
	if ch[5] != 0x01 {
		t.Errorf("Expected Handshake type 0x01 (ClientHello), got 0x%02x", ch[5])
	}
}

func TestProbeEngine_Fallback(t *testing.T) {
	eng := NewEngine()
	res := eng.GetLastResult()
	if res == nil {
		t.Fatal("Expected non-nil fallback probe result")
	}
	if res.BestMode == "" {
		t.Error("Expected non-empty BestMode")
	}
	if res.BestDelayMs <= 0 {
		t.Error("Expected positive delay")
	}
}
