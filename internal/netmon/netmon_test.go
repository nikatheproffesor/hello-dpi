package netmon

import (
	"testing"
	"time"
)

func TestMonitor_Inspect(t *testing.T) {
	m := NewMonitor("127.0.0.1", 8080, nil)
	st := m.inspect()
	if st.Timestamp.IsZero() {
		t.Errorf("Expected non-zero timestamp")
	}
}

func TestMonitor_StartStop(t *testing.T) {
	var notified bool
	m := NewMonitor("127.0.0.1", 8080, func(oldState, newState NetworkState) {
		notified = true
	})
	m.pollInterval = 50 * time.Millisecond
	m.Start()
	time.Sleep(120 * time.Millisecond)
	m.Stop()

	// Verify no panic on stop
	_ = notified
}
