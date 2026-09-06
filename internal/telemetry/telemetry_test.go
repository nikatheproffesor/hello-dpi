package telemetry

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/hellodpi/hellodpi/internal/probe"
)

func TestCollector_OptInDefaults(t *testing.T) {
	tmpConfig := filepath.Join(t.TempDir(), "telemetry.json")
	c := NewCollectorWithConfig(tmpConfig)
	if c.IsEnabled() {
		t.Errorf("Telemetry MUST be disabled by default for privacy")
	}

	c.SetEnabled(true)
	if !c.IsEnabled() {
		t.Errorf("Expected enabled after SetEnabled(true)")
	}

	c.SetEnabled(false)
	if c.IsEnabled() {
		t.Errorf("Expected disabled after SetEnabled(false)")
	}
}

func TestCollector_RecordProbe(t *testing.T) {
	tmpConfig := filepath.Join(t.TempDir(), "telemetry.json")
	c := NewCollectorWithConfig(tmpConfig)
	c.SetEnabled(true)

	sample := &probe.Result{
		Timestamp:      time.Now(),
		ISPName:        "Test Telecom Fiber",
		BestStrategy:   "out-of-order",
		BypassVerified: true,
		ISPFingerprint: probe.ISPFingerprint{
			RTTBand: "0-20ms",
		},
	}

	c.RecordProbe(sample)

	tbl := c.GetISPSuccessTable()
	stats, ok := tbl["Test Telecom Fiber"]
	if !ok {
		t.Fatalf("Expected stats for 'Test Telecom Fiber'")
	}
	if stats.TotalProbes != 1 {
		t.Errorf("Expected 1 probe, got %d", stats.TotalProbes)
	}
	if stats.SuccessCount != 1 {
		t.Errorf("Expected 1 success, got %d", stats.SuccessCount)
	}
	if stats.TopStrategy != "out-of-order" {
		t.Errorf("Expected top strategy 'out-of-order', got %s", stats.TopStrategy)
	}
}
