package sysproxy

import (
	"os/exec"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestSystemProxyLifecycle(t *testing.T) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "windows" {
		t.Skip("Proxy tests only supported on macOS and Windows")
	}

	// 1. Enable proxy
	err := SetSystemProxy("127.0.0.1", 8080)
	if err != nil {
		t.Fatalf("SetSystemProxy failed: %v", err)
	}

	time.Sleep(500 * time.Millisecond)

	// 2. Disable proxy cleanly
	err = ClearSystemProxy()
	if err != nil {
		t.Fatalf("ClearSystemProxy failed: %v", err)
	}

	time.Sleep(500 * time.Millisecond)

	// Verify on macOS that Wi-Fi proxy is disabled
	if runtime.GOOS == "darwin" {
		out, err := exec.Command("networksetup", "-getwebproxy", "Wi-Fi").Output()
		if err == nil {
			if strings.Contains(string(out), "Enabled: Yes") {
				t.Fatalf("Wi-Fi webproxy is still enabled after ClearSystemProxy: %s", string(out))
			}
		}
	}
}
