package doctor

import (
	"testing"
)

func TestRunRepair(t *testing.T) {
	report := RunRepair()
	if report == nil {
		t.Fatalf("expected report, got nil")
	}

	if len(report.Steps) < 3 {
		t.Errorf("expected at least 3 diagnostic steps, got %d", len(report.Steps))
	}

	if report.Timestamp == "" {
		t.Errorf("expected timestamp, got empty")
	}
}
