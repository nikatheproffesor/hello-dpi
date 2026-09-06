package hellocore

import (
	"testing"
)

func TestMobileEngine_Lifecycle(t *testing.T) {
	eng := NewEngine(Config{
		ListenAddr: "127.0.0.1:18080",
		SplitMode:  "auto",
		DelayMs:    3,
	})

	if eng.IsRunning() {
		t.Error("Engine should not be running before Start()")
	}

	err := eng.Start()
	if err != nil {
		t.Fatalf("Start() failed: %v", err)
	}

	if !eng.IsRunning() {
		t.Error("Engine should be running after Start()")
	}

	// Verify rule evaluation
	action := eng.EvaluateHost("ziraatbank.com.tr")
	if action != 1 { // ActionDirect == 1
		t.Errorf("Expected ActionDirect (1) for ziraatbank.com.tr, got %d", action)
	}

	actionDPI := eng.EvaluateHost("discord.com")
	if actionDPI != 2 { // ActionProxyDPI == 2
		t.Errorf("Expected ActionProxyDPI (2) for discord.com, got %d", actionDPI)
	}

	err = eng.Stop()
	if err != nil {
		t.Errorf("Stop() failed: %v", err)
	}

	if eng.IsRunning() {
		t.Error("Engine should not be running after Stop()")
	}
}
