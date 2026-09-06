package wintun

import (
	"testing"
)

func TestWintun_Lifecycle(t *testing.T) {
	// Ensure Stop() is safe to call
	if err := Stop(); err != nil {
		t.Errorf("Stop() failed: %v", err)
	}
	_ = IsAvailable()
	_ = IsRunning()
}
