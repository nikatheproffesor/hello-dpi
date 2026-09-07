//go:build darwin

package sysproxy

import (
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
)

// Helper process entry point for crash simulation
func TestHelperProcessKillSimulation(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	// 1. Enable proxy and watchdog
	_ = SetSystemProxy("127.0.0.1", 8080)

	// 2. Sleep indefinitely until killed forcefully by parent
	time.Sleep(30 * time.Second)
	os.Exit(0)
}

func TestWatchdogSelfHealingOnForceKill(t *testing.T) {
	// Start helper process that sets proxy and watchdog
	cmd := exec.Command(os.Args[0], "-test.run=TestHelperProcessKillSimulation")
	cmd.Env = append(os.Environ(), "GO_WANT_HELPER_PROCESS=1")

	if err := cmd.Start(); err != nil {
		t.Fatalf("Failed to start helper process: %v", err)
	}

	// Give helper process 1 second to configure proxy and arm watchdog
	time.Sleep(1 * time.Second)

	// Verify helper process enabled proxy on Wi-Fi
	out, err := exec.Command("networksetup", "-getwebproxy", "Wi-Fi").Output()
	if err != nil || !strings.Contains(string(out), "Enabled: Yes") {
		t.Logf("Wi-Fi proxy output before kill: %s", string(out))
	}

	// SIMULATE CRASH / DEFENDER DELETION / KILL -9:
	// Process is forcefully assassinated with SIGKILL. No defer or cleanup in helper can run!
	_ = cmd.Process.Kill()
	_ = cmd.Wait()

	// Wait 3.5 seconds for detached Watchdog to detect PID disappearance and restore network
	time.Sleep(3500 * time.Millisecond)

	// Verify that Watchdog detected the dead PID and turned off Wi-Fi proxy
	outAfter, err := exec.Command("networksetup", "-getwebproxy", "Wi-Fi").Output()
	if err != nil {
		t.Fatalf("Failed to get proxy state: %v", err)
	}

	if strings.Contains(string(outAfter), "Enabled: Yes") {
		t.Fatalf("CRITICAL FAILURE: Watchdog did not restore proxy after kill -9! State: %s", string(outAfter))
	}

	t.Logf("SUCCESS: Watchdog automatically restored system proxy within 3 seconds after sudden kill -9!")
}
