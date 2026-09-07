//go:build darwin

package sysproxy

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"
)

var watchdogCmd *exec.Cmd

func startWatchdog(host string, port int) {
	stopWatchdog()

	pid := os.Getpid()
	// Detached shell script running independently in its own process group.
	// Monitors PID with `kill -0 "$PID"`. The exact second PID exits (kill -9, Defender, crash, Trash),
	// it checks if proxy is still enabled for 127.0.0.1 and immediately turns it off on all network services.
	script := fmt.Sprintf(`
PID=%d
PORT=%d

while kill -0 "$PID" 2>/dev/null; do
    sleep 2
done

# Parent process is dead. Inspect network services and restore if still set to 127.0.0.1
for s in $(networksetup -listallnetworkservices 2>/dev/null | grep -v "\*"); do
    if networksetup -getwebproxy "$s" 2>/dev/null | grep -q "127.0.0.1"; then
        networksetup -setwebproxystate "$s" off 2>/dev/null
        networksetup -setsecurewebproxystate "$s" off 2>/dev/null
        networksetup -setsocksfirewallproxystate "$s" off 2>/dev/null
    fi
done

for ev in http_proxy https_proxy all_proxy HTTP_PROXY HTTPS_PROXY ALL_PROXY; do
    launchctl unsetenv "$ev" 2>/dev/null
done
`, pid, port)

	cmd := exec.Command("sh", "-c", script)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setsid: true, // Creates a new session so watchdog survives when parent process dies
	}

	if err := cmd.Start(); err == nil {
		watchdogCmd = cmd
	}
}

func stopWatchdog() {
	if watchdogCmd != nil && watchdogCmd.Process != nil {
		_ = watchdogCmd.Process.Kill()
		watchdogCmd = nil
	}
}
