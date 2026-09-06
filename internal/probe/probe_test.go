package probe

import (
	"os"
	"path/filepath"
	"testing"
	"time"
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
	if res.BestStrategy == "" {
		t.Error("Expected non-empty BestStrategy")
	}
	if res.BestDelayMs <= 0 {
		t.Error("Expected positive delay")
	}
}

func TestProbeEngine_Persistence(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "hellodpi_probe_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	cacheFile := filepath.Join(tmpDir, "tuning.json")

	eng := NewEngine()
	eng.cachePath = cacheFile

	sample := &Result{
		Timestamp:      time.Now(),
		ISPName:        "Test Fiber ISP",
		BestStrategy:   "sni",
		BestMode:       "sni",
		BestSplitPos:   5,
		BestDelayMs:    3,
		BestLatencyMs:  15,
		BypassVerified: true,
	}

	eng.savePersisted(sample)

	// Create new engine pointing to same cache file
	eng2 := &Engine{cachePath: cacheFile}
	eng2.loadPersisted()

	loaded := eng2.GetLastResult()
	if loaded.ISPName != "Test Fiber ISP" {
		t.Errorf("Expected ISP 'Test Fiber ISP', got '%s'", loaded.ISPName)
	}
	if loaded.BestStrategy != "sni" {
		t.Errorf("Expected BestStrategy 'sni', got '%s'", loaded.BestStrategy)
	}
	if loaded.BestLatencyMs != 15 {
		t.Errorf("Expected latency 15ms, got %d", loaded.BestLatencyMs)
	}
}
