package doctor

import (
	"context"
	"testing"
)

func TestMeasureLatency(t *testing.T) {
	ctx := context.Background()
	rtt := MeasureLatency(ctx, "1.1.1.1:443", true)
	// Should return non-negative latency or timeout (-1 if offline)
	t.Logf("1.1.1.1:443 RTT: %d ms", rtt)
}

func TestRunLivePingBenchmark(t *testing.T) {
	report := RunLivePingBenchmark()
	if len(report.Targets) == 0 {
		t.Fatalf("expected at least 1 ping target, got 0")
	}
	if report.Advantage == "" {
		t.Errorf("expected advantage description, got empty")
	}
}
