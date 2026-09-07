package sysproxy

import (
	"sync"
)

var watchdogMu sync.Mutex

// StartWatchdog launches an independent, detached guardian process
// that monitors Hello DPI's PID. If Hello DPI is forcefully terminated,
// quarantined by antivirus, crashed, or uninstalled while the proxy is on,
// the watchdog automatically restores all system network settings within 2 seconds.
func StartWatchdog(host string, port int) {
	watchdogMu.Lock()
	defer watchdogMu.Unlock()
	startWatchdog(host, port)
}

// StopWatchdog cleanly shuts down the guardian process on normal exit.
func StopWatchdog() {
	watchdogMu.Lock()
	defer watchdogMu.Unlock()
	stopWatchdog()
}
